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
	gameLocksGuard    sync.Mutex
	gameLocks         map[uint]*sync.Mutex
	workerManager     *worker.WorkerManager
	publisher         Publisher
	chainSvc          ContractClient
	gameResultSettler GameResultSettler
	args              dao.GameArgs
	stopped           bool

	// Event pools for commitment and card submissions
	txPool *txPool
}

func (r *GameManager) getGameLock(gameID uint) *sync.Mutex {
	r.gameLocksGuard.Lock()
	defer r.gameLocksGuard.Unlock()
	lock, ok := r.gameLocks[gameID]
	if !ok {
		lock = &sync.Mutex{}
		r.gameLocks[gameID] = lock
	}
	return lock
}

func (r *GameManager) withGameLock(gameID uint, fn func() error) error {
	lock := r.getGameLock(gameID)
	lock.Lock()
	defer lock.Unlock()
	return fn()
}

func (r *GameManager) executeOnGame(gameID uint, handler func(g *Game) error) error {
	if gameID == 0 {
		return fmt.Errorf("executeOnGame: missing game id")
	}
	return r.withGameLock(gameID, func() error {
		gameInfo, err := db.LoadGameByGameID(gameID)
		if err != nil {
			log.Errorw("executeOnGame: failed to load game from db", "game id", gameID, "err", err)
			return err
		}
		g := NewEphemeralGameForEvent(r.ctx, r.workerManager, r.publisher, r.txPool, r.gameResultSettler, gameInfo)
		return handler(g)
	})
}

func (r *GameManager) HandleConfirmBattle(req *proto.ConfirmBattleRequest) error {
	return r.executeOnGame(uint(req.GameID), func(g *Game) error { return g.handleConfirmBattle(req) })
}

func (r *GameManager) HandleSurrender(req *proto.SurrenderRequest) error {
	return r.executeOnGame(uint(req.GameID), func(g *Game) error { return g.handleSurrender(req) })
}

func (r *GameManager) HandleTimerEvent(ctx context.Context, evt *timerEvent) error {
	return r.executeOnGame(evt.GameID, func(g *Game) error { g.handleTimerEvent(evt); return nil })
}

func (r *GameManager) HandleSubmitPlayerCommitment(req *proto.SubmitPlayerCommitmentRequest) error {
	if err := validateSubmitPlayerCommitmentRequest(req); err != nil {
		return err
	}
	return r.executeOnGame(uint(req.GameID), func(g *Game) error { return g.handleSubmitPlayerCommitment(req) })
}

func (r *GameManager) HandleSubmitPlayerCard(req *proto.SubmitPlayerCardRequest) error {
	if err := validateSubmitPlayerCardRequest(req); err != nil {
		return err
	}
	return r.executeOnGame(uint(req.GameID), func(g *Game) error { return g.handleSubmitPlayerCard(req) })
}

func NewGameManager(ctx context.Context,
	workerManagerService *worker.WorkerManager,
	pub Publisher,
	gameArgs dao.GameArgs,
	chainSvc ContractClient,
	poolBatchSize int,
) *GameManager {
	m := &GameManager{
		ctx:           ctx,
		gameLocks:     make(map[uint]*sync.Mutex),
		workerManager: workerManagerService,
		publisher:     pub,
		chainSvc:      chainSvc,
		args:          gameArgs,
	}
	if m.args.MaxTurnsPerRound <= 0 {
		m.args.MaxTurnsPerRound = 3
	}
	m.txPool = newTxPool(chainSvc, poolBatchSize)
	return m
}

