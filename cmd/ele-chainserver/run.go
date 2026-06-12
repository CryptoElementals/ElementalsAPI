package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	chainserver "github.com/CryptoElementals/common/chain_server"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/spf13/cobra"
)

var configPath string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "start chain server",
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.InitCSConfig(configPath); err != nil {
			log.Fatal("init config failed, err: %v", err)
		}
		if err := log.InitGlobalLogger(&config.CSGConf.LogCfg); err != nil {
			log.Fatal("init logger failed, err: %v", err)
		}
		log.Infof("config: %+v", config.CSGConf)
		if err := db.Init(&config.CSGConf.DbCfg); err != nil {
			log.Fatal("init db failed, err: %v", err)
		}
		log.Info("db initialized")
		if err := db.Migrate(); err != nil {
			log.Fatalf("db migrate failed, err: %v", err)
		}
		log.Info("db migrated")

		svr, err := chainserver.New(context.Background(), &config.CSGConf)
		if err != nil {
			log.Fatalf("init chain server failed, err: %v", err)
		}
		log.Info("chain server initialized")
		if err := svr.Start(); err != nil {
			log.Fatalf("start chain server failed, err: %v", err)
		}
		log.Info("start chain server success")

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Info("receive signal, exit")
		svr.Stop()
		log.Info("chain server stopped")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	runCmd.MarkFlagRequired("config")
}
