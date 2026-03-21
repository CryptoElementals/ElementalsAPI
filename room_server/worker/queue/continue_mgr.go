package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/timer"
	"google.golang.org/protobuf/types/known/emptypb"
)

// EventPublisher publishes outbound player notifications (same contract as game.Publisher).
type EventPublisher interface {
	Publish(ctx context.Context, req *proto.PublishRequest) (*proto.PublishResponse, error)
}

func notifyContinueCanceled(ctx context.Context, pub EventPublisher, player types.PlayerAddress) {
	_, err := pub.Publish(ctx, &proto.PublishRequest{
		Topic: (&player).String(),
		Event: &proto.Event{
			Type: proto.EventType_TYPE_CONTINUE_CANCELED,
			Event: &proto.Event_X{
				X: &emptypb.Empty{},
			},
		},
	})
	if err != nil {
		log.Errorw("notifyContinueCanceled publish failed", "player", (&player).String(), "err", err)
	}
}

type gameContinueInfo struct {
	gameID  uint
	endTime time.Time
}

type continueManager struct {
	continueQueue             map[uint]map[types.PlayerAddress]bool
	playerToContinueQueue     map[types.PlayerAddress]*gameContinueInfo
	workerManager             *worker.WorkerManager
	publisher                 EventPublisher
	ctx                       context.Context
	continueTimeout           int64
	continueTimeoutRedundancy int64
	sync.RWMutex
}

func newContinueManager(ctx context.Context, workerManager *worker.WorkerManager, pub EventPublisher, continueTimeout, continueTimeoutRedundancy int64) *continueManager {
	m := &continueManager{
		continueQueue:             make(map[uint]map[types.PlayerAddress]bool),
		playerToContinueQueue:     make(map[types.PlayerAddress]*gameContinueInfo),
		workerManager:             workerManager,
		publisher:                 pub,
		ctx:                       ctx,
		continueTimeout:           continueTimeout,
		continueTimeoutRedundancy: continueTimeoutRedundancy,
	}
	m.registerTimerHandler()
	return m
}

// continueTimeoutEvent implements timer.TimerEvent for the continue-game timeout.
type continueTimeoutEvent struct {
	GameID uint `json:"game_id"`
}

func (e *continueTimeoutEvent) EventType() string { return "continue_timeout" }

func (e *continueTimeoutEvent) Marshal() []byte {
	data, _ := json.Marshal(e)
	return data
}

func (e *continueTimeoutEvent) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *continueTimeoutEvent) String() string {
	return fmt.Sprintf("continue_timeout{game=%d}", e.GameID)
}

// registerTimerHandler registers the continue-timeout handler with the global timer package.
func (m *continueManager) registerTimerHandler() {
	timer.RegisterHandler(&continueTimeoutEvent{}, func(evt timer.TimerEvent) error {
		te := evt.(*continueTimeoutEvent)
		if m.removeGameByID(te.GameID) {
			log.Infow("continue timeout, game id found", "game id", te.GameID)
		}
		return nil
	})
}

func (m *continueManager) addGame(game *dao.Game) {
	m.Lock()
	defer m.Unlock()
	continuePlayers := make(map[types.PlayerAddress]bool)
	for _, player := range game.Players {
		playerAddr := types.NewPlayerAddress(player.PlayerId, player.TemporaryAddress)
		continuePlayers[*playerAddr] = false
		info := gameContinueInfo{
			gameID:  game.ID,
			endTime: time.Now(),
		}
		m.playerToContinueQueue[*playerAddr] = &info
	}
	m.continueQueue[game.ID] = continuePlayers
	if m.continueTimeout != 0 {
		timeout := time.Duration(m.continueTimeout+m.continueTimeoutRedundancy) * time.Second
		if err := timer.ProcessIn(timeout, &continueTimeoutEvent{GameID: game.ID}); err != nil {
			log.Errorw("schedule continue timeout failed", "game id", game.ID, "err", err)
		}
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
		notifyContinueCanceled(m.ctx, m.publisher, player)
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
		notifyContinueCanceled(m.ctx, m.publisher, player)
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

func (q *Queue) HandleContinueGameEvent(req *proto.ContinueGameRequest) error {
	var address types.PlayerAddress
	address.FromProto(req.Player)
	gameID := uint(req.LastGameID)

	q.lock.Lock()
	defer q.lock.Unlock()
	if q.closing {
		log.Debugw("cannot continue game, server is closing", "addr", address.String())
		return errors.New("queue is closing")
	}

	_, ok := q.queue[address]
	if ok {
		return errors.New("player is in queue")
	}

	allPlayers, ok, err := q.continueManager.handlePlayerGameContinue(address, gameID)
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
				notifyContinueCanceled(q.ctx, q.publisher, player)
				err = q.unlockToken(&player)
				if err != nil {
					log.Infow("unlock token for continue failed", "err", err, "game", gameID, "player", player.String())
				}
			}
			log.Infow("lock token for continue failed", "err", err, "game", gameID, "player", player.String())
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
