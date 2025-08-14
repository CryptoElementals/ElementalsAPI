package main

import (
	"os"
	"os/signal"
	"syscall"

	botserver "github.com/CryptoElementals/common/bot-server"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "start a bot server",
	Run: func(cmd *cobra.Command, args []string) {
		runBotServer()
	},
}

var configPath string

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	runCmd.MarkFlagRequired("config")
}

func runBotServer() {
	err := config.InitBotConfig(configPath)
	if err != nil {
		panic("init config failed: " + err.Error())
	}
	err = log.InitGlobalLogger(&config.BotCfg.LogCfg)
	if err != nil {
		panic("init logger failed: " + err.Error())
	}
	svr := botserver.NewBotServer(&config.BotCfg)
	svr.Start()
	log.Info("start bot server success")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Info("receive signal, exit")
	svr.Stop()
}
