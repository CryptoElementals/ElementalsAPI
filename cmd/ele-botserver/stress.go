/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	botserver "github.com/CryptoElementals/common/bot-server"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/spf13/cobra"
)

// stressCmd represents the stress command
var stressCmd = &cobra.Command{
	Use:   "stress",
	Short: "stress test elementra backend server",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("stress called")
	},
}

func init() {
	rootCmd.AddCommand(stressCmd)
	stressCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	stressCmd.MarkFlagRequired("config")
}

func runStressServer() {
	err := config.InitBotConfig(configPath)
	if err != nil {
		panic("init config failed: " + err.Error())
	}
	err = log.InitGlobalLogger(&config.BotCfg.LogCfg)
	if err != nil {
		panic("init logger failed: " + err.Error())
	}
	svr := botserver.NewStressService(context.Background(), &config.BotCfg)
	svr.Start()
	log.Info("start bot server success")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Info("receive signal, exit")
	svr.Stop()
}
