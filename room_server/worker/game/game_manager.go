package game

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

// eventKey uniquely identifies an event by address, round number, and index
type eventKey struct {
	address     string
	roundNumber uint32
	index       uint32
}

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
	commitmentPool     map[eventKey]*types.SubmitPlayerCommitment
	cardPool           map[eventKey]*types.SubmitPlayerCard
	commitmentInFlight map[eventKey]bool
	cardInFlight       map[eventKey]bool
	poolLock           sync.RWMutex
}

func NewGameManager(ctx context.Context,
	workerManagerService *worker.WorkerManager,
	gameArgs dao.GameArgs,
	chainSvc ContractClient,
	noRecover bool,
) *GameManager {
	m := &GameManager{
		ctx:                ctx,
		gamesMap:           make(map[uint]*Game),
		playerToGameMap:    make(map[types.PlayerAddress]*Game),
		workerManager:      workerManagerService,
		chainSvc:           chainSvc,
		args:               gameArgs,
		noRecover:          noRecover,
		commitmentPool:     make(map[eventKey]*types.SubmitPlayerCommitment),
		cardPool:           make(map[eventKey]*types.SubmitPlayerCard),
		commitmentInFlight: make(map[eventKey]bool),
		cardInFlight:       make(map[eventKey]bool),
	}
	// Set default pool processing interval if not set
	if m.args.PoolProcessingInterval <= 0 {
		m.args.PoolProcessingInterval = 5 // Default 5 seconds
	}
	return m
}

func (r *GameManager) Start() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	err := r.recoverGames()
	if err != nil {
		return err
	}
	// Start background goroutine for pool processing
	r.wg.Add(1)
	go r.processPools()
	return nil
}

func (r *GameManager) Stop() {
	r.lock.Lock()
	log.Info("closing game manager")
	for _, game := range r.gamesMap {
		log.Infow("current running game", "game id", game.gameInfo.ID, "status", game.gameInfo.Status, "round", game.currentRound.round.Status)
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
			GameID:         gameID,
			Players:        evt.Players,
			IsContinueGame: true,
		}))
	}
	log.Infow("gameContinue: gameID %d", gameID, "players", types.ToJsonLoggable(evt.Players))
	return nil
}

func (r *GameManager) HandleGameCompletedEvent(evt *types.GameCompletedEvent) error {
	if r.gameResultSettler != nil {
		err := r.gameResultSettler.GameResultSettlement(evt)
		if err != nil {
			return err
		}
	}
	// do this async for not getting deadlock
	go func() {
		r.lock.Lock()
		defer r.lock.Unlock()
		game := r.gamesMap[evt.GameID]
		if game == nil {
			// stale event or server bootstrap event
			log.Errorf("game not found, game id: %d", evt.GameID)
			return
		}
		delete(r.gamesMap, evt.GameID)
		for _, player := range game.currentRound.gamePlayers {
			if player == nil {
				continue
			}
			delete(r.playerToGameMap, player.PlayerAddress())
		}
	}()
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
			GameID:  gameID,
			Players: evt.Players,
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
			PvPInfo: &proto.PvPInfo{
				Status: proto.PlayerStatus_PLAYER_UNKNOWN,
			},
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
	game := NewGame(r.ctx, &r.wg, players, r.workerManager, r.chainSvc, r, &r.args)
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
	game := NewGame(r.ctx, &r.wg, players, r.workerManager, r.chainSvc, r, &r.args)
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
		game := NewGameFromGameInfo(r.ctx, &r.wg, r.workerManager, r, info, r.chainSvc)
		if game == nil {
			continue
		}

		for _, player := range game.currentRound.gamePlayers {
			addr := player.PlayerAddress()
			if _, ok := r.playerToGameMap[addr]; ok {
				log.Errorf("player %s already in game, game id: %s", addr.String(), game.gameInfo.ID)
			}
			r.playerToGameMap[addr] = game
		}
		r.gamesMap[game.gameInfo.ID] = game
		game.createSelf()
	}
	return nil
}

// makeEventKey creates an eventKey from address, round number, and index
func makeEventKey(address types.PlayerAddress, roundNumber, index uint32) eventKey {
	return eventKey{
		address:     address.TemporaryAddress,
		roundNumber: roundNumber,
		index:       index,
	}
}

// getGameForPlayer gets the game for a player address
func (r *GameManager) getGameForPlayer(address types.PlayerAddress) (*Game, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	game, ok := r.playerToGameMap[address]
	if !ok {
		return nil, fmt.Errorf("player not in game")
	}
	return game, nil
}

// HandleSubmitPlayerCommitment receives and validates a commitment submission event
func (r *GameManager) HandleSubmitPlayerCommitment(evt *types.SubmitPlayerCommitment) error {
	// Validate GameID is set
	if evt.GameID == 0 {
		return fmt.Errorf("GameID is required in SubmitPlayerCommitment")
	}

	game, err := r.getGameForPlayer(evt.Address)
	if err != nil {
		return err
	}

	// Validate GameID matches the game
	if game.gameInfo.ID != evt.GameID {
		return fmt.Errorf("GameID mismatch: event has %d but game has %d", evt.GameID, game.gameInfo.ID)
	}

	key := makeEventKey(evt.Address, evt.RoundNumber, evt.CommitmentIndex)
	r.poolLock.Lock()
	defer r.poolLock.Unlock()

	if _, exists := r.commitmentPool[key]; exists {
		return fmt.Errorf("commitment already in pool")
	}
	if r.commitmentInFlight[key] {
		return fmt.Errorf("commitment already in flight")
	}

	// Send event to game worker to validate
	validateEvt := types.NewEvent(types.GAME_MANAGER_ID, evt, true)
	r.workerManager.SendEvent(game.WorkerID(), validateEvt)
	_, err = validateEvt.Await()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	r.commitmentPool[key] = evt
	return nil
}

