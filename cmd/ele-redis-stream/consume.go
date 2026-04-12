package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/stream"
	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/spf13/cobra"
)

const (
	chanBufferSize = 4096
	forwardTimeout = 5 * time.Second
	defaultBlockMs = 1000 // shorter block = faster ctx check on shutdown
)

var (
	configPath        string
	blockMs           int
	consumeStreamName string
	consumePayload    string
)

func init() {
	rootCmd.AddCommand(consumeCmd)
	consumeCmd.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "config file path (for Redis, Log)")
	consumeCmd.Flags().IntVarP(&blockMs, "block", "b", defaultBlockMs, "XREAD block timeout in ms (shorter = faster shutdown, more round-trips when idle)")
	consumeCmd.Flags().StringVarP(&consumeStreamName, "stream", "s", StreamRoomEvents, "Redis stream key (e.g. room_events, lobby_events)")
	consumeCmd.Flags().StringVar(&consumePayload, "payload", "auto", "protobuf body: auto (try Message then Event), message (pubsub.Message), event (raw pubsub.Event, as lobby StreamPublisher writes)")
}

var consumeCmd = &cobra.Command{
	Use:   "consume",
	Short: "Start Redis Stream consumer",
	Long: `Read a Redis stream (default room_events), decode protobuf, print JSON.

Examples:
  ele-redis-stream consume -c config.yaml
  ele-redis-stream consume -c config.yaml -s lobby_events
  ele-redis-stream consume -c config.yaml -s lobby_events --payload event

Lobby publishes raw pubsub.Event to lobby_events; room/produce often uses wrapped pubsub.Message on room_events.
--payload auto tries Message first, then Event.`,
	RunE: runConsume,
}

type bridgeMsg struct {
	topic   string
	entryID string
	redisTs int64
	msg     *pb.Message
	ev      *pb.Event
}

// bridgeConfig contains Redis and Log only, reuses API Server config.yaml structure
type bridgeConfig struct {
	LogCfg   log.Config   `mapstructure:"log"`
	RedisCfg redis.Config `mapstructure:"redis"`
}

func runConsume(cmd *cobra.Command, args []string) error {
	cfg := &bridgeConfig{}
	if err := config.InitConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.LogCfg.Level == "" {
		cfg.LogCfg.Level = "info"
	}
	if cfg.RedisCfg.Address == "" {
		cfg.RedisCfg.Address = "localhost:6379"
	}
	if cfg.RedisCfg.Size == 0 {
		cfg.RedisCfg.Size = 10
	}

	if err := log.InitGlobalLogger(&cfg.LogCfg); err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}

	if err := redis.Init(&cfg.RedisCfg); err != nil {
		return fmt.Errorf("failed to init Redis: %w", err)
	}

	st, err := stream.NewRedisStream()
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan bridgeMsg, chanBufferSize)

	// Consumer: receive from channel and process async (simulate forward, output to console)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-ch:
				if !ok {
					return
				}
				handleMsg(m)
			}
		}
	}()

	// Producer: stream Read loop
	go func() {
		log.Infof("Redis stream consumer started, stream=%s payload=%s block=%dms", consumeStreamName, consumePayload, blockMs)
		defer log.Infof("Redis stream consumer stopped")
		defer close(ch)

		block := blockMs
		if block < 100 {
			block = 100
		}

		lastID := "$"
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			entries, err := st.Read(ctx, consumeStreamName, lastID, block)
			if err != nil {
				log.Errorf("stream read error: %v", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			for _, e := range entries {
				lastID = e.ID
				if len(e.Payload) == 0 {
					continue
				}

				wrap, bare, derr := decodeStreamPayload(e.Payload, consumePayload)
				if derr != nil {
					log.Warnw("skip stream entry: decode failed", "stream", consumeStreamName, "id", e.ID, "topic_field", e.Topic, "err", derr)
					continue
				}

				select {
				case ch <- bridgeMsg{topic: e.Topic, entryID: e.ID, redisTs: e.Timestamp, msg: wrap, ev: bare}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return waitForSignal(cancel)
}

// handleMsg simulates forwardToClient: async with timeout protection
func handleMsg(m bridgeMsg) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		headerTime := time.Now().Format("15:04:05.000")
		opts := protojson.MarshalOptions{Multiline: true, Indent: "  "}

		if m.ev != nil {
			ev := m.ev
			line := fmt.Sprintf("[%s] entry=%s redis_ts=%d topic_field=%s payload=pubsub.Event type=%s event_message_id=%s",
				headerTime, m.entryID, m.redisTs, m.topic, ev.Type.String(), ev.MessageId)
			fmt.Println(line)
			log.Infof("%s", line)
			if b, err := opts.Marshal(ev); err == nil {
				fmt.Printf("  event json:\n%s\n", string(b))
			} else {
				fmt.Printf("  (protojson error: %v)\n", err)
			}
			if out := ev.GetTournamentMatchOutcome(); out != nil {
				fmt.Println("  summary:", summarizeTournamentMatchOutcome(out))
			}
			fmt.Println("---")
			return
		}

		if m.msg != nil {
			eventType := "unknown"
			if m.msg.Event != nil {
				eventType = m.msg.Event.Type.String()
			}
			line := fmt.Sprintf("[%s] entry=%s redis_ts=%d topic_field=%s payload=pubsub.Message event=%s msgId=%s publisher=%s ts=%d",
				headerTime, m.entryID, m.redisTs, m.topic, eventType, m.msg.MessageId, m.msg.PublisherId, m.msg.Timestamp)
			fmt.Println(line)
			log.Infof("%s", line)
			if len(m.msg.Metadata) > 0 {
				fmt.Printf("  metadata: %v\n", m.msg.Metadata)
			}
			if b, err := opts.Marshal(m.msg); err == nil {
				fmt.Printf("  message json:\n%s\n", string(b))
			}
			if m.msg.Event != nil {
				if out := m.msg.Event.GetTournamentMatchOutcome(); out != nil {
					fmt.Println("  summary:", summarizeTournamentMatchOutcome(out))
				}
			}
			fmt.Println("---")
			return
		}

		fmt.Printf("[%s] entry=%s decode produced no Message or Event\n---\n", headerTime, m.entryID)
	}()
	select {
	case <-done:
	case <-time.After(forwardTimeout):
		log.Warnf("handle msg timeout: topic=%s entry=%s", m.topic, m.entryID)
	}
}

func waitForSignal(cancel context.CancelFunc) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Info("Received shutdown signal")
	cancel()
	return nil
}
