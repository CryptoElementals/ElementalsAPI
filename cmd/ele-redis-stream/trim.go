package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/stream"
	pb "github.com/CryptoElementals/common/rpc/proto"
	"google.golang.org/protobuf/encoding/protojson"
	gproto "google.golang.org/protobuf/proto"
	"github.com/spf13/cobra"
)

var (
	trimMaxAge time.Duration
	trimStream string
	trimDryRun bool
)

func init() {
	rootCmd.AddCommand(trimCmd)
	trimCmd.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "config file path")
	trimCmd.Flags().DurationVarP(&trimMaxAge, "max-age", "m", time.Hour, "remove entries older than this (e.g. 1h, 30m, 10s)")
	trimCmd.Flags().StringVarP(&trimStream, "stream", "s", StreamRoomEvents, "stream name")
	trimCmd.Flags().BoolVarP(&trimDryRun, "dry-run", "n", false, "show what would be trimmed without deleting")
}

// formatPayload formats protobuf bytes as indented JSON, or a fallback string on error
func formatPayload(payload []byte) string {
	if len(payload) == 0 {
		return "(empty)"
	}
	var msg pb.Message
	if err := gproto.Unmarshal(payload, &msg); err != nil {
		return fmt.Sprintf("(proto unmarshal error: %v)", err)
	}
	opts := protojson.MarshalOptions{Multiline: true, Indent: "    "}
	b, err := opts.Marshal(&msg)
	if err != nil {
		return fmt.Sprintf("(json marshal error: %v)", err)
	}
	return string(b)
}

var trimCmd = &cobra.Command{
	Use:   "trim",
	Short: "Trim stream: remove entries older than max-age",
	Long: `Remove entries from Redis Stream older than the specified max-age.
Uses XTRIM MINID - stream IDs are milliseconds-since-epoch, so we compute
the cutoff ID from (now - maxAge).

Example: trim -m 1h  removes entries older than 1 hour`,
	RunE: runTrim,
}

func runTrim(cmd *cobra.Command, args []string) error {
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

	ctx := context.Background()
	cutoff := time.Now().Add(-trimMaxAge)
	minID := fmt.Sprintf("%d-0", cutoff.UnixMilli())

	entries, err := st.Range(ctx, trimStream, "-", minID)
	if err != nil {
		return fmt.Errorf("range failed: %w", err)
	}

	var toDelete []stream.Entry
	for _, e := range entries {
		if e.ID < minID {
			toDelete = append(toDelete, e)
		}
	}

	if trimDryRun {
		lenBefore, err := st.Len(ctx, trimStream)
		if err != nil {
			return fmt.Errorf("len failed: %w", err)
		}
		fmt.Printf("Stream: %s\n", trimStream)
		fmt.Printf("Max age: %s (cutoff: %s)\n", trimMaxAge, cutoff.Format(time.RFC3339))
		fmt.Printf("MINID threshold: %s\n", minID)
		fmt.Printf("Entries before trim: %d (would delete %d)\n", lenBefore, len(toDelete))
		for _, e := range toDelete {
			fmt.Printf("  [%s] topic=%s ts=%d\n", e.ID, e.Topic, e.Timestamp)
			fmt.Printf("    payload:\n%s\n", formatPayload(e.Payload))
		}
		fmt.Println("Dry run - no changes made")
		return nil
	}

	for _, e := range toDelete {
		fmt.Printf("[%s] topic=%s ts=%d\n", e.ID, e.Topic, e.Timestamp)
		fmt.Printf("  payload:\n%s\n", formatPayload(e.Payload))
	}

	expected := len(toDelete)
	deleted, err := st.Trim(ctx, trimStream, trimMaxAge)
	if err != nil {
		return fmt.Errorf("trim failed: %w", err)
	}
	if expected > 0 && deleted < expected {
		fmt.Fprintf(os.Stderr, "ERROR: trim deleted %d entries, expected %d (stream=%s)\n", deleted, expected, trimStream)
	}

	fmt.Printf("Trimmed %d entries from %s (older than %s)\n", deleted, trimStream, trimMaxAge)
	return nil
}
