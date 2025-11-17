package chain

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type batchTxEvent struct {
	txs       *proto.TransactionBatch
	blockNum  uint64
	blockHash []byte
}

type Chain struct {
	ctx                   context.Context
	workerManager         *worker.WorkerManager
	createRoomTxToGameID  cache.Cache
	gameContractToRoomID  cache.Cache
	roomV2Client          *concurrentRoomV2Client
	roomV2ContractAddress string
}

func NewChain(
	ctx context.Context,
	workerManager *worker.WorkerManager,
	chainID int64,
	client bind.ContractBackend,
	roomV2ContractAddressHex string,
	wallets []*wallet.Wallet,
	dataCache cache.Cache,
	isDevelop ...bool,
) (*Chain, error) {
	if roomV2ContractAddressHex == "" {
		return nil, errors.New("room v2 contract address is required")
	}
	roomV2Cli, err := newConcurrentRoomV2Client(ctx, client, roomV2ContractAddressHex, wallets, chainID, isDevelop...)
	if err != nil {
		log.Errorf("newConcurrentRoomV2Client: create room v2 client failed: %s", err.Error())
		return nil, err
	}
	return &Chain{
		ctx:                   ctx,
		workerManager:         workerManager,
		createRoomTxToGameID:  cache.WithPrefix("create_room_tx_to_game_id", dataCache),
		gameContractToRoomID:  cache.WithPrefix("game_contract_to_room_id", dataCache),
		roomV2Client:          roomV2Cli,
		roomV2ContractAddress: strings.ToLower(roomV2ContractAddressHex),
	}, nil
}

func (c *Chain) Start() error {
	txs, err := db.ListCreateRoomTxWithNoContractAddr()
	if err != nil {
		return err
	}
	const cacheTTL = 3600 // 1 hour in seconds
	for _, tx := range txs {
		err = c.createRoomTxToGameID.Set(tx.TxHash, fmt.Sprint(tx.GameID), cacheTTL)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Chain) CreateRoomContract(evt *types.RequireContractCreationEvent) error {
	return c.createNewRoom(evt)
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
			log.Infof("gameCreated: gameID %d, blockHash %s, blockNumber %d, tx %s", gid, blockHash, blockNumber, hash)
			c.gameCreated(batchTx, blockTime, uint(gid), hash, blockHash, blockNumber, tx)
		case *proto.Transaction_GameTurnSetupReady:
			log.Infof("gameTurnSetupReady: gameID %d, blockHash %s, blockNumber %d, tx %s", gid, blockHash, blockNumber, hash)
			c.gameTurnSetupReady(batchTx, blockTime, uint(gid), hash, blockHash, blockNumber, tx)
		case *proto.Transaction_CommitmentOnChain:
			address := types.PlayerAddress{}
			address.FromProto(tx.CommitmentOnChain.Address)
			log.Infof("commitmentOnChain: gameID %d, blockHash %s, blockNumber %d, tx %s, address %s", gid, blockHash, blockNumber, hash, address.String())
			c.commitmentOnChain(batchTx, blockTime, gid, hash, blockHash, blockNumber, tx)
		case *proto.Transaction_CardOnChain:
			address := types.PlayerAddress{}
			address.FromProto(tx.CardOnChain.Address)
			log.Infof("cardOnChain: gameID %d, blockHash %s, blockNumber %d, tx %s, player address %s", gid, blockHash, blockNumber, hash, address.String())
			c.cardOnChain(batchTx, blockTime, gid, hash, blockHash, blockNumber, tx)
		}
	}
	batchTx.Wait()
}

func (c *Chain) gameCreated(batchTx *types.EventBatch, blockTime int64, gameID uint, txHash string, blockHash string, blockNumber uint64, tx *proto.Transaction_GameCreated) error {
	// For RoomV2, there's only one contract address, so RoomContractAddress is not needed
	roomContract := c.roomV2ContractAddress
	contractCreatedEvt := types.NewEvent(types.CHAIN_MANAGER_ID, &types.RoomContractCreated{
		GameID:    gameID,
		TimeStamp: blockTime,
	}, true)
	batchTx.Add(contractCreatedEvt)
	c.workerManager.SendEvent(fmt.Sprint(gameID), contractCreatedEvt)
	const cacheTTL = 3600 // 1 hour in seconds
	c.gameContractToRoomID.Set(roomContract, fmt.Sprint(gameID), cacheTTL)
	err := c.createRoomTxToGameID.Delete(txHash)
	if err != nil {
		log.Errorf("createRoomTxToGameID: delete tx with hash %s from cache failed: %s", txHash, err.Error())
	}
	return db.UpdateCreateRoomTxBlockHashAndContractByGameID(gameID, blockHash, blockNumber, roomContract)
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
	return db.SaveCommitmentOnChainTx(&dao.CommitmentOnChainTx{
		GameID:           gameID,
		ContractAddress:  c.roomV2ContractAddress,
		TxHash:           txHash,
		BlockHash:        blockHash,
		BlockNumber:      blockNumber,
		Status:           dao.TxStatusSent,
		RoundNumber:      uint64(roundNumber),
		WalletAddress:    player.WalletAddress,
		TemporaryAddress: player.TemporaryAddress,
	})
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

	return db.SaveCardsOnChainTx(&dao.CardsOnChainTx{
		GameID:           gameID,
		ContractAddress:  c.roomV2ContractAddress,
		TxHash:           txHash,
		BlockHash:        blockHash,
		BlockNumber:      blockNumber,
		Status:           dao.TxStatusSent,
		RoundNumber:      uint64(roundNumber),
		WalletAddress:    player.WalletAddress,
		TemporaryAddress: player.TemporaryAddress,
	})
}
