package main

import (
	"os"

	"github.com/spf13/cobra"
)

var configPath string

func main() {
	Execute()
}

var rootCmd = &cobra.Command{
	Use:   "ele-bsc-scanner",
	Short: "Scan BSC finalized blocks for TokenCollector deposit/withdraw events",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "bsc_scanner_config.yaml", "config file path")
}
