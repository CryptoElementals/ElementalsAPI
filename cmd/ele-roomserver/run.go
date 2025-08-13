/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	roomserver "github.com/CryptoElementals/common/room_server"
	"github.com/spf13/cobra"
)

var (
	configPath string
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "start room server",
	Run: func(cmd *cobra.Command, args []string) {
		err := config.InitRSConfig(configPath)
		if err != nil {
			log.Fatal("init config failed, err: %v", err)
		}
		err = log.InitGlobalLogger(&config.RSGConf.LogCfg)
		if err != nil {
			log.Fatal("init logger failed, err: %v", err)
		}
		log.Infof("config: %+v", config.RSGConf)
		err = db.Init(&config.RSGConf.DbCfg)
		if err != nil {
			log.Fatal("init db failed, err: %v", err)
		}
		err = redis.Init(&config.RSGConf.RedisCfg)
		if err != nil {
			log.Fatalf("init redis failed, err: %v", err)
		}

		svr, err := roomserver.New(context.Background(), &config.RSGConf)
		if err != nil {
			log.Fatalf("init room server failed, err: %v", err)
		}
		err = svr.Start()
		if err != nil {
			log.Fatalf("start room server failed, err: %v", err)
		}
		log.Info("start room server success")
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Info("receive signal, exit")
		svr.Stop()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	runCmd.MarkFlagRequired("config")
}
