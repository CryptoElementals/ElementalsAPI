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
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/snowflake"
	"github.com/CryptoElementals/common/timer"
	"gorm.io/gorm"
)

type GameManager struct {
	ctx               context.Context
	publisher         Publisher
	roomChain         RoomChain
	gameResultSettler GameResultSettler
	// argsTemplateID is the default game_args id for new matches.
	argsTemplateID uint
	gameArgsMu     sync.RWMutex
	gameArgsByID   map[uint]*dao.GameArgs
}

// withGameMutation runs handler inside a DB transaction with a row lock on games.id, then runs
// queued after-commit hooks (settlement, etc.).
func (r *GameManager) withGameMutation(gameID int64, handler func(g *Game) error) error {
	var afterCommit []func() error
	err := db.WithGameMutationTx(gameID, func(tx *gorm.DB, gameInfo *dao.Game) error {
		gameArgs, err := r.getGameArgsForID(gameInfo.GameArgsID)
		if err != nil {
			return err
		}
		gameInfo.GameArgs = gameArgs
		g := NewEphemeralGameForEvent(r.ctx, r.publisher, r.roomChain, r.gameResultSettler, gameInfo)
		g.mutateTx = tx
		g.queueAfterTxCommit = func(fn func() error) {
			afterCommit = append(afterCommit, fn)
		}
		defer func() {
			g.mutateTx = nil
			g.queueAfterTxCommit = nil
		}()
		return handler(g)
	})
	if err != nil {
		return err
	}
	var firstErr error
	for _, fn := range afterCommit {
		if fnErr := fn(); fnErr != nil && firstErr == nil {
			firstErr = fnErr
		}
	}
	return firstErr
}

func (r *GameManager) executeOnGame(gameID int64, handler func(g *Game) error) error {
	if gameID == 0 {
		return fmt.Errorf("executeOnGame: missing game id")
	}
	return r.withGameMutation(gameID, handler)
}
func (r *GameManager) HandleConfirmBattle(req *proto.ConfirmBattleRequest) error {
	return r.executeOnGame(req.GameID, func(g *Game) error { return g.handleConfirmBattle(req) })
}

func (r *GameManager) HandleSurrender(req *proto.SurrenderRequest) error {
	return r.executeOnGame(req.GameID, func(g *Game) error { return g.handleSurrender(req) })
}

func (r *GameManager) HandleTimerEvent(ctx context.Context, evt *timerEvent) error {
	return r.executeOnGame(evt.GameID, func(g *Game) error { g.handleTimerEvent(evt); return nil })
}

func (r *GameManager) HandleSubmitPlayerCommitment(req *proto.SubmitPlayerCommitmentRequest) error {
	if err := validateSubmitPlayerCommitmentRequest(req); err != nil {
		return err
	}
	return r.executeOnGame(req.GameID, func(g *Game) error { return g.handleSubmitPlayerCommitment(req) })
}

func (r *GameManager) HandleSubmitPlayerCard(req *proto.SubmitPlayerCardRequest) error {
	if err := validateSubmitPlayerCardRequest(req); err != nil {
		return err
	}
	return r.executeOnGame(req.GameID, func(g *Game) error { return g.handleSubmitPlayerCard(req) })
}

func NewGameManager(ctx context.Context,
	pub Publisher,
	argsTemplateID uint,
	roomChain RoomChain,
) *GameManager {
	return &GameManager{
		ctx:            ctx,
		publisher:      pub,
		roomChain:      roomChain,
		argsTemplateID: argsTemplateID,
		gameArgsByID:   make(map[uint]*dao.GameArgs),
	}
}

func (r *GameManager) cloneGameArgs() (*dao.GameArgs, error) {
	return r.getGameArgsForID(r.argsTemplateID)
}

func cloneDaoGameArgs(src *dao.GameArgs) *dao.GameArgs {
	ga := *src
	return &ga
}

func (r *GameManager) preloadGameArgsCache() error {
	all, err := db.LoadAllGameArgs()
	if err != nil {
		return err
	}
	r.gameArgsMu.Lock()
	r.gameArgsByID = all
	r.gameArgsMu.Unlock()
	return nil
}

func (r *GameManager) getGameArgsForID(gameArgsID uint) (*dao.GameArgs, error) {
	if gameArgsID == 0 {
		return nil, fmt.Errorf("game args id is required")
	}
	r.gameArgsMu.RLock()
	cached, ok := r.gameArgsByID[gameArgsID]
	r.gameArgsMu.RUnlock()
	if ok && cached != nil {
		return cloneDaoGameArgs(cached), nil
	}
	loaded, err := db.LoadRoomServerGameArgs(gameArgsID)
	if err != nil {
		return nil, err
	}
	r.gameArgsMu.Lock()
	r.gameArgsByID[gameArgsID] = loaded
	r.gameArgsMu.Unlock()
	return cloneDaoGameArgs(loaded), nil
}

func (r *GameManager) Start() error {
	if r.argsTemplateID == 0 {
		return fmt.Errorf("game args template id is required")
	}
	r.registerTimerFunction()

	if err := r.preloadGameArgsCache(); err != nil {
		return err
	}

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
		if err := r.withGameMutation(gameInfo.ID, func(g *Game) error {
			return g.handleGameAbortInternalError()
		}); err != nil {
			log.Errorw("startup abort active game failed", "game id", gameInfo.ID, "err", err)
		}
	}
	return nil
}

