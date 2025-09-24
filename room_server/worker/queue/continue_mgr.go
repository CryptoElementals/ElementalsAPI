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
	continueQueue             map[uint]map[types.PlayerAddress]bool
	playerToContinueQueue     map[types.PlayerAddress]*gameContinueInfo
	workerManager             *worker.WorkerManager
	continueTimeout           int64
	continueTimeoutRedundancy int64
	sync.RWMutex
}

func newContinueManager(workerManager *worker.WorkerManager, continueTimeout, continueTimeoutRedundancy int64) *continueManager {
	return &continueManager{
		continueQueue:             make(map[uint]map[types.PlayerAddress]bool),
		playerToContinueQueue:     make(map[types.PlayerAddress]*gameContinueInfo),
		workerManager:             workerManager,
		continueTimeout:           continueTimeout,
		continueTimeoutRedundancy: continueTimeoutRedundancy,
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
		time.AfterFunc(time.Duration(m.continueTimeout+m.continueTimeoutRedundancy)*time.Second, func() {
			if m.removeGameByID(game.ID) {
				log.Infow("continue timeout, game id found", "game id", game.ID)
			}
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

func (m *continueManager) removeGameByID(gameID uint) bool {
	m.Lock()
	defer m.Unlock()
	continueMap, ok := m.continueQueue[gameID]
	if !ok {
		return false
	}
	for player := range continueMap {
		delete(m.playerToContinueQueue, player)
		m.workerManager.SendEvent(player.String(), types.NewEvent(types.QUEUE_MANAGER_ID, &types.ContinueCanceledEvent{
			GameID: gameID,
		}))
	}
	delete(m.continueQueue, gameID)
	return true
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

func (m *continueManager) getPlayerContinueInfo(address types.PlayerAddress) *types.GameContinueInfo {
	m.RLock()
	defer m.RUnlock()
	info, ok := m.playerToContinueQueue[address]
	if !ok {
		return nil
	}
	allPlayers := make([]types.PlayerAddress, 0, len(m.playerToContinueQueue))
	for player := range m.continueQueue[info.gameID] {
		allPlayers = append(allPlayers, player)
	}
	return &types.GameContinueInfo{
		GameID:          info.gameID,
		EndTime:         info.endTime,
		ContinueTimeout: m.continueTimeout,
		Players:         allPlayers,
	}
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
	for _, player := range allPlayers {
		err := q.lockToken(&player)
		if err != nil {
			for _, player := range allPlayers {
				q.workerManager.SendEvent(player.String(), types.NewEvent(types.QUEUE_MANAGER_ID, &types.ContinueCanceledEvent{
					GameID: event.GameId,
				}))
				err = q.unlockToken(&player)
				if err != nil {
					log.Infow("unlock token for continue failed", "err", err, "game", event.GameId, "player", player.String())
				}
			}
			log.Infow("lock token for continue failed", "err", err, "game", event.GameId, "player", player.String())
			return err
		}
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
