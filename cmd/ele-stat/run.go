/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/spf13/cobra"

	"github.com/CryptoElementals/common/cmd/ele-stat/rpc/server"
)

var (
	configPath string
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "start stat server",
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize configuration, logging, database
		err := config.InitStatConfig(configPath)
		if err != nil {
			log.Fatal("init config failed, err: %v", err)
		}
		err = log.InitGlobalLogger(&config.StatGConf.LogCfg)
		if err != nil {
			log.Fatal("init logger failed, err: %v", err)
		}
		log.Info("config: %+v", config.StatGConf)
		err = db.Init(&config.StatGConf.DbCfg)
		if err != nil {
			log.Fatal("init db failed, err: %v", err)
		}

		// Start gRPC service
		listenPort := uint32(30011)
		if config.StatGConf.ListenPort != 0 {
			listenPort = config.StatGConf.ListenPort
		}

		// Create server configuration
		serverConfig := server.DefaultServerConfig(listenPort)

		// Create and start gRPC server
		statServer, err := server.NewStatServer(serverConfig)
		if err != nil {
			log.Fatal("failed to create stat server: %v", err)
		}

		// Print server information
		serverInfo := statServer.GetServerInfo()
		log.Infof("Server configuration: %+v", serverInfo)

		// Start server (in goroutine)
		go func() {
			if err := statServer.Start(); err != nil {
				log.Fatalf("failed to start server: %v", err)
			}
		}()

		log.Infof("gRPC server started on port %d", listenPort)

		// Graceful shutdown handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Wait for signal
		sig := <-sigChan
		log.Infof("Received signal: %v, starting graceful shutdown...", sig)

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Execute shutdown operation in goroutine
		done := make(chan struct{})
		go func() {
			if err := statServer.Stop(); err != nil {
				log.Errorf("Error stopping server: %v", err)
			}
			close(done)
		}()

		// Wait for shutdown completion or timeout
		select {
		case <-done:
			log.Info("Server shutdown completed successfully")
		case <-shutdownCtx.Done():
			log.Warn("Server shutdown timeout, forcing exit")
		}

		log.Info("Stat server exited")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	runCmd.MarkFlagRequired("config")
}
