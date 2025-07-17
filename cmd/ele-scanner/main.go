package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	Execute()
}

var rootCmd = &cobra.Command{
	Use:   "ele-scanner",
	Short: "Make block scan and analysis and notify the roomserver",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
