package chain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
func (c *Chain) SubmitTasks(tasks [][]byte) error {
	if c.roomV3Client == nil {
		return errors.New("room v3 client not initialized")
	}
	if len(tasks) == 0 {
		return nil
	}
	txHash, err := c.roomV3Client.submitTasks(tasks)
	if err != nil {
		return err
	}
	c.recordTxStart(txHash, "submitTasks", 0)
	return nil
}

func (c *Chain) recordTxStart(txHash string, eventName string, gameID uint) {
	if txHash == "" {
		return
	}
	now := time.Now()
	c.txTimesLock.Lock()
	c.txTimes[txHash] = now
	c.txTimesLock.Unlock()
	if gameID != 0 {
		log.Debugw("chain tx sent",
			"event", eventName,
			"tx_hash", txHash,
			"game_id", gameID,
		)
	} else {
		log.Debugw("chain tx sent",
			"event", eventName,
			"tx_hash", txHash,
		)
	}

}

func (c *Chain) Start() error {
	// No longer needed - transaction tables removed
	return nil
}

func (c *Chain) batchSendTxs(evt *batchTxEvent) {
	c.handleChainEvents(evt)
}

func (c *Chain) handleChainEvents(evt *batchTxEvent) {
	batchTx := &types.EventBatch{}
	blockHash := strings.ToLower(hexutil.Encode(evt.blockHash))
	blockNumber := evt.blockNum
	blockTime := int64(evt.txs.Timestamp)
	for _, protoTx := range evt.txs.Transactions {
		hash := strings.ToLower(hexutil.Encode(protoTx.TxHash))
		gid := uint(protoTx.GameId)
		switch tx := protoTx.Tx.(type) {
		case *proto.Transaction_GameCreated:
			c.logTxCompletionIfTracked(hash, gid, blockHash, blockNumber, "gameCreated")
			c.gameCreated(batchTx, blockTime, uint(gid), hash, blockHash, blockNumber, tx)
		case *proto.Transaction_GameTurnSetupReady:
			c.logTxCompletionIfTracked(hash, gid, blockHash, blockNumber, "gameTurnSetupReady")
			c.gameTurnSetupReady(batchTx, blockTime, uint(gid), hash, blockHash, blockNumber, tx)
		case *proto.Transaction_CommitmentOnChain:
			address := types.PlayerAddress{}
			address.FromProto(tx.CommitmentOnChain.Address)
			c.logTxCompletionIfTracked(hash, gid, blockHash, blockNumber, "commitmentOnChain")
			c.commitmentOnChain(batchTx, blockTime, gid, hash, blockHash, blockNumber, tx)
		case *proto.Transaction_CardOnChain:
			address := types.PlayerAddress{}
			address.FromProto(tx.CardOnChain.Address)
			c.logTxCompletionIfTracked(hash, gid, blockHash, blockNumber, "cardOnChain")
			c.cardOnChain(batchTx, blockTime, gid, hash, blockHash, blockNumber, tx)
		}
	}
	batchTx.Wait()
}

func (c *Chain) logTxCompletionIfTracked(hash string, gid uint, blockHash string, blockNumber uint64, eventName string) {
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

func (c *Chain) gameCreated(batchTx *types.EventBatch, blockTime int64, gameID uint, txHash string, blockHash string, blockNumber uint64, tx *proto.Transaction_GameCreated) error {
	// For RoomV2, there's only one contract address, so RoomContractAddress is not needed
	contractCreatedEvt := types.NewEvent(types.CHAIN_MANAGER_ID, &types.RoomCreated{
		GameID:    gameID,
		TimeStamp: blockTime,
	}, true)
	batchTx.Add(contractCreatedEvt)
	c.workerManager.SendEvent(fmt.Sprint(gameID), contractCreatedEvt)
	// Transaction tables removed - no longer saving to database
	return nil
}

func (c *Chain) gameTurnSetupReady(batchTx *types.EventBatch, blockTime int64, gameID uint, txHash string, blockHash string, blockNumber uint64, tx *proto.Transaction_GameTurnSetupReady) error {
	turnSetupReadyEvent := types.NewEvent(types.CHAIN_MANAGER_ID, &types.NewTurnSetupComplete{
		GameID:      gameID,
		RoundNumber: tx.GameTurnSetupReady.RoundNumber,
		TurnNumber:  tx.GameTurnSetupReady.TurnNumber,
		TimeStamp:   blockTime,
	}, true)
	batchTx.Add(turnSetupReadyEvent)
	c.workerManager.SendEvent(fmt.Sprint(gameID), turnSetupReadyEvent)
	return nil
}

func (c *Chain) commitmentOnChain(batchTx *types.EventBatch, blockTime int64, gameID uint, txHash string, blockHash string, blockNumber uint64, tx *proto.Transaction_CommitmentOnChain) error {
	player := types.PlayerAddress{}
	player.FromProto(tx.CommitmentOnChain.Address)
	roundNumber := tx.CommitmentOnChain.RoundNumber
	turnNumber := tx.CommitmentOnChain.TurnNumber
	commitment := tx.CommitmentOnChain.Commitment

	// Use turn_number as commitment index (turn_number is 1-based: 1, 2, 3)
	commitmentIndex := turnNumber

	commitmentOnChainEvent := types.NewEvent(types.CHAIN_MANAGER_ID, &types.PlayerCommitmentOnChain{
		GameID:          gameID,
		Address:         player,
		RoundNumber:     roundNumber,
		Commitment:      commitment,
		CommitmentIndex: commitmentIndex, // 1-based (1, 2, 3)
		TimeStamp:       blockTime,
	}, true)
	batchTx.Add(commitmentOnChainEvent)
	c.workerManager.SendEvent(fmt.Sprint(gameID), commitmentOnChainEvent)
	// Transaction tables removed - no longer saving to database
	return nil
}

func (c *Chain) cardOnChain(batchTx *types.EventBatch, blockTime int64, gameID uint, txHash string, blockHash string, blockNumber uint64, tx *proto.Transaction_CardOnChain) error {
	player := types.PlayerAddress{}
	player.FromProto(tx.CardOnChain.Address)
	roundNumber := tx.CardOnChain.RoundNumber
	turnNumber := tx.CardOnChain.TurnNumber
	salt := tx.CardOnChain.Salt
	cardID := uint(tx.CardOnChain.CardId)

	// Use turn_number as card index (turn_number is 1-based: 1, 2, 3)
	cardIndex := turnNumber

	cardOnChainEvent := types.NewEvent(types.CHAIN_MANAGER_ID, &types.PlayerCardOnChain{
		GameID:      gameID,
		Address:     player,
		RoundNumber: roundNumber,
		Salt:        salt,
		Card:        cardID,
		CardIndex:   cardIndex, // 1-based (1, 2, 3) - matches CommitmentIndex
		TimeStamp:   blockTime,
	}, true)
	batchTx.Add(cardOnChainEvent)
	c.workerManager.SendEvent(fmt.Sprint(gameID), cardOnChainEvent)
	// Transaction tables removed - no longer saving to database
	return nil
}
