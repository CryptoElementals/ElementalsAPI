package chain

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/wallet"
	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/types"
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
	createRoomTxToGameID       map[string]uint
	gameContractToRoomID       map[string]uint
	inflightEvents             map[string]struct{}
	currentBatchTx             *batchTxEvent
	client                     bind.ContractBackend
	roomManagerContractAddress common.Address
	roundTimeout               *big.Int
	maxRounds                  *big.Int

	bindOpts *bind.TransactOpts
}

func NewChain(ctx context.Context, workerManager *worker.WorkerManager, chainID int64,
	client bind.ContractBackend, roomManagerContractAddressHex string, w *wallet.Wallet,
	roundTimeout int64, maxRounds int64) *Chain {
	roomManagerContractAddress := common.HexToAddress(roomManagerContractAddressHex)
	roundTimeoutBigInt := big.NewInt(roundTimeout)
	maxRoundsBigInt := big.NewInt(maxRounds)
	bindOpts := &bind.TransactOpts{
		Context: ctx,
		From:    w.GetAddr(),
		Signer:  w.BuildTxSinger(big.NewInt(chainID)),
	}
	return &Chain{
		ctx:                        ctx,
		workerManager:              workerManager,
		createRoomTxToGameID:       map[string]uint{},
		gameContractToRoomID:       map[string]uint{},
		inflightEvents:             map[string]struct{}{},
		client:                     client,
		roomManagerContractAddress: roomManagerContractAddress,
		roundTimeout:               roundTimeoutBigInt,
		maxRounds:                  maxRoundsBigInt,
		bindOpts:                   bindOpts,
	}
}

func (c *Chain) createSelf() {
	c.workerManager.SpwanWorker(c.ctx, types.CHAIN_MANAGER_ID, types.WORKER_TYPE_CHAIN, c)
}

