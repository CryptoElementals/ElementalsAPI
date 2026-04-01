package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	lobbyserver "github.com/CryptoElementals/common/lobby_server"
	"github.com/spf13/cobra"
)

var configPath string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "start lobby server",
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.InitLSConfig(configPath); err != nil {
			fmt.Printf("init config failed: %v\n", err)
			os.Exit(1)
		}
		if err := log.InitGlobalLogger(&config.LSGConf.LogCfg); err != nil {
			fmt.Printf("init logger failed: %v\n", err)
			os.Exit(1)
		}
		log.Infof("lobby config: %+v", config.LSGConf)
		if err := db.Init(&config.LSGConf.DbCfg); err != nil {
			log.Fatalf("init db failed: %v", err)
		}
		if !config.LSGConf.IsDevelop {
			if err := redis.Init(&config.LSGConf.RedisCfg); err != nil {
				log.Fatalf("init redis failed: %v", err)
			}
		}
		svr, err := lobbyserver.New(context.Background(), &config.LSGConf)
		if err != nil {
			log.Fatalf("init lobby server failed: %v", err)
		}
		if err := svr.Start(); err != nil {
			log.Fatalf("start lobby server failed: %v", err)
		}
		log.Info("lobby server started")
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		svr.Stop()
		log.Info("lobby server stopped")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	runCmd.MarkFlagRequired("config")
}
