package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/types"
)

type batchTxEvent struct {
	txs      [][]byte
	blockNum uint64
	blockTx  []byte
	done     chan struct{}
	errChan  chan error
}

type Chain struct {
	ctx            context.Context
	workerManager  *worker.WorkerManager
	sentTxs        map[string]string
	inflightEvents map[string]struct{}
	currentBatchTx *batchTxEvent
}

func NewChain(ctx context.Context, workerManager *worker.WorkerManager) *Chain {
	return &Chain{
		ctx:           ctx,
		workerManager: workerManager,
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
		err = c.createRoomContract(evt.Players)
		if err != nil {
			return err
		}
		log.Debugf("created contract for %s", evt.Players)
	case *types.RequireSetupNewRoundEvent:
		err := c.setRoomReady(evt.ContractAddress)
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

func (c *Chain) createRoomContract(players []types.PlayerAddress) error {
	return nil
}

func (c *Chain) setRoomReady(roomContract string) error {
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
