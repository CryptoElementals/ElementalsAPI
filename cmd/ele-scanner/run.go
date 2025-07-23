package main

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/CryptoElementals/common/cmd/ele-scanner/blockchain"
	"github.com/CryptoElementals/common/config"
	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
)

var (
	HeadNumberOnChain    uint64 = 0
	currentScannedHeight uint64
	gethClient           *ethclient.Client
	catchupCancel        context.CancelFunc
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Real-time sync of Optimism chain transactions using official tools",
	Run: func(cmd *cobra.Command, args []string) {
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

		wsRpc := config.ScannerGConf.ChainCfg.WsRpc
		//httpRpc := config.ScannerGConf.ChainCfg.HttpRpc
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Initialize logger
		if err := log.InitGlobalLogger(&config.ScannerGConf.LogCfg); err != nil {
			fmt.Printf("failed to initialize logger: %s\n", err.Error())
			return
		}
		log.Info("Logger system initialized successfully")

		// Initialize database
		if err := db.Init(&config.ScannerGConf.DbCfg); err != nil {
			log.Errorf("failed to initialize database: %s", err.Error())
			fmt.Printf("failed to initialize database: %s\n", err.Error())
			return
		}
		log.Info("Database connection initialized successfully")

		for {
			syncs, err := db.FindBlockSyncs()
			if err != nil {
				log.Errorf("db.FindBlockSyncs() failed, err %s", err.Error())
				time.Sleep(10 * time.Second)
				continue
			}
			for _, sync := range syncs {
				if sync.Type == "head" {
					HeadNumberOnChain = sync.BlockHeight
					currentScannedHeight = sync.BlockHeight + 1
				}
			}
			break
		}

		dialTimeout := 3
		for {
			gethClient, err = ethclient.Dial(wsRpc)
			if err != nil {
				log.Errorf("Failed to connect to WebSocket RPC: %v, retrying in %d seconds...", err.Error(), dialTimeout)
				time.Sleep(time.Duration(dialTimeout) * time.Second)
				continue
			}
			log.Info("WebSocket connected, subscribing to new blocks...")

			if catchupCancel != nil {
				catchupCancel() // 让旧的 goroutine 退出
			}
			catchupCtx, cancel := context.WithCancel(ctx)
			catchupCancel = cancel
			go catchUpChain(catchupCtx, gethClient, roomManagerAbi)

			headers := make(chan *types.Header)
			sub, err := gethClient.SubscribeNewHead(ctx, headers)
			if err != nil {
				log.Infof("Failed to subscribe to new blocks: %v, retrying in %d seconds...", err.Error(), dialTimeout)
				gethClient.Close()
				time.Sleep(time.Duration(dialTimeout) * time.Second)
				continue
			}

			// Subscription main loop
			for {
				select {
				case err := <-sub.Err():
					log.Infof("Subscription error: %v, reconnecting in %d seconds...", err.Error(), dialTimeout)
					sub.Unsubscribe()
					gethClient.Close()
					time.Sleep(time.Duration(dialTimeout) * time.Second)
					goto RECONNECT
				case header := <-headers:
					HeadNumberOnChain = header.Number.Uint64()
					log.Debugf("HeadNumberOnChain is %d", HeadNumberOnChain)
				}
			}
		RECONNECT:
			// Next reconnect loop
		}
	},
}

func readRoomManagerAbi() (*abi.ABI, error) {
	roomManagerAbi, err := abi.JSON(strings.NewReader(contract.RoomManagerContractABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %v", err)
	}
	return &roomManagerAbi, nil
}

func catchUpChain(ctx context.Context, client *ethclient.Client, roomManagerAbi *abi.ABI) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if currentScannedHeight > HeadNumberOnChain {
				time.Sleep(time.Millisecond * 200)
			}
			err := getAndProcessBlock(ctx, client, big.NewInt(int64(currentScannedHeight)), roomManagerAbi)
			if err != nil {
				log.Warnf("catchUpChain goroutine parse block err %v", err.Error())
				continue
			}

			err = db.SaveBlockSync(dao.BlockSync{Type: "head", BlockHeight: currentScannedHeight})
			if err != nil {
				log.Errorf("insert head block sync to db err %v", err.Error())
				continue
			}

			log.Debugf("block %d handled successfully", currentScannedHeight)
			currentScannedHeight++
		}
	}

}

func getAndProcessBlock(ctx context.Context, client *ethclient.Client, blockHeight *big.Int, roomManagerAbi *abi.ABI) error {
	block, err := blockchain.GetOptimismBlockByNumber(ctx, config.ScannerGConf.ChainCfg.HttpRpc, blockHeight)
	if err != nil {
		log.Errorf("getBlockByNumber failed, err %s", err.Error())
		return err
	}
	parsedTxs, err := blockchain.ParseOptimismTransactions(block.Transactions)
	if err != nil {
		log.Errorf("ParseOptimismTransactions failed, err %s", err.Error())
		return err
	}
	if len(parsedTxs) != len(block.Transactions) {
		log.Errorf("Parsed tx count %d does not match raw tx count %d", len(parsedTxs), len(block.Transactions))
		return err
	}
	for _, tx := range parsedTxs {
		log.Debugf("parsed tx: %+v", tx)
		if strings.EqualFold(tx.To, config.ScannerGConf.ChainCfg.ContractConfig.RoomManagerAddress) {
			log.Debugf("room manager contract tx: %+v", tx)
			roomCreatedTx, err := processCreateRoomTx(ctx, client, tx, roomManagerAbi)
			if err != nil {
				log.Errorf("processCreateRoomTx failed, err %s", err.Error())
				continue
			}
			log.Debugf("room created tx: %+v", roomCreatedTx)
		}
	}

	return nil
}

func processCreateRoomTx(ctx context.Context, gethClient *ethclient.Client, tx blockchain.OptimismTx, roomManagerAbi *abi.ABI) (*blockchain.RoomCreatedTx, error) {
	hash := common.HexToHash(tx.Hash)
	receipt, err := gethClient.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}

	roomCreatedTx, err := blockchain.ParseRoomCreatedEvent(receipt, roomManagerAbi)
	if err != nil {
		return nil, err
	}

	return roomCreatedTx, nil
}
