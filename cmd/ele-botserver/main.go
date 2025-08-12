package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	Execute()
}

var rootCmd = &cobra.Command{
	Use:   "bot-server",
	Short: "elemental battle bot server",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
