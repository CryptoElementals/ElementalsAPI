package game

import (
	"context"
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"gorm.io/gorm"
)

// NewEphemeralGameForEvent creates a transient Game instance backed by the latest DB state.
// It is intended for synchronous event handling (e.g. from GameManager) rather than long-lived workers.
func NewEphemeralGameForEvent(
	ctx context.Context,
	pub Publisher,
	txPoolEnqueuer TxPoolEnqueuer,
	gameResultSettler GameResultSettler,
	gameInfo *dao.Game,
) *Game {
	return &Game{
		ctx:               ctx,
		gameInfo:          gameInfo,
		currentRound:      buildRuntimeState(gameInfo),
		publisher:         pub,
		txPoolEnqueuer:    txPoolEnqueuer,
		gameResultSettler: gameResultSettler,
	}
}

type Game struct {
	ctx               context.Context
	gameInfo          *dao.Game
	currentRound      *round
	publisher         Publisher
	txPoolEnqueuer    TxPoolEnqueuer
	gameResultSettler GameResultSettler

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
			player:    daoPlayer,
			currentHP: ga.InitialHP,
		}
	}
	gameInfo := &dao.Game{
		Players:  daoPlayers,
		Type:     gameType,
		GameArgs: &ga,
	}
	game := &Game{
		ctx:               ctx,
		gameInfo:          gameInfo,
		currentRound:      &round{game: gameInfo, gamePlayers: gamePlayers},
		publisher:         pub,
		txPoolEnqueuer:    txPoolEnqueuer,
		gameResultSettler: gameResultSettler,
	}
	game.setupNewRound()
	return game
}

// incrementTurnNumber increments the turn number for the current round
func (g *Game) incrementTurnNumber() {
	g.currentRound.turnNumber++
}

func (g *Game) stopGame() {
	log.Infow("stop game", "game id", g.gameInfo.ID)
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
