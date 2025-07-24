package chain

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/CryptoElementals/common/cache"
	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type batchTxEvent struct {
	txs       *proto.TransactionBatch
	blockNum  uint64
	blockHash []byte
	done      chan struct{}
	errChan   chan error
}

type Chain struct {
	ctx                        context.Context
	workerManager              *worker.WorkerManager
	createRoomTxToGameID       cache.Cache
	gameContractToRoomID       cache.Cache
	inflightEvents             map[string]struct{}
	currentBatchTx             *batchTxEvent
	client                     bind.ContractBackend
	roomManagerContractAddress common.Address

	bindOpts *bind.TransactOpts
}

func NewChain(
	ctx context.Context,
	workerManager *worker.WorkerManager,
	chainID int64,
	client bind.ContractBackend,
	roomManagerContractAddressHex string,
	w *wallet.Wallet,
	dataCache cache.Cache,
	isDevelop ...bool,
) *Chain {
	roomManagerContractAddress := common.HexToAddress(roomManagerContractAddressHex)
	bindOpts := &bind.TransactOpts{
		Context: ctx,
		From:    w.GetAddr(),
		Signer:  w.BuildTxSinger(big.NewInt(chainID)),
	}
	if len(isDevelop) != 0 && isDevelop[0] {
		bindOpts.NoSend = true
	}

	return &Chain{
		ctx:                        ctx,
		workerManager:              workerManager,
		createRoomTxToGameID:       cache.WithPrefix("create_room_tx_to_game_id", dataCache),
		gameContractToRoomID:       cache.WithPrefix("game_contract_to_room_id", dataCache),
		inflightEvents:             map[string]struct{}{},
		client:                     client,
		roomManagerContractAddress: roomManagerContractAddress,
		bindOpts:                   bindOpts,
	}
}

func (c *Chain) Start() error {
	txs, err := db.ListCreateRoomTxWithNoContractAddr()
	if err != nil {
		return err
	}
	for _, tx := range txs {
		err = c.createRoomTxToGameID.Set(tx.TxHash, fmt.Sprint(tx.GameID), int(time.Minute.Seconds()))
		if err != nil {
			return err
		}
	}
	c.createSelf()
	return nil
}

func (c *Chain) createSelf() {
	c.workerManager.SpwanWorker(c.ctx, types.CHAIN_MANAGER_ID, types.WORKER_TYPE_CHAIN, c)
}

func (c *Chain) Handle(ctx context.Context, event *types.Event) error {
	switch evt := event.Data.(type) {
	case *types.RequireContractCreationEvent:
		evt, err := types.AssertInterface[*types.RequireContractCreationEvent](event)
		if err != nil {
			return err
		}
		err = c.createRoomContract(evt.GameID, evt.Players, evt.InitialHP, evt.RoundTimeout, evt.MaxRoundNumber)
		if err != nil {
			return err
		}
		log.Debugf("created contract for %s", evt.Players)
	case *types.RequireSetupNewRoundEvent:
		err := c.setRoundReady(evt.GameID, evt.RoundNumber, evt.ContractAddress)
		if err != nil {
			return err
		}
	case *batchTxEvent:
		if c.currentBatchTx != nil {
			evt.errChan <- errors.New("a tx batch is inflight")
			close(evt.done)
			return nil
		}
		// send all events
		c.handleChainEvents(evt)
	case *types.AckEvent:
		// stale event
		if c.currentBatchTx == nil {
			return nil
		}
		delete(c.inflightEvents, evt.EventID)
		if len(c.inflightEvents) == 0 {
			close(c.currentBatchTx.errChan)
			close(c.currentBatchTx.done)
			c.currentBatchTx = nil
		}
	}
	return nil
}

func (c *Chain) batchSendTxs(evt *batchTxEvent) {
	// send a fake event to chain
	c.workerManager.SendEvent(types.CHAIN_MANAGER_ID, types.NewEvent("", evt))
}

