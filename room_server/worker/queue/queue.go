package queue

import (
	"context"
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
	gameCreator   GameCreator
}

type GameCreator interface {
	HandleGameMatchedEvent(evt *types.GameMatchedEvent) error
}

func NewQueue(ctx context.Context, workerManager *worker.WorkerManager, queueCache cache.Cache, gameCreator GameCreator) *Queue {
	queueCache = cache.WithPrefix(queueInfoPrefix, queueCache)
	q := &Queue{
		ctx:           ctx,
		queue:         make(map[types.PlayerAddress]struct{}),
		workerManager: workerManager,
		queueCache:    queueCache,
		gameCreator:   gameCreator,
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
	return nil
}

func (q *Queue) close() {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.closing = true
}

func (q *Queue) HandleJoinQueueEvent(event *types.JoinQueueEvent) {
	q.lock.Lock()
	defer q.lock.Unlock()
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
		err := q.gameCreator.HandleGameMatchedEvent(evt)
		if err != nil {
			log.Errorf("handle game matched event failed: %s", err.Error())
		}
		delete(q.queue, player)

		// might have some corner case here if failed
		err = q.queueCache.Delete(player.String())
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

func (q *Queue) HandleExitQueueEvent(event *types.ExitQueueEvent) {
	q.lock.Lock()
	defer q.lock.Unlock()
	delete(q.queue, event.PlayerAddress)
	q.queueCache.Delete(event.PlayerAddress.String())
}

// func (w *Queue) handleErrEvent(eventErr *types.ErrorEvent) {
// 	// otherwise we notify the players in events
// 	for _, player := range eventErr.OriginalEvent.Data.(*types.GameMatchedEvent).Players {
// 		w.workerManager.SendEvent(player.String(), types.NewEvent(types.QUEUE_MANAGER_ID, &types.ErrorEvent{
// 			OriginalEvent:    eventErr.OriginalEvent,
// 			OriginalReceiver: types.GAME_MANAGER_ID,
// 			Err:              fmt.Errorf("%w: %s", types.MatchFailedError, eventErr.Err.Error()),
// 		}))
// 	}
// }

func (q *Queue) isPlayerInQueue(address types.PlayerAddress) bool {
	q.lock.RLock()
	defer q.lock.RUnlock()
	_, ok := q.queue[address]
	return ok
}
