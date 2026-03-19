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

  consume  - Consumer: consume events from room_events and output to console (optimized: channel decoupling, async, timeout)
  produce  - Producer: write test events to room_events
  trim     - Trim stream: remove entries older than max-age (e.g. 1h)

Redis config required. Use -c config.yaml to specify config file.`,
}