func (c *Chain) Handle(ctx context.Context, sender worker.EventSender, event *types.Event) error {
	switch evt := event.Data.(type) {
	case *types.RequireContractCreationEvent:
		evt, err := types.AssertInterface[*types.RequireContractCreationEvent](event)
		if err != nil {
			return err
		}

		err = c.createRoomContract(evt.GameID, evt.Players)
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

func (c *Chain) createRoomContract(gameID uint, players []types.PlayerAddress) error {
	roomManagerContract, err := contract.NewRoomManagerContract(c.roomManagerContractAddress, c.client)
	if err != nil {
		return err
	}
	player1WalletAddress := common.HexToAddress(players[0].WalletAddress)
	player2WalletAddress := common.HexToAddress(players[1].WalletAddress)
	player1TemporaryAddress := common.HexToAddress(players[0].TemporaryAddress)
	player2TemporaryAddress := common.HexToAddress(players[1].TemporaryAddress)
	tx, err := roomManagerContract.CreateRoom(c.bindOpts, player1WalletAddress, player2WalletAddress,
		player1TemporaryAddress, player2TemporaryAddress, c.roundTimeout, c.maxRounds)
	if err != nil {
		return err
	}
	txHash := tx.Hash().String()
	c.createRoomTxToGameID[txHash] = gameID
	createRoomTxModel := &dao.CreateRoomTx{
		GameID:       gameID,
		Status:       dao.TxStatusSent,
		TxHash:       txHash,
		RoundTimeout: time.Duration(c.roundTimeout.Int64()) * time.Second,
		MaxRounds:    c.maxRounds.Uint64(),
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
	for _, protoTx := range c.currentBatchTx.txs.Transactions {
		hash := hexutil.Encode(protoTx.TxHash)
		switch tx := protoTx.Tx.(type) {
		case *proto.Transaction_RoomContractCreated:
			gid, ok := c.createRoomTxToGameID[hash]
			if !ok {
				log.Errorf("tx with hash %s not found")
				continue
			}
			c.contractCreated(gid, blockHash, tx)
		case *proto.Transaction_RoomContractSetupReady:
			gid, ok := c.gameContractToRoomID[tx.RoomContractSetupReady.RoomContractAddress]
			if !ok {
				log.Errorf("contract with hash %s not found")
				continue
			}
			c.roundSetupCompleted(gid, blockHash, tx)
		case *proto.Transaction_CommitmentsOnChain:
			gid, ok := c.gameContractToRoomID[tx.CommitmentsOnChain.RoomContractAddress]
			if !ok {
				log.Errorf("contract with hash %s not found")
				continue
			}
			address := types.PlayerAddress{}
			address.FromProto(tx.CommitmentsOnChain.Address)
			c.commitmentOnChain(gid, hash, blockHash, tx)
		case *proto.Transaction_CardsOnChain:
			gid, ok := c.gameContractToRoomID[tx.CardsOnChain.RoomContractAddress]
			if !ok {
				log.Errorf("contract with hash %s not found")
				continue
			}
			address := types.PlayerAddress{}
			address.FromProto(tx.CardsOnChain.Address)
			c.cardsOnChain(gid, hash, blockHash, tx)
		}
	}
}

func (c *Chain) contractCreated(gameID uint, blockHash string, tx *proto.Transaction_RoomContractCreated) error {
	roomContract := tx.RoomContractCreated.RoomContractAddress
	contractCreatedEvt := types.NewEvent(types.CHAIN_MANAGER_ID, &types.RoomContractCreated{
		GameID:              gameID,
		RoomContractAddress: roomContract,
	}, true)
	evtID := contractCreatedEvt.EventID
	c.inflightEvents[evtID] = struct{}{}
	c.workerManager.SendEvent(fmt.Sprint(gameID), contractCreatedEvt)
	c.gameContractToRoomID[roomContract] = gameID
	delete(c.createRoomTxToGameID, roomContract)
	return db.UpdateCreateRoomTxBlockHashAndContractByGameID(gameID, blockHash, roomContract)
}

func (c *Chain) roundSetupCompleted(gameID uint, blockHash string, tx *proto.Transaction_RoomContractSetupReady) error {
	roundNumber := tx.RoomContractSetupReady.RoundNumber
	roundSetupCompletedEvent := types.NewEvent(types.CHAIN_MANAGER_ID, &types.NewRoundSetupComplete{
		GameID:      gameID,
		RoundNumber: roundNumber,
	}, true)
	evtID := roundSetupCompletedEvent.EventID
	c.inflightEvents[evtID] = struct{}{}
	c.workerManager.SendEvent(fmt.Sprint(gameID), roundSetupCompletedEvent)
	return db.UpdateSetRoundReadyTxBlockHashByGameID(gameID, blockHash)
}

func (c *Chain) commitmentOnChain(gameID uint, txHash string, blockHash string, tx *proto.Transaction_CommitmentsOnChain) error {
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
		Status:           dao.TxStatusSent,
		RoundNumber:      uint64(roundNumber),
		WalletAddress:    player.WalletAddress,
		TemporaryAddress: player.TemporaryAddress,
	})
}

func (c *Chain) cardsOnChain(gameID uint, txHash string, blockHash string, tx *proto.Transaction_CardsOnChain) error {
	player := types.PlayerAddress{}
	player.FromProto(tx.CardsOnChain.Address)
	roundNumber := tx.CardsOnChain.RoundNumber
	salt := tx.CardsOnChain.Salt
	cards := tx.CardsOnChain.Cards
	cardsOnChainEvent := types.NewEvent(types.CHAIN_MANAGER_ID, &types.PlayerCardsOnChain{
		GameID:      gameID,
		Address:     player,
		RoundNumber: roundNumber,
		Salt:        salt,
		Cards:       cards,
	}, true)
	evtID := cardsOnChainEvent.EventID
	c.inflightEvents[evtID] = struct{}{}
	c.workerManager.SendEvent(fmt.Sprint(gameID), cardsOnChainEvent)
	return db.SaveCardsOnChainTx(&dao.CardsOnChainTx{
		GameID:           gameID,
		ContractAddress:  tx.CardsOnChain.RoomContractAddress,
		TxHash:           txHash,
		BlockHash:        blockHash,
		Status:           dao.TxStatusSent,
		RoundNumber:      uint64(roundNumber),
		WalletAddress:    player.WalletAddress,
		TemporaryAddress: player.TemporaryAddress,
	})
}
