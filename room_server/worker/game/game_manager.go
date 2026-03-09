package game

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
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
	chainSvc          ContractClient
	gameResultSettler GameResultSettler
	args              dao.GameArgs
	noRecover         bool
	stopped           bool
	wg                sync.WaitGroup

	// Event pools for commitment and card submissions
	txPool *txPool
}

func NewGameManager(ctx context.Context,
	workerManagerService *worker.WorkerManager,
	gameArgs dao.GameArgs,
	chainSvc ContractClient,
	noRecover bool,
	poolBatchSize int,
) *GameManager {
	m := &GameManager{
		ctx:             ctx,
		gamesMap:        make(map[uint]*Game),
		playerToGameMap: make(map[types.PlayerAddress]*Game),
		workerManager:   workerManagerService,
		chainSvc:        chainSvc,
		args:            gameArgs,
		noRecover:       noRecover,
	}
	// Set default pool processing interval if not set
	if m.args.PoolProcessingInterval <= 0 {
		m.args.PoolProcessingInterval = 5 // Default 5 seconds
	}
	m.txPool = newTxPool(chainSvc, poolBatchSize)
	return m
}

func (r *GameManager) Start() error {
	err := r.recoverGames()
	if err != nil {
		return err
	}
	// Start background goroutine for pool processing
	r.wg.Add(1)
	go r.txPool.processPools(r.ctx, &r.wg, r.args)
	return nil
}

func (r *GameManager) Stop() {
	r.lock.Lock()
	log.Info("closing game manager")
	for _, game := range r.gamesMap {
		log.Infow("current running game", "game id", game.gameInfo.ID, "status", game.gameInfo.Status, "turn", game.currentRound.turnStatus)
	}
	r.stopped = true
	r.lock.Unlock()
	// wait until all games done
	r.wg.Wait()
	log.Info("game manager closed")
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
			GameID:              gameID,
			Players:             evt.Players,
			IsContinueGame:      true,
			ConfirmationTimeout: r.args.ConfirmationTimeout,
		}))
	}
	log.Infow("gameContinue: gameID %d", gameID, "players", types.ToJsonLoggable(evt.Players))
	return nil
}

func (r *GameManager) HandleGameCompletedEvent(evt *types.GameCompletedEvent) error {
	// Settlement and tx pool clear are done by the Game instance; we only remove the game from maps
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.gamesMap, evt.GameID)
	for _, player := range evt.GameInfo.Players {
		if player == nil {
			continue
		}
		delete(r.playerToGameMap, types.PlayerAddress{
			Id:               player.PlayerId,
			TemporaryAddress: player.TemporaryAddress,
		})
	}
	return nil
}

func (r *GameManager) HandleGameMatchedEvent(evt *types.GameMatchedEvent) (uint, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.stopped {
		return 0, errors.New("server stopping, drop game matched event")
	}
	gameID, err := r.createGame(evt.Players)
	if err != nil {
		return 0, err
	}
	// also notify players
	for _, player := range evt.Players {
		evt := types.NewEvent(types.GAME_MANAGER_ID, &types.GameCreatedEvent{
			GameID:              gameID,
			Players:             evt.Players,
			ConfirmationTimeout: r.args.ConfirmationTimeout,
		})
		r.workerManager.SendEvent(player.String(), evt)
	}
	log.Infow("gameMatched", "game id", gameID, "players", types.ToJsonLoggable(evt.Players))
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
	game, ok := r.playerToGameMap[player]
	r.lock.RUnlock()
	if !ok {
		return nil
	}

	// Create and send request event
	reqEvt := types.NewEvent(types.GAME_MANAGER_ID, &types.GetGameInfoRequest{}, true)
	r.workerManager.SendEvent(game.WorkerID(), reqEvt)

	// Wait for response
	response, err := reqEvt.Await()
	if err != nil {
		log.Errorw("failed to get game info", "err", err, "game id", game.gameInfo.ID)
		return nil
	}

	// Type assert the response
	gameInfo, ok := response.(*proto.GameInfo)
	if !ok {
		log.Errorw("invalid response type for game info", "game id", game.gameInfo.ID, "response type", fmt.Sprintf("%T", response))
		return nil
	}

	return gameInfo
}

func (r *GameManager) GetGamePhase(address types.PlayerAddress) (*proto.GamePhase, error) {
	r.lock.RLock()
	game, ok := r.playerToGameMap[address]
	r.lock.RUnlock()
	if !ok {
		return nil, errors.New("player not in game")
	}

	// Create and send request event
	reqEvt := types.NewEvent(types.GAME_MANAGER_ID, &types.GetGamePhaseRequest{}, true)
	r.workerManager.SendEvent(game.WorkerID(), reqEvt)

	// Wait for response
	response, err := reqEvt.Await()
	if err != nil {
		log.Errorw("failed to get game phase", "err", err, "game id", game.gameInfo.ID)
		return nil, err
	}

	// Type assert the response
	gamePhase, ok := response.(*proto.GamePhase)
	if !ok {
		log.Errorw("invalid response type for game phase", "game id", game.gameInfo.ID, "response type", fmt.Sprintf("%T", response))
		return nil, fmt.Errorf("invalid response type for game phase")
	}

	return gamePhase, nil
}

