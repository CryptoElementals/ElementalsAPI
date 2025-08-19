package queue

import (
	"errors"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

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
	if m.continueTimeout != 0 {
		time.AfterFunc(m.continueTimeout, func() {
			m.removeGameByID(game.ID)
		})
	}
}

func (m *continueManager) removeGameByAddress(addr types.PlayerAddress, gameID uint) {
	m.Lock()
	defer m.Unlock()
	gameInfo, ok := m.playerToContinueQueue[addr]
	if !ok {
		return
	}
	if gameID != 0 && gameInfo.gameID != gameID {
		return
	}
	continueMap := m.continueQueue[gameInfo.gameID]
	for player := range continueMap {
		delete(m.playerToContinueQueue, player)
		m.workerManager.SendEvent(player.String(), types.NewEvent(types.QUEUE_MANAGER_ID, &types.ContinueCanceledEvent{
			GameID: gameInfo.gameID,
		}))
	}
	delete(m.continueQueue, gameInfo.gameID)
}

func (m *continueManager) removeGameByID(gameID uint) {
	m.Lock()
	defer m.Unlock()
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

func (m *continueManager) removeGameByIDNoSendEvent(gameID uint) {
	continueMap, ok := m.continueQueue[gameID]
	if !ok {
		return
	}
	for player := range continueMap {
		delete(m.playerToContinueQueue, player)
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
	m.removeGameByIDNoSendEvent(gameID)
	return allPlayers, true, nil
}

func (q *Queue) HandleContinueGameEvent(event *types.PlayerContinueEvent) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	if q.closing {
		log.Debugw("cannot continue game, server is closing", "addr", event.PlayerAddress.String())
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
	err = q.lockTokenForContinue(allPlayers, event.GameId)
	if err != nil {
		log.Infow("lock token for continue failed", "err", err, "game", event.GameId)
		// evt := &types.ContinueCanceledEvent{
		// 	GameID: event.GameId,
		// }
		// // if error, we cancel the continue, no need to throw error to frontend
		// for _, player := range allPlayers {
		// 	q.workerManager.SendEvent(player.String(), types.NewEvent(types.QUEUE_MANAGER_ID, evt))
		// }
		return err
	}
	evt := &types.GameContinueEvent{
		Players: allPlayers,
	}
	return q.callContinue(evt)
}

func (q *Queue) callContinue(evt *types.GameContinueEvent) error {
	err := q.gameCreator.HandleGameContinueEvent(evt)
	if err != nil {
		log.Errorf("handle game continue event failed: %s", err.Error())
		return err
	}
	return nil
}

func (q *Queue) RefuseContinueGame(playerAddress types.PlayerAddress, lastGameID uint) error {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.continueManager.removeGameByAddress(playerAddress, lastGameID)
	return nil
}
