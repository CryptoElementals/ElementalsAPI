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
	gproto "google.golang.org/protobuf/proto"

	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/spf13/cobra"
)

const (
	defaultStreamName = "room_events"
	chanBufferSize    = 4096
	forwardTimeout    = 5 * time.Second
	defaultBlockMs    = 1000 // shorter block = faster ctx check on shutdown
)

var (
	configPath string
	blockMs    int
)

func init() {
	rootCmd.AddCommand(consumeCmd)
	consumeCmd.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "config file path (for Redis, Log)")
	consumeCmd.Flags().IntVarP(&blockMs, "block", "b", defaultBlockMs, "XREAD block timeout in ms (shorter = faster shutdown, more round-trips when idle)")
}

var consumeCmd = &cobra.Command{
	Use:   "consume",
	Short: "Start Redis Stream consumer",
	Long: `Consume events from room_events with optimized logic (channel decoupling, async, timeout protection),
output to console.`,
	RunE: runConsume,
}

type bridgeMsg struct {
	topic string
	msg   *pb.Message
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
		log.Infof("Redis stream consumer started, stream=%s, block=%dms", defaultStreamName, blockMs)
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

			entries, err := st.Read(ctx, defaultStreamName, lastID, block)
			if err != nil {
				log.Errorf("stream read error: %v", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			for _, e := range entries {
				lastID = e.ID
				if e.Topic == "" || len(e.Payload) == 0 {
					continue
				}

				var pbMsg pb.Message
				if err := gproto.Unmarshal(e.Payload, &pbMsg); err != nil {
					continue
				}

				select {
				case ch <- bridgeMsg{topic: e.Topic, msg: &pbMsg}:
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
	// Test service: output to stdout and log
	done := make(chan struct{})
	go func() {
		defer close(done)
		eventType := "unknown"
		if m.msg.Event != nil {
			eventType = m.msg.Event.Type.String()
		}
		line := fmt.Sprintf("[%s] topic=%s event=%s msgId=%s publisher=%s ts=%d",
			time.Now().Format("15:04:05.000"), m.topic, eventType, m.msg.MessageId, m.msg.PublisherId, m.msg.Timestamp)
		fmt.Println(line)
		log.Infof("%s", line)
		if len(m.msg.Metadata) > 0 {
			meta := fmt.Sprintf("  metadata: %v", m.msg.Metadata)
			fmt.Println(meta)
		}
		// Full message as JSON (indented for readability)
		opts := protojson.MarshalOptions{Multiline: true, Indent: "  "}
		if b, err := opts.Marshal(m.msg); err == nil {
			payload := fmt.Sprintf("  payload:\n%s", string(b))
			fmt.Println(payload)
		}
		fmt.Println("---")
	}()
	select {
	case <-done:
	case <-time.After(forwardTimeout):
		log.Warnf("handle msg timeout: topic=%s", m.topic)
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
