package game

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

type GameManager struct {
	ctx             context.Context
	lock            sync.RWMutex
	gamesMap        map[uint]*Game
	playerToGameMap map[types.PlayerAddress]*Game
	workerManager   *worker.WorkerManager
	gameInitialHP   int64
}

func NewGameManager(ctx context.Context, workerMangerService *worker.WorkerManager, initialHP int64) *GameManager {
	m := &GameManager{
		ctx:             ctx,
		gamesMap:        make(map[uint]*Game),
		playerToGameMap: make(map[types.PlayerAddress]*Game),
		workerManager:   workerMangerService,
		gameInitialHP:   initialHP,
	}

	return m
}

func (r *GameManager) Start() error {
	err := r.recoverGames()
	if err != nil {
		return err
	}
	r.createSelf()
	return nil
}

// Handle implements worker.EventHandler.
func (r *GameManager) Handle(ctx context.Context, event *types.Event) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	switch evt := event.Data.(type) {
	case *types.ErrorEvent:
		// just retry
		r.workerManager.SendEvent(evt.OriginalReceiver, evt.OriginalEvent)
		return nil
	case *types.GameMatchedEvent:
		gameID, err := r.createGame(evt.Players)
		if err != nil {
			return err
		}
		// also notify players
		for _, player := range evt.Players {
			r.workerManager.SendEvent(player.String(), types.NewEvent(types.GAME_MANAGER_ID, &types.GameCreatedEvent{
				GameID:  gameID,
				Players: evt.Players,
			}))
		}
		return nil
	default:
		return fmt.Errorf("GameManager Handle err: event type not match, %d", reflect.TypeOf(evt))
	}
}

func (r *GameManager) IsPlayerInGame(player types.PlayerAddress) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	_, ok := r.playerToGameMap[player]
	return ok
}

func (r *GameManager) GetActiveGame(player types.PlayerAddress) *dao.Game {
	r.lock.RLock()
	defer r.lock.RUnlock()
	game, ok := r.playerToGameMap[player]
	if !ok {
		return nil
	}
	return game.gameInfo
}

func (r *GameManager) GetActiveGameByID(id uint) *Game {
	r.lock.RLock()
	defer r.lock.RUnlock()
	game, ok := r.gamesMap[id]
	if !ok {
		return nil
	}
	return game
}

func (r *GameManager) createGame(players []types.PlayerAddress) (uint, error) {
	for _, player := range players {
		if game, ok := r.playerToGameMap[player]; ok {
			return 0, fmt.Errorf("player %s already in game, game id: %d", player.String(), game.gameInfo.ID)
		}
	}
	game := NewGame(r.ctx, players, r.workerManager, r.gameInitialHP)
	err := game.saveGame()
	if err != nil {
		return 0, err
	}
	r.gamesMap[game.gameInfo.ID] = game
	for _, player := range players {
		r.playerToGameMap[player] = game
	}
	game.createSelf()
	return game.gameInfo.ID, nil
}

func (r *GameManager) recoverGames() error {
	gameInfos, err := db.GetAllActiveGames()
	if err != nil {
		return err
	}
	for _, info := range gameInfos {
		game := NewGameFromGameInfo(r.ctx, r.workerManager, info)
		players := game.gamePlayers
		for _, player := range players {
			addr := player.PlayerAddress()
			if _, ok := r.playerToGameMap[addr]; ok {
				log.Fatalf("player %s already in game, game id: %s", addr.String(), game.gameInfo.ID)
			}
			r.playerToGameMap[addr] = game
		}
		r.gamesMap[game.gameInfo.ID] = game
		game.createSelf()
	}
	return nil
}

func (r *GameManager) createSelf() {
	r.workerManager.SpwanWorker(r.ctx, types.GAME_MANAGER_ID, types.WORKER_TYPE_GAME_MANAGER, r)
}
