package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

// NewEphemeralGameForEvent creates a transient Game instance backed by the latest DB state.
// It is intended for synchronous event handling (e.g. from GameManager) rather than long-lived workers.
func NewEphemeralGameForEvent(
	ctx context.Context,
	workerManagerService *worker.WorkerManager,
	pub Publisher,
	txPoolEnqueuer TxPoolEnqueuer,
	gameResultSettler GameResultSettler,
	gameInfo *dao.Game,
) *Game {
	return &Game{
		ctx:                  ctx,
		gameInfo:             gameInfo,
		currentRound:         buildRuntimeState(gameInfo),
		workerManagerService: workerManagerService,
		publisher:            pub,
		txPoolEnqueuer:       txPoolEnqueuer,
		gameResultSettler:    gameResultSettler,
	}
}

type Game struct {
	ctx                  context.Context
	gameInfo             *dao.Game
	currentRound         *round
	workerManagerService *worker.WorkerManager
	publisher            Publisher
	txPoolEnqueuer       TxPoolEnqueuer
	gameResultSettler    GameResultSettler

	// mutateTx, when non-nil, is the outer game mutation transaction (pessimistic lock on games row).
	mutateTx *gorm.DB
	// queueAfterTxCommit schedules work to run after the mutation transaction commits (e.g. settlement DB).
	queueAfterTxCommit func(func() error)
}

// afterTx runs fn after the pessimistic game mutation transaction commits when queueAfterTxCommit is set
// (Phase 3: avoid Pub/Sub, timers, and tx-pool enqueue while holding FOR UPDATE). Otherwise runs fn immediately.
func (g *Game) afterTx(fn func() error) error {
	if g.queueAfterTxCommit != nil {
		g.queueAfterTxCommit(fn)
		return nil
	}
	return fn()
}

func NewGame(
	ctx context.Context,
	players []types.PlayerAddress,
	workerManagerService *worker.WorkerManager,
	pub Publisher,
	txPoolEnqueuer TxPoolEnqueuer,
	gameResultSettler GameResultSettler,
	gameType uint,
	gameArgs *dao.GameArgs) *Game {
	if gameType == 0 {
		gameType = types.GameTypePVP
	}
	daoPlayers := make([]*dao.GamePlayerInfo, 0, len(players))
	gamePlayers := make(map[string]*gamePlayer)
	ga := *gameArgs
	for _, player := range players {
		daoPlayer := player.ToDao()
		daoPlayers = append(daoPlayers, daoPlayer)
		gamePlayers[player.TemporaryAddress] = &gamePlayer{
			player:     daoPlayer,
			currentHP:  ga.InitialHP,
			multiplier: uint32(ga.InitialMultiplier),
		}
	}
	gameInfo := &dao.Game{
		Players:  daoPlayers,
		Type:     gameType,
		GameArgs: &ga,
	}
	game := &Game{
		ctx:                  ctx,
		gameInfo:             gameInfo,
		currentRound:         &round{game: gameInfo, gamePlayers: gamePlayers},
		workerManagerService: workerManagerService,
		publisher:            pub,
		txPoolEnqueuer:       txPoolEnqueuer,
		gameResultSettler:    gameResultSettler,
	}
	game.setupNewRound()
	return game
}

// pushStateToContractCreating marks all players ready and enqueues contract creation (no DB persist of PTIs).
// Used when recovering a GAME_INIT row from DB after restart; new games use bootstrapFirstTurnAfterQueueConfirmations after insert.
func (g *Game) pushStateToContractCreating() error {
	g.gameInfo.Status = proto.GameStatus_GAME_RUNNING
	allPlayers := make([]types.PlayerAddress, 0, len(g.currentRound.gamePlayers))
	for _, player := range g.currentRound.gamePlayers {
		allPlayers = append(allPlayers, player.PlayerAddress())
		if player.currentTurnInfo != nil {
			player.currentTurnInfo.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_READY
		}
	}
	if err := g.sendContractCreation(allPlayers); err != nil {
		g.handleGameAbortInternalError()
		return err
	}
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN)
	return nil
}

// setupNewTurn sends event to chain manager to setup a new turn
// Note: For the first turn of the first round, this is not needed as the contract creation handles it
func (g *Game) setupNewTurn() error {
	turnNumber := g.currentRound.getCurrentTurnNumber()
	log.Infow("setup new turn", "game id", g.gameInfo.ID, "round number", g.currentRound.roundNumber, "turn number", turnNumber)
	err := g.sendTurnReady()
	if err != nil {
		return err
	}
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN)
	return nil
}

// incrementTurnNumber increments the turn number for the current round
func (g *Game) incrementTurnNumber() {
	g.currentRound.turnNumber++
}

func (g *Game) stopGame() {
	log.Infow("stop game", "game id", g.gameInfo.ID)
	g.workerManagerService.CloseWorker(g.workerID())
}

func (g *Game) workerID() string {
	return fmt.Sprint(g.gameInfo.ID)
}

// WorkerID returns the worker ID for this game (exported version)
func (g *Game) WorkerID() string {
	return g.workerID()
}

func (g *Game) setupNewRound() {
	maxR := dao.MaxRoundNumberFromTurns(g.gameInfo.Turns)
	roundNum := uint32(1)
	if maxR >= 1 {
		roundNum = maxR + 1
	}
	g.currentRound.roundNumber = roundNum
	g.currentRound.turnNumber = 1
	g.currentRound.createNewTurn()
	g.sendTimerEventByCurrentRound()
}

func (g *Game) getGamePlayer(tempAddr string) (*gamePlayer, error) {
	player, ok := g.currentRound.gamePlayers[strings.ToLower(tempAddr)]
	if !ok {
		return nil, fmt.Errorf("player %s not found", tempAddr)
	}
	return player, nil
}

func (g *Game) sendContractCreation(allPlayers []types.PlayerAddress) error {
	ga := g.gameInfo.GameArgs
	evt := &types.RequireGameCreationEvent{
		GameID:         g.gameInfo.ID,
		Players:        allPlayers,
		InitialHP:      ga.InitialHP,
		RoundTimeout:   ga.CommitmentSubmissionTimeout,
		MaxRoundNumber: int64(dao.EffectiveMaxRounds(g.gameInfo)),
	}
	g.txPoolEnqueuer.AddCreateRoom(evt)
	return nil
}

func (g *Game) sendTurnReady() error {
	evt := &types.RequireSetupNewTurnEvent{
		GameID:      g.gameInfo.ID,
		RoundNumber: g.currentRound.roundNumber,
		TurnNumber:  g.currentRound.getCurrentTurnNumber(),
	}
	g.txPoolEnqueuer.AddSetTurnReady(evt)
	return nil
}

// completeGameAndNotify runs settlement, clears tx pool info for this game, then notifies the manager to remove the game from maps.
func (g *Game) completeGameAndNotify(evt *types.GameCompletedEvent) error {
	if g.gameResultSettler != nil {
		if err := g.gameResultSettler.GameResultSettlement(evt); err != nil {
			return err
		}
	}
	g.txPoolEnqueuer.ClearGameInfo(evt.GameID)
	return nil
}