func (r *GameManager) Start() error {
	r.registerTimerFunction()

	// Start background goroutine for pool processing
	go r.txPool.processPools(r.ctx)

	// On startup, abort any games that were left active (non-ended/non-aborted).
	// This matches the "stateless" model: we do not attempt to recover/resume games after restart.
	games, err := db.GetAllActiveGames()
	if err != nil {
		return err
	}
	for _, gameInfo := range games {
		if gameInfo == nil {
			continue
		}
		g := NewEphemeralGameForEvent(r.ctx, r.workerManager, r.publisher, r.txPool, r.gameResultSettler, gameInfo)
		if err := g.handleGameAbortInternalError(); err != nil {
			log.Errorw("startup abort active game failed", "game id", gameInfo.ID, "err", err)
		}
	}
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
	created := &types.GameCreatedEvent{
		GameID:              gameID,
		Players:             evt.Players,
		IsContinueGame:      true,
		ConfirmationTimeout: r.args.ConfirmationTimeout,
	}
	for _, player := range evt.Players {
		r.notifyPlayerGameCreated(player, created)
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
	created := &types.GameCreatedEvent{
		GameID:              gameID,
		Players:             evt.Players,
		ConfirmationTimeout: r.args.ConfirmationTimeout,
	}
	for _, player := range evt.Players {
		r.notifyPlayerGameCreated(player, created)
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
	if currentRound == nil || currentRound.game == nil {
		return nil, errors.New("no round data for game")
	}

	roundView := conversion.RoundByNumber(gameInfo, currentRound.roundNumber)
	if roundView == nil {
		return nil, errors.New("no round data for game")
	}

	// Determine turnStartAt like in game_handler.go
	turnNumber := currentRound.getCurrentTurnNumber()
	turnStartAt := int64(0)
	currentTurn := currentRound.getCurrentTurn()
	if currentTurn != nil && currentTurn.TurnStartAt > 0 {
		turnStartAt = currentTurn.TurnStartAt
	} else {
		turnStartAt = gameInfo.CreatedAt.Unix()
	}

	gamePhase := conversion.DbGameToProtoGamePhase(gameInfo, roundView, turnNumber, turnStartAt)
	return gamePhase, nil
}

// SyncGamePhase publishes the current game phase to the player's PubSub topic.
func (r *GameManager) SyncGamePhase(address types.PlayerAddress) error {
	gamePhase, err := r.GetGamePhase(address)
	if err != nil {
		// Player not in game: send empty phase
		gamePhase = &proto.GamePhase{
			GameType: proto.GameType_PVP,
		}
	}
	return r.syncGamePhasePublish(address, gamePhase)
}

func (r *GameManager) SubmitTransactions(txs *proto.TransactionBatch) error {
	if txs == nil {
		return nil
	}
	log.Info("receive tx batch, block number: ", txs.BlockNumber)
	defer r.chainSvc.NotifyTxsCompleted(txs)
	blockTime := int64(txs.Timestamp)
	for _, protoTx := range txs.Transactions {
		gameID := uint(protoTx.GameId)
		switch tx := protoTx.Tx.(type) {
		case *proto.Transaction_GameCreated:
			if err := r.executeOnGame(gameID, func(g *Game) error { return g.handleRoomCreated(gameID, blockTime) }); err != nil {
				return err
			}
		case *proto.Transaction_GameTurnSetupReady:
			if tx == nil || tx.GameTurnSetupReady == nil {
				continue
			}
			if err := r.executeOnGame(gameID, func(g *Game) error {
				return g.handleNewTurnSetupOnChain(gameID, blockTime, tx.GameTurnSetupReady)
			}); err != nil {
				return err
			}
		case *proto.Transaction_CommitmentOnChain:
			if tx == nil || tx.CommitmentOnChain == nil {
				continue
			}
			if err := r.executeOnGame(gameID, func(g *Game) error {
				return g.handleGameStateWaittingCommitments(gameID, blockTime, tx.CommitmentOnChain)
			}); err != nil {
				return err
			}
		case *proto.Transaction_CardOnChain:
			if tx == nil || tx.CardOnChain == nil {
				continue
			}
			if err := r.executeOnGame(gameID, func(g *Game) error {
				return g.handleGameStateCardSubmitted(gameID, blockTime, tx.CardOnChain)
			}); err != nil {
				return err
			}
		}
	}

	log.Info("SubmitTransactions done")
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
	game := NewGame(r.ctx, players, r.workerManager, r.publisher, r.txPool, r.gameResultSettler, &r.args)
	if err := game.pushStateToContractCreating(); err != nil {
		return 0, err
	}
	if err := game.saveGame(); err != nil {
		return 0, err
	}
	return game.gameInfo.ID, nil
}

func (r *GameManager) createGame(players []types.PlayerAddress) (uint, error) {
	game := NewGame(r.ctx, players, r.workerManager, r.publisher, r.txPool, r.gameResultSettler, &r.args)
	if err := game.saveGame(); err != nil {
		return 0, err
	}
	return game.gameInfo.ID, nil
}
