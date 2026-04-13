package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "ele-redis-stream",
	Short: "Redis Stream test tool",
	Long: `Redis Stream test tool with producer and consumer:

  consume  - Read a stream (default room_events); -s lobby_events for lobby. Decodes pubsub.Message or raw pubsub.Event, prints JSON (+ tournament summary when applicable).
  produce  - Producer: write test pubsub.Message frames to room_events
  trim     - Trim stream: remove entries older than max-age (e.g. 1h)

Redis config required. Use -c config.yaml to specify config file.`,
}
