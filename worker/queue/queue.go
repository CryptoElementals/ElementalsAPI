package queue

import (
	"context"
	"fmt"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/types"
)

type Queue struct {
	ctx           context.Context
	queue         map[types.PlayerAddress]struct{}
	workerManager *worker.WorkerManager
}

func NewQueue(ctx context.Context, workerManager *worker.WorkerManager) *Queue {
	q := &Queue{
		ctx:           ctx,
		queue:         make(map[types.PlayerAddress]struct{}),
		workerManager: workerManager,
	}
	q.createSelf()
	return q
}

func (q *Queue) Handle(ctx context.Context, event *types.Event) error {
	switch event.EventType {
	case types.EVENT_TYPE_JOIN_QUEUE:
		q.handleJoinQueueEvent(event.Data.(*types.JoinQueueEvent))
	case types.EVENT_TYPE_EXIT_QUEUE:
		q.handleExitQueueEvent(event.Data.(*types.ExitQueueEvent))
	case types.EVENT_TYPE_ERR:
		q.handleErrEvent(event.Data.(*types.ErrorEvent))
	default:
		return fmt.Errorf("queue worker handle event type %d not supported", event.EventType)
	}
	return nil
}

func (q *Queue) handleJoinQueueEvent(event *types.JoinQueueEvent) {
	if _, ok := q.queue[event.PlayerAddress]; ok {
		return
	}
	if len(q.queue) == 0 {
		q.queue[event.PlayerAddress] = struct{}{}
		return
	}
	for player := range q.queue {
		// don't match players with same wallet address
		if player.WalletAddress == event.PlayerAddress.WalletAddress {
			continue
		}
		evt := &types.NewGameEvent{
			MsgSender: types.QUEUE_MANAGER_ID,
			Players:   []types.PlayerAddress{player, event.PlayerAddress},
		}
		q.workerManager.SendEvent(types.GAME_MANAGER_ID, types.NewEvent(types.QUEUE_MANAGER_ID, types.EVENT_TYPE_NEW_GAME, evt))
		delete(q.queue, player)
	}
}

func (q *Queue) handleExitQueueEvent(event *types.ExitQueueEvent) {
	delete(q.queue, event.PlayerAddress)
}

func (w *Queue) handleErrEvent(eventErr *types.ErrorEvent) {
	// we just ignore event receivers if it's not game_manager
	if eventErr.OriginalReceiver != types.GAME_MANAGER_ID {
		log.Errorf("Queue handleErrEvent err: event receiver not match, %d", eventErr.OriginalReceiver)
		return
	}
	// otherwise we notify the players in events
	for _, player := range eventErr.OriginalEvent.Data.(*types.NewGameEvent).Players {
		w.workerManager.SendEvent(player.String(), types.NewEvent(types.QUEUE_MANAGER_ID, types.EVENT_TYPE_ERR, &types.ErrorEvent{
			OriginalEvent:    eventErr.OriginalEvent,
			OriginalReceiver: types.GAME_MANAGER_ID,
			Err:              fmt.Errorf("%w: %s", types.MatchFailedError, eventErr.Err.Error()),
		}))
	}
}

func (q *Queue) createSelf() {
	q.workerManager.RegisterWorkerFactory(types.WORKER_TYPE_QUEUE, func(id string, t worker.WorkerType) *worker.Worker {
		return worker.NewWorker(q.ctx, id, types.WORKER_TYPE_QUEUE)
	})
	q.workerManager.SpwanWorker(types.QUEUE_MANAGER_ID, types.WORKER_TYPE_QUEUE, q)
}
