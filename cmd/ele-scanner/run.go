package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/CryptoElementals/common/cmd/ele-scanner/scanner"
	"github.com/CryptoElementals/common/config"
	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

func readRoomManagerAbi() (*abi.ABI, error) {
	roomManagerAbi, err := abi.JSON(strings.NewReader(contract.RoomManagerContractABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}
	return &roomManagerAbi, nil
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Real-time sync of Optimism chain transactions using official tools",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())

		roomManagerAbi, err := readRoomManagerAbi()
		if err != nil {
			fmt.Printf("readRoomManagerAbi()  failed: %+v", err)
			os.Exit(-1)
		}
		err = config.InitScannerConfig(configPath)
		if err != nil {
			fmt.Printf("load config failed: %+v", err)
			os.Exit(-1)
		}

		if err := log.InitGlobalLogger(&config.ScannerGConf.LogCfg); err != nil {
			fmt.Printf("failed to initialize logger: %s\n", err.Error())
			os.Exit(-1)
		}
		log.Info("Logger system initialized successfully")

		if err := db.Init(&config.ScannerGConf.DbCfg); err != nil {
			log.Errorf("failed to initialize database: %s", err.Error())
			fmt.Printf("failed to initialize database: %s\n", err.Error())
			os.Exit(-1)
		}
		log.Info("Database connection initialized successfully")

		gethWsRpc := config.ScannerGConf.ChainCfg.WsRpc
		gethHttpRpc := config.ScannerGConf.ChainCfg.HttpRpc
		roomServerHttpRpc := config.ScannerGConf.RoomServerHttpRpc
		roomManagerAddress := config.ScannerGConf.ChainCfg.ContractConfig.RoomManagerAddress
		scanner := scanner.NewScanner(ctx, gethWsRpc, gethHttpRpc, roomServerHttpRpc, roomManagerAddress, roomManagerAbi)
		scanner.Run()

		// Wait for interrupt signal
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		log.Info("Received shutdown signal, closing scanner server...")

		cancel()
		scanner.Stop()
		time.Sleep(6 * time.Second)

	},
}
