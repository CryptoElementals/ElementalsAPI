package game

import (
	"context"
	"fmt"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/types"
)

type GameManager struct {
	ctx           context.Context
	gamesMap      map[uint]*Game
	workerManager *worker.WorkerManager
}

func NewGameManager(ctx context.Context, workerMangerService *worker.WorkerManager) *GameManager {
	m := &GameManager{
		ctx:           ctx,
		gamesMap:      make(map[uint]*Game),
		workerManager: workerMangerService,
	}
	m.createSelf()
	m.registerGameWorkerFactory()
	return m
}

func (r *GameManager) CreateGame(players []dao.GamePlayer) (uint, error) {
	game := NewGame(r.ctx, players, r.workerManager)
	err := game.saveGame()
	if err != nil {
		return 0, err
	}
	game.id = game.gameInfo.ID
	r.gamesMap[game.id] = game
	return game.id, nil
}

func (r *GameManager) RecoverGame() error {
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
	}
	return nil
}

// Handle implements worker.EventHandler.
func (r *GameManager) Handle(ctx context.Context, event *types.Event) error {
	switch event.EventType {
	case types.EVENT_TYPE_ERR:
		eventErr := event.Data.(*types.ErrorEvent)
		// just retry
		r.workerManager.SendEvent(eventErr.OriginalReceiver, eventErr.OriginalEvent)
		return nil
	case types.EVENT_TYPE_NEW_GAME:
		evt := event.Data.(*types.NewGameEvent)
		players := make([]dao.GamePlayer, 0, len(evt.Players))
		for _, player := range evt.Players {
			players = append(players, dao.GamePlayer{
				WalletAddress: player.WalletAddress,
				TempAddress:   player.TemporaryAddress,
			})
		}
		gameID, err := r.CreateGame(players)
		if err != nil {
			return err
		}
		// also notify players
		for _, player := range evt.Players {
			r.workerManager.SendEvent(player.String(), types.NewEvent(types.GAME_MANAGER_ID, types.EVENT_TYPE_NEW_GAME, &types.GameCreatedEvent{
				GameID:      gameID,
				GamePlayers: evt.Players,
			}))
		}
		return nil
	default:
		return fmt.Errorf("GameManager Handle err: event type not match, %d", event.EventType)
	}
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
