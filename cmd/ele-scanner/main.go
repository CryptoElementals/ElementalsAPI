package main

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	configPath string
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

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.yaml", "config file path")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
