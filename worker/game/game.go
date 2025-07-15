package game

import (
	"context"
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/worker"
	"github.com/CryptoElementals/common/worker/types"
)

type gamePlayer struct {
	types.PlayerAddress
	status proto.PlayerStatus
}

type Game struct {
	ctx                 context.Context
	id                  uint
	gameInfo            *dao.GameInfo
	gamePlayers         []gamePlayer
	workerMangerService *worker.WorkerManager
}

func NewGame(ctx context.Context, players []types.PlayerAddress, workerMangerService *worker.WorkerManager) *Game {
	daoPlayers := make([]dao.GamePlayer, 0)
	gamePlayers := make([]gamePlayer, 0)
	for _, player := range players {
		gamePlayers = append(gamePlayers, gamePlayer{
			PlayerAddress: player,
			status:        proto.PlayerStatus_PLAYER_MATCHED,
		})
		daoPlayers = append(daoPlayers, player.ToDao())
	}
	game := &Game{
		ctx: ctx,
		gameInfo: &dao.GameInfo{
			Players: daoPlayers,
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
		gp := gamePlayer{}
		gp.PlayerAddress.FromDao(player)
		g.gamePlayers = append(g.gamePlayers, gp)
	}
	return nil
}

func (g *Game) Handle(ctx context.Context, event types.Event) error {
	switch g.gameInfo.Status {
	case proto.GameStatus_GAME_UNKNOWN:
		return g.handleGameStateMatched(event)
	case proto.GameStatus_GAME_WAITTING_CONTRACT:
		return g.handleGameStatusContractReady(event)
	case proto.GameStatus_GAME_RUNNING:
		return g.handleRound(event)
	case proto.GameStatus_GAME_END:
		return g.handleGameEnd(event)
	}
	return fmt.Errorf("invalid game status: %d", g.gameInfo.Status)
}

func (g *Game) handleGameStateMatched(event types.Event) error {
	return nil
}

func (g *Game) handleGameStatusContractReady(event types.Event) error {
	if event.EventType != types.EVENT_TYPE_ROOM_CONTRACT_CREATED {
		return fmt.Errorf("invalid event type: %d", event.EventType)
	}
	evt := event.Data.(*types.RoomContractCreated)
	g.gameInfo.RoomContract = evt.RoomContractAddress
	g.gameInfo.Rounds = append(g.gameInfo.Rounds, dao.Round{
		MatchID:     g.gameInfo.ID,
		RoundNumber: 1,
		Status:      proto.RoundStatus_ROUND_WAITTING_COMMITMENTS,
		PlayerRoundInfos: []dao.PlayerRoundInfo{
			dao.PlayerRoundInfo{
				GamePlayerID: g.gameInfo.Players[0].ID,
			},
			dao.PlayerRoundInfo{
				GamePlayerID: g.gameInfo.Players[1].ID,
			},
		},
	})
	return nil
}

func (g *Game) handleGameStateCommitmentsSubmitted(event types.Event) error {
	if event.EventType != types.EVENT_TYPE_PLAYER_COMMITMENT_ON_CHAIN {
		return fmt.Errorf("invalid event type: %d", event.EventType)
	}
	//evt := event.Data.(*types.PlayerCommitmentOnChain)

	return nil
}

func (g *Game) handleGameStateCardSubmitted(event types.Event) error {
	return nil
}

func (g *Game) handleGameEnd(event types.Event) error {
	return nil
}

func (g *Game) handleRound(event types.Event) error {
	currentRound := g.gameInfo.Rounds[len(g.gameInfo.Rounds)-1]
	switch currentRound.Status {
	case proto.RoundStatus_ROUND_WAITTING_COMMITMENTS:
		return g.handleGameStateCommitmentsSubmitted(event)
	case proto.RoundStatus_ROUND_WAITTING_CARDS:
		return g.handleGameStateCardSubmitted(event)
	}
	return nil
}
