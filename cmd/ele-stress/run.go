package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/stress"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "start stress test",
	Run: func(cmd *cobra.Command, args []string) {
		runStressTest()
	},
}

var configPath string

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	runCmd.MarkFlagRequired("config")
}

func runStressTest() {
	err := config.InitStressConfig(configPath)
	if err != nil {
		panic("init config failed: " + err.Error())
	}
	err = log.InitGlobalLogger(&config.StressCfg.LogCfg)
	if err != nil {
		panic("init logger failed: " + err.Error())
	}

	// Convert config.StressConfig to stress.Config
	stressConfig := &stress.Config{
		BaseURL:   config.StressCfg.BaseURL,
		NumBots:   config.StressCfg.NumBots,
		BotInfoCSV: config.StressCfg.BotInfoCSV,
	}

	// Create and start manager
	manager, err := stress.NewManager(stressConfig)
	if err != nil {
		log.Fatalf("failed to create manager: %v", err)
	}

	if err := manager.Start(); err != nil {
		log.Fatalf("failed to start manager: %v", err)
	}
	log.Info("start stress test success")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Info("receive signal, exit")
	manager.Stop()
	log.Info("stress test stopped")
}
