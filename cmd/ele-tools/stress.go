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

var stressCmd = &cobra.Command{
	Use:   "stress",
	Short: "stress-test the game server with multiple bots",
	Long: `Runs multiple bots against the game server: wallet generation, login,
and game loops. Use stress run with a stress config YAML.`,
}

var stressRunCmd = &cobra.Command{
	Use:   "run",
	Short: "start stress test",
	Run: func(cmd *cobra.Command, args []string) {
		runStressTest()
	},
}

var stressConfigPath string

func init() {
	rootCmd.AddCommand(stressCmd)
	stressCmd.AddCommand(stressRunCmd)
	stressRunCmd.Flags().StringVarP(&stressConfigPath, "config", "c", "", "stress config file path")
	_ = stressRunCmd.MarkFlagRequired("config")
}

func runStressTest() {
	if err := config.InitStressConfig(stressConfigPath); err != nil {
		panic("init config failed: " + err.Error())
	}
	if err := log.InitGlobalLogger(&config.StressCfg.LogCfg); err != nil {
		panic("init logger failed: " + err.Error())
	}

	stressConfig := &stress.Config{
		BaseURL:    config.StressCfg.BaseURL,
		NumBots:    config.StressCfg.NumBots,
		BotInfoCSV: config.StressCfg.BotInfoCSV,
	}

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
