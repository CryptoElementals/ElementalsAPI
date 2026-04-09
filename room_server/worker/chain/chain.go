package chain

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/ethclient"
)

type batchTxEvent struct {
	txs       *proto.TransactionBatch
	blockNum  uint64
	blockHash []byte
}

type Chain struct {
	ctx                   context.Context
	workerManager         *worker.WorkerManager
	roomV3Client          *concurrentRoomV3Client
	roomV2ContractAddress string

	// track chain tx handling time from submission to on-chain completion
	txTimes     map[string]time.Time
	txTimesLock sync.Mutex
}

func NewChain(
	ctx context.Context,
	workerManager *worker.WorkerManager,
	chainID int64,
	client *ethclient.Client,
	roomV3ContractAddressHex string,
	wallets []*wallet.Wallet,
	isDevelop ...bool,
) (*Chain, error) {
	if roomV3ContractAddressHex == "" {
		return nil, errors.New("room contract address is required")
	}
	roomV3Cli, err := newConcurrentRoomV3Client(ctx, client, roomV3ContractAddressHex, wallets, chainID, isDevelop...)
	if err != nil {
		log.Errorf("newConcurrentRoomV3Client: create room v3 client failed: %s", err.Error())
		return nil, err
	}
	return &Chain{
		ctx:                   ctx,
		workerManager:         workerManager,
		roomV3Client:          roomV3Cli,
		roomV2ContractAddress: strings.ToLower(roomV3ContractAddressHex),
		txTimes:               make(map[string]time.Time),
	}, nil
}

// SubmitTasks submits a batch of pre-encoded RoomV3 tasks to the chain.
// Tasks are ABI-encoded payloads compatible with RoomV3.batchSubmitTasks.
func (c *Chain) SubmitTasks(tasks []types.RoomContractTask) error {
	if c.roomV3Client == nil {
		return errors.New("room v3 client not initialized")
	}
	if len(tasks) == 0 {
		return nil
	}

	indexes := make([]uint8, 0, len(tasks))
	payloads := make([][]byte, 0, len(tasks))
	for _, t := range tasks {
		indexes = append(indexes, t.Index)
		payloads = append(payloads, t.Task)
	}

	txHash, err := c.roomV3Client.submitTasks(indexes, payloads)
	if err != nil {
		return err
	}
	c.recordTxStart(txHash, "submitTasks", len(tasks))
	return nil
}

func (c *Chain) recordTxStart(txHash string, eventName string, taskCount int) {
	if txHash == "" {
		return
	}
	now := time.Now()
	c.txTimesLock.Lock()
	c.txTimes[txHash] = now
	c.txTimesLock.Unlock()
	log.Debugw("chain tx sent",
		"event", eventName,
		"tx_hash", txHash,
		"task_count", taskCount,
	)
}

func (c *Chain) Start() error {
	// No longer needed - transaction tables removed
	return nil
}

func (c *Chain) logTxCompletionIfTracked(hash string, gid int64, blockHash string, blockNumber uint64, eventName string) {
	// If we have a recorded submission time for this tx, log the end-to-end latency
	c.txTimesLock.Lock()
	start, ok := c.txTimes[hash]
	if ok {
		delete(c.txTimes, hash)
	}
	c.txTimesLock.Unlock()
	if !ok {
		return
	}

	elapsed := time.Since(start)
	log.Debugw("chain tx completed",
		"tx_hash", hash,
		"game_id", gid,
		"block_hash", blockHash,
		"block_number", blockNumber,
		"event", eventName,
		"duration_ms", elapsed.Milliseconds(),
	)
}

// NotifyTxsCompleted logs completion latency for tx hashes that were previously tracked by SubmitTasks.
// This is used when tx handling ingress is outside chain service (e.g. game manager).
func (c *Chain) NotifyTxsCompleted(txs *proto.TransactionBatch) {
	if txs == nil {
		return
	}
	blockHash := strings.ToLower("0x" + hex.EncodeToString(txs.BlockHash))
	blockNumber := txs.BlockNumber
	for _, protoTx := range txs.Transactions {
		hash := strings.ToLower("0x" + hex.EncodeToString(protoTx.TxHash))
		gid := protoTx.GameId
		switch protoTx.Tx.(type) {
		case *proto.Transaction_GameCreated:
			c.logTxCompletionIfTracked(hash, gid, blockHash, blockNumber, "gameCreated")
		case *proto.Transaction_GameTurnSetupReady:
			c.logTxCompletionIfTracked(hash, gid, blockHash, blockNumber, "gameTurnSetupReady")
		case *proto.Transaction_CommitmentOnChain:
			c.logTxCompletionIfTracked(hash, gid, blockHash, blockNumber, "commitmentOnChain")
		case *proto.Transaction_CardOnChain:
			c.logTxCompletionIfTracked(hash, gid, blockHash, blockNumber, "cardOnChain")
		}
	}
}
