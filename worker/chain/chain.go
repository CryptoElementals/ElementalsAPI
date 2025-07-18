package chain

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type batchTxEvent struct {
	txs      [][]byte
	blockNum uint64
	blockTx  []byte
	done     chan struct{}
	errChan  chan error
}

type Chain struct {
	ctx                        context.Context
	workerManager              *worker.WorkerManager
	sentTxs                    map[string]string
	inflightEvents             map[string]struct{}
	currentBatchTx             *batchTxEvent
	client                     *ethclient.Client
	roomManagerContractAddress common.Address
	roundTimeout               *big.Int
	maxRounds                  *big.Int
}

func NewChain(ctx context.Context, workerManager *worker.WorkerManager,
	client *ethclient.Client, roomManagerContractAddressHex string,
	roundTimeout int64, maxRounds int64) *Chain {
	roomManagerContractAddress := common.HexToAddress(roomManagerContractAddressHex)
	roundTimeoutBigInt := big.NewInt(roundTimeout)
	maxRoundsBigInt := big.NewInt(maxRounds)
	return &Chain{
		ctx:                        ctx,
		workerManager:              workerManager,
		sentTxs:                    map[string]string{},
		inflightEvents:             map[string]struct{}{},
		client:                     client,
		roomManagerContractAddress: roomManagerContractAddress,
		roundTimeout:               roundTimeoutBigInt,
		maxRounds:                  maxRoundsBigInt,
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

		err = c.createRoomContract(fmt.Sprintf("%d", evt.GameID), evt.Players)
		if err != nil {
			return err
		}
		log.Debugf("created contract for %s", evt.Players)
	case *types.RequireSetupNewRoundEvent:
		err := c.setRoomReady(fmt.Sprintf("%d", evt.GameID), evt.ContractAddress)
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
	case *types.AckEvent:
		// stale event
		if c.currentBatchTx == nil {
			return nil
		}
		delete(c.inflightEvents, evt.EventID)
		if len(c.inflightEvents) == 0 {
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

func (c *Chain) createRoomContract(gameID string, players []types.PlayerAddress) error {
	roomManagerContract, err := contract.NewRoomManagerContract(c.roomManagerContractAddress, c.client)
	if err != nil {
		return err
	}
	player1WalletAddress := common.HexToAddress(players[0].WalletAddress)
	player2WalletAddress := common.HexToAddress(players[1].WalletAddress)
	player1TemporaryAddress := common.HexToAddress(players[0].TemporaryAddress)
	player2TemporaryAddress := common.HexToAddress(players[1].TemporaryAddress)
	tx, err := roomManagerContract.CreateRoom(&bind.TransactOpts{
		Context: c.ctx,
	}, player1WalletAddress, player2WalletAddress,
		player1TemporaryAddress, player2TemporaryAddress, c.roundTimeout, c.maxRounds)
	if err != nil {
		return err
	}
	c.sentTxs[tx.Hash().String()] = gameID
	return nil
}

func (c *Chain) setRoomReady(gameID string, roomContractHex string) error {
	roomContractAddress := common.HexToAddress(roomContractHex)
	roomContract, err := contract.NewRoomContract(roomContractAddress, c.client)
	if err != nil {
		return err
	}
	tx, err := roomContract.StartANewRound(&bind.TransactOpts{
		Context: c.ctx,
	})
	if err != nil {
		return err
	}
	c.sentTxs[tx.Hash().String()] = gameID
	return nil
}

func (c *Chain) contractCreated(gameID uint, roomContract string) {
	contractCreatedEvt := types.NewEvent(types.CHAIN_MANAGER_ID, &types.RoomContractCreated{
		GameID:              gameID,
		RoomContractAddress: roomContract,
	}, true)
	evtID := contractCreatedEvt.EventID
	c.inflightEvents[evtID] = struct{}{}
	c.workerManager.SendEvent(fmt.Sprint(gameID), contractCreatedEvt)
}

func (c *Chain) roundSetupCompleted(gameID uint, roundNumber uint32) {
	roundSetupCompletedEvent := types.NewEvent(types.CHAIN_MANAGER_ID, &types.NewRoundSetupComplete{
		GameID:      gameID,
		RoundNumber: roundNumber,
	}, true)
	evtID := roundSetupCompletedEvent.EventID
	c.inflightEvents[evtID] = struct{}{}
	c.workerManager.SendEvent(fmt.Sprint(gameID), roundSetupCompletedEvent)
}

func (c *Chain) commitmentOnChain(gameID uint, roundNumber uint32, player types.PlayerAddress, commitment []byte) {
	commitmentOnChainEvent := types.NewEvent(types.CHAIN_MANAGER_ID, &types.PlayerCommitmentOnChain{
		GameID:      gameID,
		Address:     player,
		RoundNumber: roundNumber,
		Commitment:  commitment,
	}, true)
	evtID := commitmentOnChainEvent.EventID
	c.inflightEvents[evtID] = struct{}{}
	c.workerManager.SendEvent(fmt.Sprint(gameID), commitmentOnChainEvent)
}

func (c *Chain) cardsOnChain(gameID uint, roundNumber uint32, player types.PlayerAddress, cards []uint32, salt []byte) {
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
}
