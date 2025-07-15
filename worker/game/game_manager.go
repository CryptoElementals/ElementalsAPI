package game

import (
	"context"
	"fmt"
	"sync"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/types"
)

type GameManager struct {
	ctx             context.Context
	lock            sync.RWMutex
	gamesMap        map[uint]*Game
	playerToGameMap map[types.PlayerAddress]*Game
	workerManager   *worker.WorkerManager
}

func NewGameManager(ctx context.Context, workerMangerService *worker.WorkerManager) *GameManager {
	m := &GameManager{
		ctx:           ctx,
		gamesMap:      make(map[uint]*Game),
		workerManager: workerMangerService,
	}

	err := m.recoverGames()
	if err != nil {
		log.Errorf("recover games failed: %s", err.Error())
	}
	m.createSelf()
	m.registerGameWorkerFactory()
	return m
}

// Handle implements worker.EventHandler.
func (r *GameManager) Handle(ctx context.Context, event *types.Event) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	switch event.EventType {
	case types.EVENT_TYPE_ERR:
		eventErr := event.Data.(*types.ErrorEvent)
		// just retry
		r.workerManager.SendEvent(eventErr.OriginalReceiver, eventErr.OriginalEvent)
		return nil
	case types.EVENT_TYPE_NEW_GAME:
		evt := event.Data.(*types.NewGameEvent)
		gameID, err := r.createGame(evt.Players)
		if err != nil {
			return err
		}
		// also notify players
		for _, player := range evt.Players {
			r.workerManager.SendEvent(player.String(), types.NewEvent(types.GAME_MANAGER_ID, types.EVENT_TYPE_NEW_GAME, &types.GameCreatedEvent{
				GameID:  gameID,
				Players: evt.Players,
			}))
		}
		return nil
	default:
		return fmt.Errorf("GameManager Handle err: event type not match, %d", event.EventType)
	}
}

func (r *GameManager) IsPlayerInGame(player types.PlayerAddress) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	_, ok := r.playerToGameMap[player]
	return ok
}

func (r *GameManager) GetActiveGame(player types.PlayerAddress) *dao.GameInfo {
	r.lock.RLock()
	defer r.lock.RUnlock()
	game, ok := r.playerToGameMap[player]
	if !ok {
		return nil
	}
	return game.gameInfo
}

func (r *GameManager) createGame(players []types.PlayerAddress) (uint, error) {
	game := NewGame(r.ctx, players, r.workerManager)
	err := game.saveGame()
	if err != nil {
		return 0, err
	}
	game.id = game.gameInfo.ID
	r.gamesMap[game.id] = game
	game.createSelf()
	return game.id, nil
}

func (r *GameManager) recoverGames() error {
	gameInfos, err := db.GetAllActiveGames()
	if err != nil {
		return err
	}
	for _, info := range gameInfos {
		game := NewGame(r.ctx, nil, r.workerManager)
		err := game.recoverGame(info)
		if err != nil {
			return err
		}
		players := game.gamePlayers
		for _, player := range players {
			if _, ok := r.playerToGameMap[player.PlayerAddress()]; ok {
				log.Fatalf("player %s already in game, game id: %s", player.PlayerAddress(), game.id)
			}
			r.playerToGameMap[player.PlayerAddress()] = game
		}
		game.createSelf()
	}
	return nil
}

func (r *GameManager) createSelf() {
	r.workerManager.RegisterWorkerFactory(types.WORKER_TYPE_GAME_MANAGER, func(id string, t worker.WorkerType) *worker.Worker {
		w := worker.NewWorker(r.ctx, id, t)
		w.SetSender(r.workerManager)
		return w
	})
	r.workerManager.SpwanWorker(types.GAME_MANAGER_ID, types.WORKER_TYPE_GAME_MANAGER, r)
}

func (r *GameManager) registerGameWorkerFactory() {
	r.workerManager.RegisterWorkerFactory(types.WORKER_TYPE_GAME, func(id string, t worker.WorkerType) *worker.Worker {
		w := worker.NewWorker(r.ctx, id, t)
		w.SetSender(r.workerManager)
		return w
	})
}
