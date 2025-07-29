package queue

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

const queueInfoPrefix = "queue_info:"
const queueInfoVal = "v"

type Queue struct {
	ctx             context.Context
	lock            sync.RWMutex
	queue           map[types.PlayerAddress]struct{}
	continueManager *continueManager
	workerManager   *worker.WorkerManager
	queueCache      cache.Cache
	closing         bool
	gameCreator     GameCreator
	continueTimeout time.Duration
}

type gameContinueInfo struct {
	gameID  uint
	endTime time.Time
}

type continueManager struct {
	continueQueue         map[uint]map[types.PlayerAddress]bool
	playerToContinueQueue map[types.PlayerAddress]*gameContinueInfo
	workerManager         *worker.WorkerManager
	continueTimeout       time.Duration
	sync.RWMutex
}

func newContinueManager(workerManager *worker.WorkerManager, continueTimeout time.Duration) *continueManager {
	return &continueManager{
		continueQueue:         make(map[uint]map[types.PlayerAddress]bool),
		playerToContinueQueue: make(map[types.PlayerAddress]*gameContinueInfo),
		workerManager:         workerManager,
		continueTimeout:       continueTimeout,
	}
}

func (m *continueManager) addGame(game *dao.Game) {
	m.Lock()
	defer m.Unlock()
	continuePlayers := make(map[types.PlayerAddress]bool)
	for _, player := range game.Players {
		playerAddr := types.NewPlayerAddress(player.WalletAddress, player.TemporaryAddress)
		continuePlayers[*playerAddr] = false
		info := gameContinueInfo{
			gameID:  game.ID,
			endTime: time.Now(),
		}
		m.playerToContinueQueue[*playerAddr] = &info
	}
	m.continueQueue[game.ID] = continuePlayers
	time.AfterFunc(m.continueTimeout, func() {
		m.removeGameByID(game.ID)
	})
}

func (m *continueManager) removeGameByAddress(addr types.PlayerAddress) {
	m.Lock()
	defer m.Unlock()
	gameInfo, ok := m.playerToContinueQueue[addr]
	if !ok {
		return
	}
	continueMap := m.continueQueue[gameInfo.gameID]
	for player := range continueMap {
		delete(m.playerToContinueQueue, player)
		if player != addr {
			m.workerManager.SendEvent(player.String(), types.NewEvent(types.QUEUE_MANAGER_ID, &types.ContinueCanceledEvent{
				GameID: gameInfo.gameID,
			}))
		}
	}
	delete(m.continueQueue, gameInfo.gameID)
}

func (m *continueManager) removeGameByID(gameID uint) {
	m.Lock()
	defer m.Unlock()
	m.removeGameByIDNoLock(gameID)
}

func (m *continueManager) removeGameByIDNoLock(gameID uint) {
	continueMap, ok := m.continueQueue[gameID]
	if !ok {
		return
	}
	for player := range continueMap {
		delete(m.playerToContinueQueue, player)
		m.workerManager.SendEvent(player.String(), types.NewEvent(types.QUEUE_MANAGER_ID, &types.ContinueCanceledEvent{
			GameID: gameID,
		}))
	}
	delete(m.continueQueue, gameID)
}

// if game ready, return all players in the game, and purge continue info
func (m *continueManager) handlePlayerGameContinue(playerAddr types.PlayerAddress, gameID uint) ([]types.PlayerAddress, bool, error) {
	m.Lock()
	defer m.Unlock()
	gameInfo, ok := m.playerToContinueQueue[playerAddr]
	if !ok || gameInfo == nil {
		return nil, false, errors.New("player not in continue queue")
	}

	if gameInfo.gameID != gameID {
		return nil, false, errors.New("game id not match")
	}
	continueMap := m.continueQueue[gameInfo.gameID]
	continueMap[playerAddr] = true
	var allPlayers []types.PlayerAddress
	for player, ok := range continueMap {
		if !ok {
			return nil, false, nil
		}
		allPlayers = append(allPlayers, player)
	}
	m.removeGameByIDNoLock(gameID)
	return allPlayers, true, nil
}

type GameCreator interface {
	HandleGameMatchedEvent(evt *types.GameMatchedEvent) (uint, error)
	HandleGameContinueEvent(evt *types.GameContinueEvent) error
}

func NewQueue(ctx context.Context, workerManager *worker.WorkerManager, c cache.Cache, gameCreator GameCreator) *Queue {
	queueCache := cache.WithPrefix(queueInfoPrefix, c)
	q := &Queue{
		ctx:           ctx,
		queue:         make(map[types.PlayerAddress]struct{}),
		workerManager: workerManager,
		queueCache:    queueCache,
		gameCreator:   gameCreator,
		// set it to one minute for now
		continueManager: newContinueManager(workerManager, time.Minute),
		continueTimeout: time.Minute,
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

func (q *Queue) HandleJoinQueueEvent(event *types.JoinQueueEvent) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	if _, ok := q.queue[event.PlayerAddress]; ok {
		return errors.New("player already in queue")
	}
	// delete player from continue queue anyway
	q.continueManager.removeGameByAddress(event.PlayerAddress)
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
		_, err := q.gameCreator.HandleGameMatchedEvent(evt)
		if err != nil {
			log.Errorf("handle game matched event failed: %s", err.Error())
			return err
		}
		delete(q.queue, player)

		// might have some corner case here if failed
		err = q.queueCache.Delete(player.String())
		if err != nil {
			log.Errorf("delete player from queue cache failed: %s", err.Error())
			return err
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
	return nil
}

func (q *Queue) HandleContinueGameEvent(event *types.PlayerContinueEvent) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	if q.closing {
		return errors.New("queue is closing")
	}

	_, ok := q.queue[event.PlayerAddress]
	if ok {
		return errors.New("player is in queue")
	}

	allPlayers, ok, err := q.continueManager.handlePlayerGameContinue(event.PlayerAddress, event.GameId)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	evt := &types.GameContinueEvent{
		Players: allPlayers,
	}
	err = q.gameCreator.HandleGameContinueEvent(evt)
	if err != nil {
		log.Errorf("handle game continue event failed: %s", err.Error())
		return err
	}
	return nil
}

func (q *Queue) HandleExitQueueEvent(event *types.ExitQueueEvent) {
	q.lock.Lock()
	defer q.lock.Unlock()
	delete(q.queue, event.PlayerAddress)
	q.queueCache.Delete(event.PlayerAddress.String())
}

func (q *Queue) GameResultSettlement(event *types.GameCompletedEvent) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	err := db.BattleResultSettlement(event.GameInfo)
	if err != nil {
		log.Error("BattleResultSettlement failed, err: ", err)
		return err
	}
	q.continueManager.addGame(event.GameInfo)
	return nil
}

func (q *Queue) isPlayerInQueue(address types.PlayerAddress) bool {
	q.lock.RLock()
	defer q.lock.RUnlock()
	_, ok := q.queue[address]
	return ok
}