// SyncGamePhase sends the current game phase directly to the player worker via workerManager
func (r *GameManager) SyncGamePhase(address types.PlayerAddress) error {
	r.lock.RLock()
	game, ok := r.playerToGameMap[address]
	r.lock.RUnlock()
	if !ok {
		// Player not in game, send empty game phase
		gamePhase := &proto.GamePhase{
			GameType: proto.GameType_PVP,
		}
		syncEvt := types.NewEvent(types.GAME_MANAGER_ID, &types.GamePhaseSyncEvent{
			GamePhase: gamePhase,
		})
		r.workerManager.SendEvent(address.String(), syncEvt)
		return nil
	}

	// Send SyncGamePhaseRequest to game worker, which will send game phase directly to player worker
	reqEvt := types.NewEvent(types.GAME_MANAGER_ID, &types.SyncGamePhaseRequest{
		Receiver: &address,
	}, false) // No AckChan needed since game worker sends directly to receiver
	r.workerManager.SendEvent(game.WorkerID(), reqEvt)
	return nil
}

func (r *GameManager) GetBattleInfo(gameID uint, roundNum uint32) (*proto.RoundResult, *proto.GameResult, error) {
	r.lock.RLock()
	game, ok := r.gamesMap[gameID]
	r.lock.RUnlock()
	if !ok {
		return nil, nil, fmt.Errorf("game not found: %d", gameID)
	}

	// Create and send request event
	reqEvt := types.NewEvent(types.GAME_MANAGER_ID, &types.GetBattleInfoRequest{
		RoundNumber: roundNum,
	}, true)
	r.workerManager.SendEvent(game.WorkerID(), reqEvt)

	// Wait for response
	response, err := reqEvt.Await()
	if err != nil {
		log.Errorw("failed to get battle info", "err", err, "game id", gameID, "round num", roundNum)
		return nil, nil, err
	}

	// Type assert the response
	battleInfo, ok := response.(*types.GetBattleInfoResponse)
	if !ok {
		log.Errorw("invalid response type for battle info", "game id", gameID, "round num", roundNum, "response type", fmt.Sprintf("%T", response))
		return nil, nil, fmt.Errorf("invalid response type for battle info")
	}

	if battleInfo.RoundResult == nil {
		return nil, nil, errors.New("round not found")
	}

	return battleInfo.RoundResult, battleInfo.GameResult, nil
}

// registerGame registers a game in the game manager's maps
func (r *GameManager) registerGame(game *Game, players []types.PlayerAddress) {
	r.gamesMap[game.gameInfo.ID] = game
	for _, player := range players {
		r.playerToGameMap[player] = game
	}
	game.createSelf()
}

// validatePlayersNotInGame checks if any of the players are already in a game
func (r *GameManager) validatePlayersNotInGame(players []types.PlayerAddress) error {
	for _, player := range players {
		if game, ok := r.playerToGameMap[player]; ok {
			return fmt.Errorf("player %s already in game, game id: %d", player.String(), game.gameInfo.ID)
		}
	}
	return nil
}

func (r *GameManager) continueGame(players []types.PlayerAddress) (uint, error) {
	if err := r.validatePlayersNotInGame(players); err != nil {
		return 0, err
	}
	game := NewGame(r.ctx, &r.wg, players, r.workerManager, r.txPool, r.gameResultSettler, r, &r.args)
	if err := game.saveGame(); err != nil {
		return 0, err
	}
	if err := game.pushStateToContractCreating(); err != nil {
		return 0, err
	}
	r.registerGame(game, players)
	return game.gameInfo.ID, nil
}

func (r *GameManager) createGame(players []types.PlayerAddress) (uint, error) {
	if err := r.validatePlayersNotInGame(players); err != nil {
		return 0, err
	}
	game := NewGame(r.ctx, &r.wg, players, r.workerManager, r.txPool, r.gameResultSettler, r, &r.args)
	if err := game.saveGame(); err != nil {
		return 0, err
	}
	r.registerGame(game, players)
	return game.gameInfo.ID, nil
}

func (r *GameManager) recoverGames() error {
	if r.noRecover {
		return nil
	}
	gameInfos, err := db.GetAllActiveGames()
	if err != nil {
		return err
	}
	for _, info := range gameInfos {
		game := NewGameFromGameInfo(r.ctx, &r.wg, r.workerManager, r, info, r.txPool, r.gameResultSettler)
		if game == nil {
			continue
		}

		r.lock.Lock()
		for _, player := range game.currentRound.gamePlayers {
			addr := player.PlayerAddress()
			if _, ok := r.playerToGameMap[addr]; ok {
				log.Errorf("player %s already in game, game id: %s", addr.String(), game.gameInfo.ID)
			}
			r.playerToGameMap[addr] = game
		}
		r.gamesMap[game.gameInfo.ID] = game
		r.lock.Unlock()

		game.createSelf()
	}
	return nil
}

// HandleSubmitPlayerCommitment forwards a commitment submission to the game worker for validation and tx pool enqueue
func (r *GameManager) HandleSubmitPlayerCommitment(evt *types.SubmitPlayerCommitment) error {
	// Forward to game worker: it validates and enqueues to tx pool
	ev := types.NewEvent(types.GAME_MANAGER_ID, evt, true)
	r.workerManager.SendEvent(fmt.Sprint(evt.GameID), ev)
	_, err := ev.Await()
	return err
}

// HandleSubmitPlayerCard receives a card submission and forwards it to the game worker for validation and tx pool enqueue
func (r *GameManager) HandleSubmitPlayerCard(evt *types.SubmitPlayerCard) error {
	ev := types.NewEvent(types.GAME_MANAGER_ID, evt, true)
	r.workerManager.SendEvent(fmt.Sprint(evt.GameID), ev)
	_, err := ev.Await()
	return err
}
