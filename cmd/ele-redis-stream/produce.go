package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/stream"
	"github.com/google/uuid"
	gproto "google.golang.org/protobuf/proto"

	pb "github.com/CryptoElementals/common/rpc/proto"
	"github.com/spf13/cobra"
)

var (
	produceTopic    string
	produceInterval time.Duration
	produceCount    int
)

func init() {
	rootCmd.AddCommand(produceCmd)
	produceCmd.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "config file path")
	produceCmd.Flags().StringVarP(&produceTopic, "topic", "t", "test_topic_0x123_0x456", "test topic")
	produceCmd.Flags().DurationVarP(&produceInterval, "interval", "i", time.Second, "send interval (0 = send once)")
	produceCmd.Flags().IntVarP(&produceCount, "count", "n", 0, "message count (0 = send until Ctrl+C)")
}

var produceCmd = &cobra.Command{
	Use:   "produce",
	Short: "Producer: write test events to room_events",
	Long: `Write test events to Redis Stream room_events. Format matches Room Server PubSub.Publish.
Use with consume for end-to-end testing.`,
	RunE: runProduce,
}

func runProduce(cmd *cobra.Command, args []string) error {
	cfg := &bridgeConfig{}
	if err := config.InitConfig(configPath, cfg); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.RedisCfg.Address == "" {
		cfg.RedisCfg.Address = "localhost:6379"
	}
	if cfg.RedisCfg.Size == 0 {
		cfg.RedisCfg.Size = 10
	}

	if err := redis.Init(&cfg.RedisCfg); err != nil {
		return fmt.Errorf("failed to init Redis: %w", err)
	}

	st, err := stream.NewRedisStream()
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ctx := context.Background()
	sent := 0
	for {
		select {
		case <-sigCh:
			fmt.Printf("Produced %d messages, exiting\n", sent)
			return nil
		default:
		}

		msg := &pb.Message{
			MessageId:   uuid.New().String(),
			Topic:       produceTopic,
			Timestamp:   time.Now().Unix(),
			PublisherId: "ele-redis-stream-produce",
			Event: &pb.Event{
				Type: pb.EventType_TYPE_GAME_PHASE_SYNC,
				Event: &pb.Event_GamePhase{
					GamePhase: &pb.GamePhase{
						GameType: pb.GameType_PVP,
						GameID:   1,
					},
				},
			},
		}

		b, err := gproto.Marshal(msg)
		if err != nil {
			return fmt.Errorf("marshal failed: %w", err)
		}

		_, err = st.Publish(ctx, defaultStreamName, msg.Topic, b, msg.Timestamp)
		if err != nil {
			return fmt.Errorf("publish failed: %w", err)
		}

		sent++
		fmt.Printf("[%s] produced #%d topic=%s msgId=%s\n", time.Now().Format("15:04:05.000"), sent, produceTopic, msg.MessageId)

		if produceCount > 0 && sent >= produceCount {
			return nil
		}

		if produceInterval > 0 {
			select {
			case <-sigCh:
				fmt.Printf("Produced %d messages, exiting\n", sent)
				return nil
			case <-time.After(produceInterval):
			}
		}
	}
}