func (r *GameManager) Stop() {
	log.Info("closing game manager")
	timer.StopTimer(timer.ScopeRoom)
	log.Info("game manager closed")
}

// createGameAndNotify persists the new game graph, bootstraps first turn / chain flow (same for queue PVP, continue, tournament),
// then returns the game id. completedMatchID is queue PVP only; when non-zero we also use it as games.id.
func (r *GameManager) createGameAndNotify(players []types.PlayerAddress, gameType uint, completedMatchID int64) (int64, error) {
	if err := r.validatePlayersNotInGame(players); err != nil {
		return 0, err
	}
	if gameType == 0 {
		gameType = types.GameTypePVP
	}
	gameArgs, err := r.cloneGameArgs()
	if err != nil {
		return 0, err
	}
	game := NewGame(r.ctx, players, r.publisher, r.roomChain, r.gameResultSettler, gameType, gameArgs)
	game.gameInfo.ID = completedMatchID
	game.gameInfo.QueueMatchID = completedMatchID
	if gameType == types.GameTypeTournament {
		rr := uint32(game.gameInfo.GameArgs.MaxNormalRounds)
		if rr == 0 {
			rr = 3
		}
		game.gameInfo.RegulationRounds = rr
		game.gameInfo.OvertimeRoundsCap = dao.TournamentMaxOvertimeRounds
	}
	if err := game.persistInsertNewGameGraph(); err != nil {
		return 0, err
	}
	gameID := game.gameInfo.ID
	if err := r.executeOnGame(gameID, func(g *Game) error {
		return g.bootstrapFirstTurnAfterQueueConfirmations()
	}); err != nil {
		return 0, err
	}
	return gameID, nil
}

// CreateGameAndRun persists a new game (queue PVP, continue, or tournament), bootstraps chain for turn 1, then notifies.
func (r *GameManager) CreateGameAndRun(players []types.PlayerAddress, gameType uint, completedMatchID int64) (int64, error) {
	matchID := completedMatchID
	if completedMatchID == 0 {
		matchID = snowflake.GenerateID()
	}
	gameID, err := r.createGameAndNotify(players, gameType, matchID)
	if err != nil {
		return 0, err
	}
	log.Infow("queueMatchGameCreated", "game id", gameID, "match id", completedMatchID, "players", types.ToJsonLoggable(players), "game type", gameType)
	return gameID, nil
}

func (r *GameManager) IsPlayerInGame(player types.PlayerAddress) bool {
	// DB is the source of truth; treat any active game as "in game".
	_, err := db.GetActiveGameByPlayer(player.Id, player.TemporaryAddress)
	return err == nil
}

func (r *GameManager) GetGamePhase(address types.PlayerAddress) (*proto.GamePhase, error) {
	gameInfo, err := db.GetActiveGameByPlayer(address.Id, address.TemporaryAddress)
	if err != nil {
		return nil, err
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

	gamePhase := conversion.DbGameToProtoGamePhase(gameInfo, roundView, turnNumber, turnStartAt, address)
	return gamePhase, nil
}

// SyncGamePhase publishes the current game phase on the shared room_events PubSub stream for this player (Event.Receivers).
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
	blockTime := int64(txs.Timestamp)
	for _, protoTx := range txs.Transactions {
		gameID := protoTx.GameId
		switch tx := protoTx.Tx.(type) {
		case *proto.Transaction_GameCreated:
			if err := r.executeOnGame(gameID, func(g *Game) error { return g.handleRoomCreated(blockTime) }); err != nil {
				log.Errorw("SubmitTransactions: handler failed", "error", err, "gameID", gameID, "txKind", "game_created")
			}
		case *proto.Transaction_GameTurnSetupReady:
			if tx == nil || tx.GameTurnSetupReady == nil {
				continue
			}
			if err := r.executeOnGame(gameID, func(g *Game) error {
				return g.handleNewTurnSetupOnChain(gameID, blockTime, tx.GameTurnSetupReady)
			}); err != nil {
				log.Errorw("SubmitTransactions: handler failed", "error", err, "gameID", gameID, "txKind", "game_turn_setup_ready")
			}
		case *proto.Transaction_CommitmentOnChain:
			if tx == nil || tx.CommitmentOnChain == nil {
				continue
			}
			if err := r.executeOnGame(gameID, func(g *Game) error {
				return g.handleGameStateWaittingCommitments(tx.CommitmentOnChain)
			}); err != nil {
				log.Errorw("SubmitTransactions: handler failed", "error", err, "gameID", gameID, "txKind", "commitment_on_chain")
			}
		case *proto.Transaction_CardOnChain:
			if tx == nil || tx.CardOnChain == nil {
				continue
			}
			if err := r.executeOnGame(gameID, func(g *Game) error {
				return g.handleGameStateCardSubmitted(tx.CardOnChain)
			}); err != nil {
				log.Errorw("SubmitTransactions: handler failed", "error", err, "gameID", gameID, "txKind", "card_on_chain")
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
