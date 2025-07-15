package game

import (
	"context"
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/types"
)

type GameStatus uint16

const (
	GAME_STATE_MATCHED GameStatus = iota
	GAME_STATE_WAITTING_READY
	GAME_STATE_COMMITMENTS_SUBMITTED
	GAME_STATE_CARD_SUBMITTED
	GAME_END
)

type playerStatus uint16

const (
	PLAYER_STATUS_MATCHED = iota
	PLAYER_STATUS_READY
	PLAYER_STATUS_COMMITMENTS_SUBMITTED
	PLAYER_STATUS_CARD_SUBMITTED
)

type gamePlayer struct {
	dao.GamePlayer
	status playerStatus
}

type Game struct {
	ctx                 context.Context
	id                  uint
	status              GameStatus
	contractAddress     string
	roomWorker          *worker.Worker
	gameInfo            *dao.GameInfo
	gamePlayers         []gamePlayer
	workerMangerService *worker.WorkerManager
}

func NewGame(ctx context.Context, players []dao.GamePlayer, workerMangerService *worker.WorkerManager) *Game {
	gamePlayers := make([]gamePlayer, 0)
	for _, player := range players {
		gamePlayers = append(gamePlayers, gamePlayer{
			GamePlayer: player,
			status:     PLAYER_STATUS_MATCHED,
		})
	}
	game := &Game{
		ctx: ctx,
		gameInfo: &dao.GameInfo{
			Players: players,
			Type:    types.GameTypePVP,
		},
		gamePlayers:         gamePlayers,
		workerMangerService: workerMangerService,
	}
	return game
}

func (g *Game) saveGame() error {
	err := db.SaveGame(g.gameInfo)
	if err != nil {
		log.Errorf("SaveGame failed, err: %v", err)
		return err
	}
	return nil
}

func (g *Game) recoverGame(gameInfo *dao.GameInfo) error {
	g.id = gameInfo.ID
	g.gameInfo = gameInfo
	for _, player := range g.gameInfo.Players {
		gp := gamePlayer{
			GamePlayer: player,
		}
		g.gamePlayers = append(g.gamePlayers, gp)
	}
	return nil
}

func (g *Game) Handle(ctx context.Context, event types.Event) error {
	switch g.status {
	case GAME_STATE_MATCHED:
		return g.handleGameStateMatched(event)
	case GAME_STATE_WAITTING_READY:
		return g.handleGameStateWaittingReady(event)
	case GAME_STATE_COMMITMENTS_SUBMITTED:
		return g.handleGameStateCommitmentsSubmitted(event)
	case GAME_STATE_CARD_SUBMITTED:
		return g.handleGameStateCardSubmitted(event)
	case GAME_END:
		return g.handleGameEnd(event)
	}
	return fmt.Errorf("invalid game status: %d", g.status)
}

func (g *Game) handleGameStateMatched(event types.Event) error {
	return nil
}

func (g *Game) handleGameStateWaittingReady(event types.Event) error {
	return nil
}

func (g *Game) handleGameStateCommitmentsSubmitted(event types.Event) error {
	return nil
}

func (g *Game) handleGameStateCardSubmitted(event types.Event) error {
	return nil
}

func (g *Game) handleGameEnd(event types.Event) error {
	return nil
}
