package game

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/CryptoElementals/common/conversion"
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
	workerManager     *worker.WorkerManager
	chainSvc          ContractClient
	gameResultSettler GameResultSettler
	args              dao.GameArgs
	stopped           bool

	// Event pools for commitment and card submissions
	txPool *txPool
}

// Handle implements worker.EventHandler so GameManager can run as a worker.
// It receives chain-related game events and processes them by rebuilding runtime game state.
func (r *GameManager) Handle(ctx context.Context, event *types.Event) error {
	gameID, ok := gameIDFromEventData(event.Data)
	if !ok {
		return nil
	}
	return r.handleGameEvent(ctx, gameID, event)
}

func gameIDFromEventData(data any) (uint, bool) {
	switch evt := data.(type) {
	case *timerEvent:
		return evt.GameID, true
	case *types.RoomCreated:
		return evt.GameID, true
	case *types.NewTurnSetupComplete:
		return evt.GameID, true
	case *types.PlayerCommitmentOnChain:
		return evt.GameID, true
	case *types.PlayerCardOnChain:
		return evt.GameID, true
	case *types.PlayerReadyEvent:
		return evt.GameId, true
	case *types.SurrenderEvent:
		return evt.GameID, true
	case *types.SubmitPlayerCommitment:
		return evt.GameID, true
	case *types.SubmitPlayerCard:
		return evt.GameID, true
	default:
		return 0, false
	}
}

// handleGameEvent loads latest game state from DB and delegates to an ephemeral Game instance.
func (r *GameManager) handleGameEvent(ctx context.Context, gameID uint, event *types.Event) error {
	if gameID == 0 {
		return fmt.Errorf("handleGameEvent: missing game id")
	}

	gameInfo, err := db.LoadGameByGameID(gameID)
	if err != nil {
		log.Errorw("handleGameEvent: failed to load game from db", "game id", gameID, "err", err)
		return err
	}

	// Build ephemeral runtime Game and let it handle the event synchronously.
	g := NewEphemeralGameForEvent(ctx, r.workerManager, r.txPool, r.gameResultSettler, gameInfo)
	return g.Handle(ctx, event)
}

func NewGameManager(ctx context.Context,
	workerManagerService *worker.WorkerManager,
	gameArgs dao.GameArgs,
	chainSvc ContractClient,
	poolBatchSize int,
) *GameManager {
	m := &GameManager{
		ctx:           ctx,
		workerManager: workerManagerService,
		chainSvc:      chainSvc,
		args:          gameArgs,
	}
	// Set default pool processing interval if not set
	if m.args.PoolProcessingInterval <= 0 {
		m.args.PoolProcessingInterval = 5 // Default 5 seconds
	}
	m.txPool = newTxPool(chainSvc, poolBatchSize)
	return m
}

func (r *GameManager) Start() error {
	// Start background goroutine for pool processing
	go r.txPool.processPools(r.ctx, r.args)
	return nil
}

func (r *GameManager) Stop() {
	r.lock.Lock()
	log.Info("closing game manager")
	r.stopped = true
	r.lock.Unlock()
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
	log.Infow("gameContinue", "game id", gameID, "players", types.ToJsonLoggable(evt.Players))
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
	// DB is the source of truth; treat any active game as "in game".
	_, err := db.GetActiveGameByPlayer(player.Id, player.TemporaryAddress)
	return err == nil
}

func (r *GameManager) GetActiveGame(player types.PlayerAddress) *proto.GameInfo {
	gameInfo, err := db.GetActiveGameByPlayer(player.Id, player.TemporaryAddress)
	if err != nil {
		log.Errorw("GetActiveGame: failed to load game by player", "player", player.String(), "err", err)
		return nil
	}
	return conversion.DbGameInfoToProtoGameInfo(gameInfo)
}

func (r *GameManager) GetGamePhase(address types.PlayerAddress) (*proto.GamePhase, error) {
	gameInfo, err := db.GetActiveGameByPlayer(address.Id, address.TemporaryAddress)
	if err != nil {
		return &proto.GamePhase{
			GameType: proto.GameType_PVP,
		}, nil
	}

	// Rebuild runtime round state from DB
	currentRound := buildRuntimeState(gameInfo)
	if currentRound == nil || currentRound.round == nil {
		return nil, errors.New("no round data for game")
	}

	// Determine turnStartAt like in game_handler.go
	turnNumber := currentRound.getCurrentTurnNumber()
	turnStartAt := int64(0)
	currentTurn := currentRound.getCurrentTurn()
	if currentTurn != nil && currentTurn.TurnStartAt > 0 {
		turnStartAt = currentTurn.TurnStartAt
	} else {
		turnStartAt = currentRound.round.CreatedAt.Unix()
	}

	gamePhase := conversion.DbGameToProtoGamePhase(gameInfo, currentRound.round, turnNumber, turnStartAt)
	return gamePhase, nil
}

// SyncGamePhase sends the current game phase directly to the player worker via workerManager
func (r *GameManager) SyncGamePhase(address types.PlayerAddress) error {
	gamePhase, err := r.GetGamePhase(address)
	if err != nil {
		// Player not in game: send empty phase
		gamePhase = &proto.GamePhase{
			GameType: proto.GameType_PVP,
		}
	}
	syncEvt := types.NewEvent(types.GAME_MANAGER_ID, &types.GamePhaseSyncEvent{
		GamePhase: gamePhase,
	})
	r.workerManager.SendEvent(address.String(), syncEvt)
	return nil
}

// validatePlayersNotInGame checks if any of the players are already in a game
func (r *GameManager) validatePlayersNotInGame(players []types.PlayerAddress) error {
	for _, player := range players {
		if existing, err := db.GetActiveGameByPlayer(player.Id, player.TemporaryAddress); err == nil && existing != nil {
			return fmt.Errorf("player %s already in game, game id: %d", player.String(), existing.ID)
		}
	}
	return nil
}

func (r *GameManager) continueGame(players []types.PlayerAddress) (uint, error) {
	if err := r.validatePlayersNotInGame(players); err != nil {
		return 0, err
	}
	game := NewGame(r.ctx, players, r.workerManager, r.txPool, r.gameResultSettler, &r.args)
	if err := game.pushStateToContractCreating(); err != nil {
		return 0, err
	}
	if err := game.saveGame(); err != nil {
		return 0, err
	}
	return game.gameInfo.ID, nil
}

func (r *GameManager) createGame(players []types.PlayerAddress) (uint, error) {
	game := NewGame(r.ctx, players, r.workerManager, r.txPool, r.gameResultSettler, &r.args)
	if err := game.saveGame(); err != nil {
		return 0, err
	}
	return game.gameInfo.ID, nil
}

// HandleSubmitPlayerCommitment forwards a commitment submission to the game worker for validation and tx pool enqueue
func (r *GameManager) HandleSubmitPlayerCommitment(evt *types.SubmitPlayerCommitment) error {
	return r.forwardToSelfAndAwait(evt)
}

// HandleSubmitPlayerCard receives a card submission and forwards it to the game worker for validation and tx pool enqueue
func (r *GameManager) HandleSubmitPlayerCard(evt *types.SubmitPlayerCard) error {
	return r.forwardToSelfAndAwait(evt)
}

func (r *GameManager) forwardToSelfAndAwait(data any) error {
	ev := types.NewEvent(types.GAME_MANAGER_ID, data, true)
	// Route to game manager worker; it will load from DB and delegate to ephemeral Game.
	r.workerManager.SendEvent(types.GAME_MANAGER_ID, ev)
	_, err := ev.Await()
	return err
}
