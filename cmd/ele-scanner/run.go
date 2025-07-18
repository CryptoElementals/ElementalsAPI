package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().String("rpc", "ws://localhost:8546", "Opstack chain RPC address (recommended ws://)")
	runCmd.Flags().String("http", "http://localhost:8545", "Opstack chain HTTP RPC address (for catch-up)")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Real-time sync of opstack chain transactions, with catch-up after reconnect",
	Run: func(cmd *cobra.Command, args []string) {
		wsRpc, _ := cmd.Flags().GetString("rpc")
		httpRpc, _ := cmd.Flags().GetString("http")
		ctx := context.Background()

		var lastBlockNumber uint64 = 0 // You can persist this to file/db

		for {
			client, err := ethclient.Dial(wsRpc)
			if err != nil {
				log.Printf("Failed to connect to WebSocket RPC: %v, retrying in 5 seconds...", err)
				time.Sleep(5 * time.Second)
				continue
			}
			fmt.Println("WebSocket connected, subscribing to new blocks...")

			headers := make(chan *types.Header)
			sub, err := client.SubscribeNewHead(ctx, headers)
			if err != nil {
				log.Printf("Failed to subscribe to new blocks: %v, retrying in 5 seconds...", err)
				client.Close()
				time.Sleep(5 * time.Second)
				continue
			}

			// Subscription main loop
			for {
				select {
				case err := <-sub.Err():
					log.Printf("Subscription error: %v, reconnecting in 5 seconds...", err)
					sub.Unsubscribe()
					client.Close()
					time.Sleep(5 * time.Second)
					// Catch up missed blocks after disconnect
					lastBlockNumber = catchUpMissedBlocks(httpRpc, lastBlockNumber)
					goto RECONNECT
				case header := <-headers:
					block, err := client.BlockByHash(ctx, header.Hash())
					if err != nil {
						log.Printf("Failed to get block: %v", err)
						continue
					}
					handleBlock(block)
					lastBlockNumber = block.NumberU64()
				}
			}
		RECONNECT:
			// Next reconnect loop
		}
	},
}

func catchUpMissedBlocks(httpRpc string, lastBlockNumber uint64) uint64 {
	ctx := context.Background()
	client, err := ethclient.Dial(httpRpc)
	if err != nil {
		log.Printf("Failed to connect to HTTP RPC for catch-up: %v", err)
		return lastBlockNumber
	}
	defer client.Close()

	latest, err := client.BlockNumber(ctx)
	if err != nil {
		log.Printf("Failed to get latest block number for catch-up: %v", err)
		return lastBlockNumber
	}
	if lastBlockNumber >= latest {
		return lastBlockNumber
	}
	fmt.Printf("Catching up blocks: [%d, %d]\n", lastBlockNumber+1, latest)
	for i := lastBlockNumber + 1; i <= latest; i++ {
		for {
			block, err := client.BlockByNumber(ctx, big.NewInt(int64(i)))
			if err != nil {
				log.Printf("Failed to catch up block %d: %v, retrying in 3 seconds...", i, err)
				time.Sleep(3 * time.Second)
				continue // Retry current block
			}
			handleBlock(block)
			lastBlockNumber = i
			break // Success, move to next block
		}
	}
	return lastBlockNumber
}

// Process block and transactions
func handleBlock(block *types.Block) {
	fmt.Printf("Block: %d, Hash: %s, TxCount: %d\n", block.NumberU64(), block.Hash().Hex(), len(block.Transactions()))
	for _, tx := range block.Transactions() {
		fmt.Printf("  Tx: %s, To: %v, Value: %s\n",
			tx.Hash().Hex(), tx.To(), tx.Value().String())
		// Extend: process receipt, events, etc.
	}
}