func (c *Chain) createRoomContract(gameID uint, players []types.PlayerAddress, initialHP int64, roundTimeout int64, maxRounds int64) error {
	roomManagerContract, err := contract.NewRoomManagerContract(c.roomManagerContractAddress, c.client)
	if err != nil {
		return err
	}
	player1WalletAddress := common.HexToAddress(players[0].WalletAddress)
	player2WalletAddress := common.HexToAddress(players[1].WalletAddress)
	player1TemporaryAddress := common.HexToAddress(players[0].TemporaryAddress)
	player2TemporaryAddress := common.HexToAddress(players[1].TemporaryAddress)
	roundTimeoutBigInt := big.NewInt(roundTimeout)
	maxRoundsBigInt := big.NewInt(maxRounds)
	tx, err := roomManagerContract.CreateRoom(c.bindOpts, player1WalletAddress, player2WalletAddress,
		player1TemporaryAddress, player2TemporaryAddress, roundTimeoutBigInt, maxRoundsBigInt)
	if err != nil {
		return err
	}
	txHash := tx.Hash().String()
	c.createRoomTxToGameID.Set(txHash, fmt.Sprint(gameID), int(time.Minute.Seconds()))
	createRoomTxModel := &dao.CreateRoomTx{
		GameID:       gameID,
		Status:       dao.TxStatusSent,
		TxHash:       txHash,
		RoundTimeout: time.Duration(roundTimeout) * time.Second,
		MaxRounds:    uint64(maxRounds),
	}
	return db.SaveCreateRoomTx(createRoomTxModel)
}

func (c *Chain) setRoundReady(gameID uint, roundNumber uint32, roomContractHex string) error {
	roomContractAddress := common.HexToAddress(roomContractHex)
	roomContract, err := contract.NewRoomContract(roomContractAddress, c.client)
	if err != nil {
		return err
	}
	tx, err := roomContract.StartANewRound(c.bindOpts)
	if err != nil {
		return err
	}
	txHash := tx.Hash().String()
	createRoomTxModel := &dao.SetRoundReadyTx{
		GameID:          gameID,
		Status:          dao.TxStatusSent,
		ContractAddress: roomContractHex,
		RoundNumber:     uint64(roundNumber),
		TxHash:          txHash,
	}
	return db.SaveSetRoundReadyTx(createRoomTxModel)
}

func (c *Chain) handleChainEvents(evt *batchTxEvent) {
	c.currentBatchTx = evt
	blockHash := hexutil.Encode(evt.blockHash)
	blockNumber := evt.blockNum
	for _, protoTx := range c.currentBatchTx.txs.Transactions {
		hash := hexutil.Encode(protoTx.TxHash)
		switch tx := protoTx.Tx.(type) {
		case *proto.Transaction_RoomContractCreated:
			gidStr, err := c.createRoomTxToGameID.Get(hash)
			if err != nil {
				log.Errorf("createRoomTxToGameID: load tx with hash %s from cache failed: %s", hash, err.Error())
				continue
			}
			gid, err := strconv.Atoi(gidStr)
			if err != nil {
				log.Errorf("createRoomTxToGameID: decoded loaded tx with hash %s failed: %s", hash, err.Error())
				continue
			}
			c.contractCreated(uint(gid), blockHash, blockNumber, tx)
		case *proto.Transaction_RoomContractSetupReady:
			gid, err := c.getRoomIDByContract(tx.RoomContractSetupReady.RoomContractAddress)
			if err != nil {
				log.Errorf("cannot find room contract tx with contract hash %s, err: %s", err.Error())
				continue
			}
			c.roundSetupCompleted(gid, blockHash, blockNumber, tx)
		case *proto.Transaction_CommitmentsOnChain:
			gid, err := c.getRoomIDByContract(tx.CommitmentsOnChain.RoomContractAddress)
			if err != nil {
				log.Errorf("cannot find room contract tx with contract hash %s, err: %s", err.Error())
				continue
			}
			address := types.PlayerAddress{}
			address.FromProto(tx.CommitmentsOnChain.Address)
			c.commitmentOnChain(gid, hash, blockHash, blockNumber, tx)
		case *proto.Transaction_CardsOnChain:
			gid, err := c.getRoomIDByContract(tx.CardsOnChain.RoomContractAddress)
			if err != nil {
				log.Errorf("cannot find room contract tx with contract hash %s, err: %s", err.Error())
				continue
			}
			address := types.PlayerAddress{}
			address.FromProto(tx.CardsOnChain.Address)
			c.cardsOnChain(gid, hash, blockHash, blockNumber, tx)
		}
	}
}

func (c *Chain) getRoomIDByContract(contractAddress string) (uint, error) {
	gidStr, err := c.gameContractToRoomID.Get(contractAddress)
	if err == nil {
		gid, err := strconv.Atoi(gidStr)
		if err == nil {
			return uint(gid), nil
		}
	}
	dbRoom, err := db.GetCreateRoomTxByContract(contractAddress)
	if err != nil {
		return 0, err
	}
	c.gameContractToRoomID.Set(contractAddress, fmt.Sprint(dbRoom.GameID), int(time.Minute.Seconds()))
	return dbRoom.GameID, nil
}

