package queue

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

const queueInfoPrefix = "queue_info:"
const pendingContinuePrefix = "pending_continue:"
const queueInfoVal = "v"

type Queue struct {
	ctx           context.Context
	lock          sync.RWMutex
	queue         map[types.PlayerAddress]struct{}
	workerManager *worker.WorkerManager
	queueCache    cache.Cache
	closing       bool
}

func NewQueue(ctx context.Context, workerManager *worker.WorkerManager, queueCache cache.Cache) *Queue {
	queueCache = cache.WithPrefix(queueInfoPrefix, queueCache)
	q := &Queue{
		ctx:           ctx,
		queue:         make(map[types.PlayerAddress]struct{}),
		workerManager: workerManager,
		queueCache:    queueCache,
	}
	return q
}

func (q *Queue) start() error {
	keys, err := q.queueCache.List("")
	if err != nil {
		return err
	}
	for _, key := range keys {
		var player types.PlayerAddress
		if err := player.Parse(key); err != nil {
			return err
		}
		q.queue[player] = struct{}{}
	}
	q.createSelf()
	return nil
}

func (q *Queue) close() {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.closing = true
}

func (q *Queue) Handle(ctx context.Context, event *types.Event) error {
	switch evt := event.Data.(type) {
	case *types.JoinQueueEvent:
		if q.closing {
			return errors.New("server is closing, can not join queue")
		}
		q.handleJoinQueueEvent(evt)
	case *types.ExitQueueEvent:
		q.handleExitQueueEvent(evt)
	case *types.ErrorEvent:
		q.handleErrEvent(evt)
	default:
		return fmt.Errorf("queue worker handle event type %d not supported", reflect.TypeOf(evt))
	}
	return nil
}

func (q *Queue) handleJoinQueueEvent(event *types.JoinQueueEvent) {
	if _, ok := q.queue[event.PlayerAddress]; ok {
		return
	}
	matched := false
	for player := range q.queue {
		// don't match players with same wallet address
		if player.WalletAddress == event.PlayerAddress.WalletAddress {
			continue
		}
		if player.TemporaryAddress == event.PlayerAddress.TemporaryAddress {
			continue
		}
		evt := &types.GameMatchedEvent{
			Players: []types.PlayerAddress{player, event.PlayerAddress},
		}
		q.workerManager.SendEvent(types.GAME_MANAGER_ID, types.NewEvent(types.QUEUE_MANAGER_ID, evt))
		delete(q.queue, player)

		// might have some corner case here if failed
		err := q.queueCache.Delete(player.String())
		if err != nil {
			log.Errorf("delete player from queue cache failed: %s", err.Error())
		}
		matched = true
	}
	if !matched {
		q.queue[event.PlayerAddress] = struct{}{}
		err := q.queueCache.Set(event.PlayerAddress.String(), queueInfoVal, 0)
		if err != nil {
			log.Errorf("set player to queue cache failed: %s", err.Error())
		}
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
	for _, player := range eventErr.OriginalEvent.Data.(*types.GameMatchedEvent).Players {
		w.workerManager.SendEvent(player.String(), types.NewEvent(types.QUEUE_MANAGER_ID, &types.ErrorEvent{
			OriginalEvent:    eventErr.OriginalEvent,
			OriginalReceiver: types.GAME_MANAGER_ID,
			Err:              fmt.Errorf("%w: %s", types.MatchFailedError, eventErr.Err.Error()),
		}))
	}
}

func (q *Queue) createSelf() {
	q.workerManager.SpwanWorker(q.ctx, types.QUEUE_MANAGER_ID, types.WORKER_TYPE_QUEUE, q)
}

func (q *Queue) isPlayerInQueue(address types.PlayerAddress) bool {
	q.lock.RLock()
	defer q.lock.RUnlock()
	_, ok := q.queue[address]
	return ok
}
