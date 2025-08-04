package game

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type GameManager struct {
	ctx               context.Context
	lock              sync.RWMutex
	gamesMap          map[uint]*Game
	playerToGameMap   map[types.PlayerAddress]*Game
	workerManager     *worker.WorkerManager
	gameInitialHP     int64
	roundTimeout      int64
	maxRounds         int64
	chainSvc          ContractClient
	gameResultSettler GameResultSettler
}

func NewGameManager(ctx context.Context,
	workerManagerService *worker.WorkerManager,
	gameInitialHP int64,
	roundTimeout int64,
	maxRounds int64,
	chainSvc ContractClient) *GameManager {
	m := &GameManager{
		ctx:             ctx,
		gamesMap:        make(map[uint]*Game),
		playerToGameMap: make(map[types.PlayerAddress]*Game),
		workerManager:   workerManagerService,
		gameInitialHP:   gameInitialHP,
		maxRounds:       maxRounds,
		roundTimeout:    roundTimeout,
		chainSvc:        chainSvc,
	}
	return m
}

func (r *GameManager) Start() error {
	err := r.recoverGames()
	if err != nil {
		return err
	}
	return nil
}

func (r *GameManager) HandleGameContinueEvent(evt *types.GameContinueEvent) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	gameID, err := r.continueGame(evt.Players)
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
	log.Infof("gameContinue: gameID %d", gameID)
	return nil
}

func (r *GameManager) HandleGameCompletedEvent(evt *types.GameCompletedEvent) error {
	if r.gameResultSettler != nil {
		err := r.gameResultSettler.GameResultSettlement(evt)
		if err != nil {
			return err
		}
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	game := r.gamesMap[evt.GameID]
	if game == nil {
		return fmt.Errorf("game not found, game id: %d", evt.GameID)
	}
	delete(r.gamesMap, evt.GameID)
	for _, player := range game.gamePlayers {
		delete(r.playerToGameMap, *player.addr)
	}
	return nil
}

func (r *GameManager) HandleGameMatchedEvent(evt *types.GameMatchedEvent) (uint, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	gameID, err := r.createGame(evt.Players)
	if err != nil {
		return 0, err
	}
	// also notify players
	for _, player := range evt.Players {
		evt := types.NewEvent(types.GAME_MANAGER_ID, &types.GameCreatedEvent{
			GameID:  gameID,
			Players: evt.Players,
		})
		r.workerManager.SendEvent(player.String(), evt)
	}
	log.Infof("gameMatched: gameID %d", gameID)
	return gameID, nil
}

func (r *GameManager) IsPlayerInGame(player types.PlayerAddress) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	_, ok := r.playerToGameMap[player]
	return ok
}

func (r *GameManager) GetActiveGame(player types.PlayerAddress) *proto.GameInfo {
	r.lock.RLock()
	defer r.lock.RUnlock()
	game, ok := r.playerToGameMap[player]
	if !ok {
		return nil
	}
	return game.ToProto()
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

func (r *GameManager) GetGamePhase(address types.PlayerAddress) (*proto.GamePhase, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	game, ok := r.playerToGameMap[address]
	if !ok {
		return nil, errors.New("player not in game")
	}
	return game.GetGamePhase(), nil
}

func (r *GameManager) continueGame(players []types.PlayerAddress) (uint, error) {
	for _, player := range players {
		if game, ok := r.playerToGameMap[player]; ok {
			return 0, fmt.Errorf("player %s already in game, game id: %d", player.String(), game.gameInfo.ID)
		}
	}
	game := NewGame(r.ctx, players, r.workerManager, r.chainSvc, r, r.gameInitialHP, r.roundTimeout, r.maxRounds)
	err := game.saveGame()
	if err != nil {
		return 0, err
	}
	err = game.pushStateToContractCreating()
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

func (r *GameManager) createGame(players []types.PlayerAddress) (uint, error) {
	for _, player := range players {
		if game, ok := r.playerToGameMap[player]; ok {
			return 0, fmt.Errorf("player %s already in game, game id: %d", player.String(), game.gameInfo.ID)
		}
	}
	game := NewGame(r.ctx, players, r.workerManager, r.chainSvc, r, r.gameInitialHP, r.roundTimeout, r.maxRounds)
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
	if r.roundTimeout == 0 {
		return nil
	}
	gameInfos, err := db.GetAllActiveGames()
	if err != nil {
		return err
	}
	for _, info := range gameInfos {
		game := NewGameFromGameInfo(r.ctx, r.workerManager, r, info, r.chainSvc)
		if time.Since(info.CreatedAt) > time.Duration(r.roundTimeout)*time.Second*time.Duration(r.maxRounds) {
			log.Errorf("game %d expired, terminate", info.ID)
			err := game.handleRoundEnd()
			if err != nil {
				log.Errorf("expired game terminate failed, game: %d, err %s", info.ID, err)
			}
			continue
		}

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