// HandleSubmitPlayerCard receives and validates a card submission event
func (r *GameManager) HandleSubmitPlayerCard(evt *types.SubmitPlayerCard) error {
	// Validate GameID is set
	if evt.GameID == 0 {
		return fmt.Errorf("GameID is required in SubmitPlayerCard")
	}

	game, err := r.getGameForPlayer(evt.Address)
	if err != nil {
		return err
	}

	// Validate GameID matches the game
	if game.gameInfo.ID != evt.GameID {
		return fmt.Errorf("GameID mismatch: event has %d but game has %d", evt.GameID, game.gameInfo.ID)
	}

	key := makeEventKey(evt.Address, evt.RoundNumber, evt.CardIndex)
	r.poolLock.Lock()
	defer r.poolLock.Unlock()

	if _, exists := r.cardPool[key]; exists {
		return fmt.Errorf("card already in pool")
	}
	if r.cardInFlight[key] {
		return fmt.Errorf("card already in flight")
	}

	// Send event to game worker to validate
	validateEvt := types.NewEvent(types.GAME_MANAGER_ID, evt, true)
	r.workerManager.SendEvent(game.WorkerID(), validateEvt)
	_, err = validateEvt.Await()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	r.cardPool[key] = evt
	return nil
}

// processPools periodically processes events in the pools and sends them to chain manager
func (r *GameManager) processPools() {
	defer r.wg.Done()
	ticker := time.NewTicker(time.Duration(r.args.PoolProcessingInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.processCommitmentPool()
			r.processCardPool()
		}
	}
}

// processCommitmentPool processes commitment pool and sends non-inflight events to chain manager in batch
func (r *GameManager) processCommitmentPool() {
	// Collect events to process while holding lock
	r.poolLock.Lock()
	batchEvents := make([]*types.SubmitPlayerCommitment, 0, len(r.commitmentPool))
	eventKeys := make([]eventKey, 0, len(r.commitmentPool))
	for key, evt := range r.commitmentPool {
		if !r.commitmentInFlight[key] && evt.GameID != 0 {
			r.commitmentInFlight[key] = true
			batchEvents = append(batchEvents, evt)
			eventKeys = append(eventKeys, key)
		} else if evt.GameID == 0 {
			log.Errorw("commitment event missing GameID", "address", evt.Address.TemporaryAddress)
		}
	}
	r.poolLock.Unlock()

	if len(batchEvents) == 0 {
		return
	}

	// Submit batch
	if err := r.chainSvc.SubmitPlayerCommitmentsBatch(batchEvents); err != nil {
		log.Errorw("failed to submit commitments batch to chain", "error", err, "count", len(batchEvents))
		// Remove all from in flight on error so they can be retried
		r.poolLock.Lock()
		for _, key := range eventKeys {
			delete(r.commitmentInFlight, key)
		}
		r.poolLock.Unlock()
		return
	}

	log.Infow("submitted commitments batch to chain", "count", len(batchEvents))
}

// processCardPool processes card pool and sends non-inflight events to chain manager in batch
func (r *GameManager) processCardPool() {
	// Collect events to process while holding lock
	r.poolLock.Lock()
	batchEvents := make([]*types.SubmitPlayerCard, 0, len(r.cardPool))
	eventKeys := make([]eventKey, 0, len(r.cardPool))
	for key, evt := range r.cardPool {
		if !r.cardInFlight[key] && evt.GameID != 0 {
			r.cardInFlight[key] = true
			batchEvents = append(batchEvents, evt)
			eventKeys = append(eventKeys, key)
		} else if evt.GameID == 0 {
			log.Errorw("card event missing GameID", "address", evt.Address.TemporaryAddress)
		}
	}
	r.poolLock.Unlock()

	if len(batchEvents) == 0 {
		return
	}

	// Submit batch
	if err := r.chainSvc.SubmitPlayerCardsBatch(batchEvents); err != nil {
		log.Errorw("failed to submit cards batch to chain", "error", err, "count", len(batchEvents))
		// Remove all from in flight on error so they can be retried
		r.poolLock.Lock()
		for _, key := range eventKeys {
			delete(r.cardInFlight, key)
		}
		r.poolLock.Unlock()
		return
	}

	log.Infow("submitted cards batch to chain", "count", len(batchEvents))
}

// OnCommitmentSubmitted is called by chain manager when commitment is successfully submitted
func (r *GameManager) OnCommitmentSubmitted(address types.PlayerAddress, roundNumber, commitmentIndex uint32) {
	key := makeEventKey(address, roundNumber, commitmentIndex)
	r.poolLock.Lock()
	defer r.poolLock.Unlock()
	delete(r.commitmentPool, key)
	delete(r.commitmentInFlight, key)
	log.Infow("commitment confirmed on chain", "address", address.TemporaryAddress, "round", roundNumber, "index", commitmentIndex)
}

// OnCardSubmitted is called by chain manager when card is successfully submitted
func (r *GameManager) OnCardSubmitted(address types.PlayerAddress, roundNumber, cardIndex uint32) {
	key := makeEventKey(address, roundNumber, cardIndex)
	r.poolLock.Lock()
	defer r.poolLock.Unlock()
	delete(r.cardPool, key)
	delete(r.cardInFlight, key)
	log.Infow("card confirmed on chain", "address", address.TemporaryAddress, "round", roundNumber, "index", cardIndex)
}