func (c *Chain) contractCreated(gameID uint, blockHash string, blockNumber uint64, tx *proto.Transaction_RoomContractCreated) error {
	roomContract := tx.RoomContractCreated.RoomContractAddress
	contractCreatedEvt := types.NewEvent(types.CHAIN_MANAGER_ID, &types.RoomContractCreated{
		GameID:              gameID,
		RoomContractAddress: roomContract,
	}, true)
	evtID := contractCreatedEvt.EventID
	c.inflightEvents[evtID] = struct{}{}
	c.workerManager.SendEvent(fmt.Sprint(gameID), contractCreatedEvt)
	c.gameContractToRoomID.Set(roomContract, fmt.Sprint(gameID), int(time.Minute.Seconds()))
	err := c.createRoomTxToGameID.Delete(roomContract)
	if err != nil {
		log.Errorf("createRoomTxToGameID: delete tx with hash %s from cache failed: %s", roomContract, err.Error())
	}
	return db.UpdateCreateRoomTxBlockHashAndContractByGameID(gameID, blockHash, blockNumber, roomContract)
}

func (c *Chain) roundSetupCompleted(gameID uint, blockHash string, blockNumber uint64, tx *proto.Transaction_RoomContractSetupReady) error {
	roundNumber := tx.RoomContractSetupReady.RoundNumber
	roundSetupCompletedEvent := types.NewEvent(types.CHAIN_MANAGER_ID, &types.NewRoundSetupComplete{
		GameID:      gameID,
		RoundNumber: roundNumber,
	}, true)
	evtID := roundSetupCompletedEvent.EventID
	c.inflightEvents[evtID] = struct{}{}
	c.workerManager.SendEvent(fmt.Sprint(gameID), roundSetupCompletedEvent)
	return db.UpdateSetRoundReadyTxBlockHashByGameID(gameID, blockHash, blockNumber)
}

func (c *Chain) commitmentOnChain(gameID uint, txHash string, blockHash string, blockNumber uint64, tx *proto.Transaction_CommitmentsOnChain) error {
	player := types.PlayerAddress{}
	player.FromProto(tx.CommitmentsOnChain.Address)
	roundNumber := tx.CommitmentsOnChain.RoundNumber
	commitment := tx.CommitmentsOnChain.Commitment
	commitmentOnChainEvent := types.NewEvent(types.CHAIN_MANAGER_ID, &types.PlayerCommitmentOnChain{
		GameID:      gameID,
		Address:     player,
		RoundNumber: roundNumber,
		Commitment:  commitment,
	}, true)
	evtID := commitmentOnChainEvent.EventID
	c.inflightEvents[evtID] = struct{}{}
	c.workerManager.SendEvent(fmt.Sprint(gameID), commitmentOnChainEvent)
	return db.SaveCommitmentOnChainTx(&dao.CommitmentOnChainTx{
		GameID:           gameID,
		ContractAddress:  tx.CommitmentsOnChain.RoomContractAddress,
		TxHash:           txHash,
		BlockHash:        blockHash,
		BlockNumber:      blockNumber,
		Status:           dao.TxStatusSent,
		RoundNumber:      uint64(roundNumber),
		WalletAddress:    player.WalletAddress,
		TemporaryAddress: player.TemporaryAddress,
	})
}

func (c *Chain) cardsOnChain(gameID uint, txHash string, blockHash string, blockNumber uint64, tx *proto.Transaction_CardsOnChain) error {
	player := types.PlayerAddress{}
	player.FromProto(tx.CardsOnChain.Address)
	roundNumber := tx.CardsOnChain.RoundNumber
	salt := tx.CardsOnChain.Salt
	cardsUint := make([]uint, len(tx.CardsOnChain.Cards))
	for i := range cardsUint {
		cardsUint[i] = uint(tx.CardsOnChain.Cards[i])
	}
	cardsOnChainEvent := types.NewEvent(types.CHAIN_MANAGER_ID, &types.PlayerCardsOnChain{
		GameID:      gameID,
		Address:     player,
		RoundNumber: roundNumber,
		Salt:        salt,
		Cards:       cardsUint,
	}, true)
	evtID := cardsOnChainEvent.EventID
	c.inflightEvents[evtID] = struct{}{}
	c.workerManager.SendEvent(fmt.Sprint(gameID), cardsOnChainEvent)
	return db.SaveCardsOnChainTx(&dao.CardsOnChainTx{
		GameID:           gameID,
		ContractAddress:  tx.CardsOnChain.RoomContractAddress,
		TxHash:           txHash,
		BlockHash:        blockHash,
		BlockNumber:      blockNumber,
		Status:           dao.TxStatusSent,
		RoundNumber:      uint64(roundNumber),
		WalletAddress:    player.WalletAddress,
		TemporaryAddress: player.TemporaryAddress,
	})
}
