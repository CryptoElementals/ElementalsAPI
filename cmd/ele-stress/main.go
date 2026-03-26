package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	Execute()
}

var rootCmd = &cobra.Command{
	Use:   "ele-stress",
	Short: "Elementals stress testing tool",
	Long: `Elementals stress testing tool manages multiple bots to stress test
the game server. It automatically generates wallets, logs in bots, and runs
game loops for each bot.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
