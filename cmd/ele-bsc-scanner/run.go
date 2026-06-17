package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CryptoElementals/common/cmd/ele-bsc-scanner/scanner"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run BSC finalized block scanner",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())

		if err := config.InitBscScannerConfig(configPath); err != nil {
			fmt.Printf("load config failed: %+v\n", err)
			os.Exit(1)
		}
		if err := log.InitGlobalLogger(&config.BscScannerGConf.LogCfg); err != nil {
			fmt.Printf("failed to initialize logger: %s\n", err.Error())
			os.Exit(1)
		}
		if err := db.Init(&config.BscScannerGConf.DbCfg); err != nil {
			fmt.Printf("failed to initialize database: %s\n", err.Error())
			os.Exit(1)
		}

		bscScanner, err := scanner.NewBscScanner(ctx)
		if err != nil {
			fmt.Printf("NewBscScanner failed: %+v\n", err)
			os.Exit(1)
		}
		bscScanner.Run()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		log.Info("Received shutdown signal, closing ele-bsc-scanner...")
		cancel()
		bscScanner.Stop()
		time.Sleep(3 * time.Second)
	},
}
