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
		g := NewEphemeralGameForEvent(r.ctx, r.workerManager, r.txPool, r.gameResultSettler, gameInfo)
		return handler(g)
	})
}

func (r *GameManager) HandlePlayerReadyEvent(ctx context.Context, evt *types.PlayerReadyEvent) error {
	return r.executeOnGame(evt.GameId, func(g *Game) error { return g.handleWaittingPlayersConfirmed(evt) })
}

func (r *GameManager) HandleSurrenderEvent(ctx context.Context, evt *types.SurrenderEvent) error {
	return r.executeOnGame(evt.GameID, func(g *Game) error { return g.handleSurrenderEvent(evt) })
}

func (r *GameManager) HandleTimerEvent(ctx context.Context, evt *timerEvent) error {
	return r.executeOnGame(evt.GameID, func(g *Game) error { g.handleTimerEvent(evt); return nil })
}

func (r *GameManager) HandleSubmitPlayerCommitment(evt *types.SubmitPlayerCommitment) error {
	return r.executeOnGame(evt.GameID, func(g *Game) error { return g.handleSubmitPlayerCommitment(evt) })
}

func (r *GameManager) HandleSubmitPlayerCard(evt *types.SubmitPlayerCard) error {
	return r.executeOnGame(evt.GameID, func(g *Game) error { return g.handleSubmitPlayerCard(evt) })
}

func NewGameManager(ctx context.Context,
	workerManagerService *worker.WorkerManager,
	gameArgs dao.GameArgs,
	chainSvc ContractClient,
	poolBatchSize int,
) *GameManager {
	m := &GameManager{
		ctx:            ctx,
		gameLocks:      make(map[uint]*sync.Mutex),
		workerManager:  workerManagerService,
		chainSvc:       chainSvc,
		args:           gameArgs,
	}
	// Set default pool processing interval if not set
	if m.args.PoolProcessingInterval <= 0 {
		m.args.PoolProcessingInterval = 5 // Default 5 seconds
	}
	m.txPool = newTxPool(chainSvc, poolBatchSize)
	return m
}

func (r *GameManager) Start() error {
	r.registerTimerFunction()

	// Start background goroutine for pool processing
	go r.txPool.processPools(r.ctx, r.args)

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
		g := NewEphemeralGameForEvent(r.ctx, r.workerManager, r.txPool, r.gameResultSettler, gameInfo)
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
			evt := &types.RoomCreated{GameID: gameID, TimeStamp: blockTime}
			if err := r.executeOnGame(gameID, func(g *Game) error { return g.handleRoomCreated(evt) }); err != nil {
				return err
			}
		case *proto.Transaction_GameTurnSetupReady:
			if tx == nil || tx.GameTurnSetupReady == nil {
				continue
			}
			evt := &types.NewTurnSetupComplete{
				GameID:      gameID,
				RoundNumber: tx.GameTurnSetupReady.RoundNumber,
				TurnNumber:  tx.GameTurnSetupReady.TurnNumber,
				TimeStamp:   blockTime,
			}
			if err := r.executeOnGame(gameID, func(g *Game) error { return g.handleNewTurnSetupOnChain(evt) }); err != nil {
				return err
			}
		case *proto.Transaction_CommitmentOnChain:
			if tx == nil || tx.CommitmentOnChain == nil {
				continue
			}
			player := types.PlayerAddress{}
			player.FromProto(tx.CommitmentOnChain.Address)
			evt := &types.PlayerCommitmentOnChain{
				GameID:          gameID,
				Address:         player,
				RoundNumber:     tx.CommitmentOnChain.RoundNumber,
				Commitment:      tx.CommitmentOnChain.Commitment,
				CommitmentIndex: tx.CommitmentOnChain.TurnNumber, // 1-based (1,2,3)
				TimeStamp:       blockTime,
			}
			if err := r.executeOnGame(gameID, func(g *Game) error { return g.handleGameStateWaittingCommitments(evt) }); err != nil {
				return err
			}
		case *proto.Transaction_CardOnChain:
			if tx == nil || tx.CardOnChain == nil {
				continue
			}
			player := types.PlayerAddress{}
			player.FromProto(tx.CardOnChain.Address)
			evt := &types.PlayerCardOnChain{
				GameID:      gameID,
				Address:     player,
				RoundNumber: tx.CardOnChain.RoundNumber,
				Salt:        tx.CardOnChain.Salt,
				Card:        uint(tx.CardOnChain.CardId),
				CardIndex:   tx.CardOnChain.TurnNumber, // 1-based (1,2,3)
				TimeStamp:   blockTime,
			}
			if err := r.executeOnGame(gameID, func(g *Game) error { return g.handleGameStateCardSubmitted(evt) }); err != nil {
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
