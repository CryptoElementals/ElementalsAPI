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
	eleClient "github.com/CryptoElementals/common/rpc/client"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
)

var (
	gethClient    *ethclient.Client
	catchupCancel context.CancelFunc
)

func init() {
	rootCmd.AddCommand(runCmd)
}

// Scanner encapsulates the state and logic for block catching up
type Scanner struct {
	ctx                  context.Context
	client               *ethclient.Client
	roomManagerAbi       *abi.ABI
	currentScannedHeight uint64
	headNumberOnChain    uint64
}

func NewScanner(ctx context.Context, client *ethclient.Client, roomManagerAbi *abi.ABI, startHeight, headHeight uint64) *Scanner {
	return &Scanner{
		ctx:                  ctx,
		client:               client,
		roomManagerAbi:       roomManagerAbi,
		currentScannedHeight: startHeight,
		headNumberOnChain:    headHeight,
	}
}

func (s *Scanner) SetHeadNumberOnChain(height uint64) {
	s.headNumberOnChain = height
}

func (s *Scanner) CatchUpChain() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			if s.currentScannedHeight > s.headNumberOnChain {
				time.Sleep(time.Millisecond * 200)
				continue
			}
			err := s.getAndProcessBlock(big.NewInt(int64(s.currentScannedHeight)))
			if err != nil {
				log.Warnf("catchUpChain goroutine parse block err %v", err.Error())
				time.Sleep(time.Second * 5)
				continue
			}
			err = db.SaveBlockSync(dao.BlockSync{Type: "head", BlockHeight: s.currentScannedHeight})
			if err != nil {
				log.Errorf("insert head block sync to db err %v", err.Error())
				time.Sleep(time.Second * 5)
				continue
			}
			log.Debugf("block %d handled successfully", s.currentScannedHeight)
			s.currentScannedHeight++
		}
	}
}

func (s *Scanner) getAndProcessBlock(blockHeight *big.Int) error {
	block, err := blockchain.GetOptimismBlockByNumber(s.ctx, config.ScannerGConf.ChainCfg.HttpRpc, blockHeight)
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
			roomCreatedTx, err := processCreateRoomTx(s.ctx, s.client, tx, s.roomManagerAbi)
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
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := log.InitGlobalLogger(&config.ScannerGConf.LogCfg); err != nil {
			fmt.Printf("failed to initialize logger: %s\n", err.Error())
			return
		}
		log.Info("Logger system initialized successfully")

		if err := db.Init(&config.ScannerGConf.DbCfg); err != nil {
			log.Errorf("failed to initialize database: %s", err.Error())
			fmt.Printf("failed to initialize database: %s\n", err.Error())
			return
		}
		log.Info("Database connection initialized successfully")

		var headNumberOnChain, currentScannedHeight uint64
		for {
			syncs, err := db.FindBlockSyncs()
			if err != nil {
				log.Errorf("db.FindBlockSyncs() failed, err %s", err.Error())
				time.Sleep(10 * time.Second)
				continue
			}
			for _, sync := range syncs {
				if sync.Type == "head" {
					headNumberOnChain = sync.BlockHeight
					currentScannedHeight = sync.BlockHeight + 1
				}
			}
			break
		}

		rpcClient, err := eleClient.NewRpcClient("localhost:50051")
		if err != nil {
			log.Errorf("Failed to create rpcClient to roomServer: %v", err)
			return
		}
		defer rpcClient.Close()

		dialTimeout := 3
		var scanner *Scanner
		for {
			gethClient, err = ethclient.Dial(wsRpc)
			if err != nil {
				log.Errorf("Failed to connect to WebSocket RPC: %v, retrying in %d seconds...", err.Error(), dialTimeout)
				time.Sleep(time.Duration(dialTimeout) * time.Second)
				continue
			}
			log.Info("WebSocket connected, subscribing to new blocks...")

			if catchupCancel != nil {
				catchupCancel() // stop old goroutine
			}
			catchupCtx, cancel := context.WithCancel(ctx)
			catchupCancel = cancel
			if scanner != nil {
				currentScannedHeight = scanner.currentScannedHeight
				headNumberOnChain = scanner.headNumberOnChain
			}
			scanner = NewScanner(catchupCtx, gethClient, roomManagerAbi, currentScannedHeight, headNumberOnChain)
			go scanner.CatchUpChain()

			headers := make(chan *types.Header)
			sub, err := gethClient.SubscribeNewHead(ctx, headers)
			if err != nil {
				log.Infof("Failed to subscribe to new blocks: %v, retrying in %d seconds...", err.Error(), dialTimeout)
				gethClient.Close()
				time.Sleep(time.Duration(dialTimeout) * time.Second)
				continue
			}

			for {
				select {
				case err := <-sub.Err():
					log.Infof("Subscription error: %v, reconnecting in %d seconds...", err.Error(), dialTimeout)
					sub.Unsubscribe()
					gethClient.Close()
					time.Sleep(time.Duration(dialTimeout) * time.Second)
					goto RECONNECT
				case header := <-headers:
					headNumberOnChain := header.Number.Uint64()
					if scanner != nil {
						scanner.SetHeadNumberOnChain(headNumberOnChain)
					}
					log.Debugf("HeadNumberOnChain is %d", headNumberOnChain)
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
