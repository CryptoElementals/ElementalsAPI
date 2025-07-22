package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// WaitForReceipt waits for a transaction to be mined and returns its receipt
func WaitForReceipt(ctx context.Context, client *ethclient.Client, txHash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// ParseRoomCreatedEvent decodes the RoomCreated event from the receipt logs using the contract ABI
func ParseRoomCreatedEvent(receipt *types.Receipt, contractAbi *abi.ABI) (map[string]interface{}, error) {
	eventName := "RoomCreated"
	event, ok := contractAbi.Events[eventName]
	if !ok {
		return nil, fmt.Errorf("event %s not found in ABI", eventName)
	}
	eventSigHash := event.ID

	for _, vLog := range receipt.Logs {
		if vLog.Topics[0] == eventSigHash {
			dataMap := make(map[string]interface{})

			dataMap["Height"] = receipt.BlockNumber
			dataMap["TxHash"] = receipt.TxHash
			dataMap["BlockHash"] = receipt.BlockHash
			dataMap["BlockNumber"] = receipt.BlockNumber
			dataMap["TransactionIndex"] = receipt.TransactionIndex
			dataMap["Status"] = receipt.Status

			if err := contractAbi.UnpackIntoMap(dataMap, eventName, vLog.Data); err != nil {
				return nil, err
			}
			// Decode indexed params
			for i, input := range event.Inputs {
				if input.Indexed {
					dataMap[input.Name] = vLog.Topics[i+1].Hex()
				}
			}
			return dataMap, nil
		}
	}
	return nil, fmt.Errorf("RoomCreated event not found in receipt")
}

// CreateRoomAndWaitReceiptAndParseEvent demonstrates the full process: call contract, wait for receipt, decode event
func CreateRoomAndWaitReceiptAndParseEvent(
	client *ethclient.Client,
	contractAddr common.Address,
	contractAbi *abi.ABI,
	roomManagerContract RoomManagerContract, // use interface, not *interface
	bindOpts *bind.TransactOpts,
	player1, player2, temp1, temp2 common.Address,
	roundTimeout, maxRounds *big.Int,
	timeout time.Duration,
) (map[string]interface{}, error) {
	tx, err := roomManagerContract.CreateRoom(bindOpts, player1, player2, temp1, temp2, roundTimeout, maxRounds)
	if err != nil {
		return nil, err
	}
	receipt, err := WaitForReceipt(context.Background(), client, tx.Hash(), timeout)
	if err != nil {
		return nil, err
	}
	if receipt.Status != 1 {
		return nil, fmt.Errorf("tx failed, status=%d", receipt.Status)
	}
	eventData, err := ParseRoomCreatedEvent(receipt, contractAbi)
	if err != nil {
		return nil, err
	}
	eventData["Sender"] = bindOpts.From
	return eventData, nil
}

// RoomManagerContract is a placeholder for the abigen-generated contract binding
// Replace this with your actual abigen binding import and type
type RoomManagerContract interface {
	CreateRoom(opts *bind.TransactOpts, player1, player2, temp1, temp2 common.Address, roundTimeout, maxRounds *big.Int) (*types.Transaction, error)
}
