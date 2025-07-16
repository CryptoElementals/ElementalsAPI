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

type playerStatusInGame uint32

const (
	player_init = iota
	player_ready
	player_commitment_on_chain
	player_cards_on_chain
)

type gamePlayer struct {
	player      *dao.GamePlayer
	roundPlayer *dao.PlayerRoundInfo
	status      playerStatusInGame
}

func (p *gamePlayer) PlayerAddress() types.PlayerAddress {
	addr := types.PlayerAddress{}
	addr.FromDao(*p.player)
	return addr
}

func (p *gamePlayer) String() string {
	return fmt.Sprintf("%s_%s", p.player.WalletAddress, p.player.TemporaryAddress)
}

type Game struct {
	ctx                 context.Context
	id                  uint
	gameInfo            *dao.GameInfo
	gamePlayers         map[types.PlayerAddress]*gamePlayer
	currentRound        *dao.Round
	workerMangerService *worker.WorkerManager
}

func NewGame(ctx context.Context, players []types.PlayerAddress, workerMangerService *worker.WorkerManager) *Game {
	daoPlayers := make([]*dao.GamePlayer, 0, len(players))
	gamePlayers := make(map[types.PlayerAddress]*gamePlayer)
	for _, player := range players {
		daoPlayer := player.ToDao()
		daoPlayers = append(daoPlayers, &daoPlayer)
		gamePlayers[player] = &gamePlayer{
			player: &daoPlayer,
			status: player_init,
		}
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
	for i := range g.gameInfo.Players {
		addrKey := types.PlayerAddress{
			WalletAddress:    g.gameInfo.Players[i].WalletAddress,
			TemporaryAddress: g.gameInfo.Players[i].TemporaryAddress,
		}
		g.gamePlayers[addrKey] = &gamePlayer{
			player: g.gameInfo.Players[i],
		}
	}
	if len(g.gameInfo.Rounds) != 0 {
		g.currentRound = g.gameInfo.Rounds[len(g.gameInfo.Rounds)-1]
	}
	// recover player status, too

	return nil
}

func (g *Game) Handle(ctx context.Context, event *types.Event) error {
	switch g.gameInfo.Status {
	case proto.GameStatus_GAME_UNKNOWN:
		return g.handleGameStateMatched(event)
	case proto.GameStatus_GAME_WAITTING_CONTRACT:
		return g.handleGameStatusContractReady(event)
	case proto.GameStatus_GAME_RUNNING:
		return g.handleRound(event)
	}
	return fmt.Errorf("invalid game status: %d", g.gameInfo.Status)
}

func (g *Game) handleGameStateMatched(event *types.Event) error {
	if event.EventType != types.EVENT_TYPE_GAME_CREATED {
		return fmt.Errorf("invalid event type: %d", event.EventType)
	}
	evt := event.Data.(*types.PlayerReadyEvent)
	player := g.gamePlayers[evt.PlayerAddress]
	player.status = player_ready
	allPlayers := make([]types.PlayerAddress, 0, len(g.gamePlayers))
	for _, player := range g.gamePlayers {
		if player.status != player_ready {
			return nil
		}
		allPlayers = append(allPlayers, player.PlayerAddress())
	}
	g.gameInfo.Status = proto.GameStatus_GAME_WAITTING_CONTRACT
	err := g.saveGame()
	if err != nil {
		return err
	}
	g.workerMangerService.SendEvent(types.CHAIN_MANAGER_ID, types.NewEvent(g.workerID(), types.EVENT_TYPE_REQUIRE_CONTRACT_CREATION, &types.RequireContractCreationEvent{
		Players: allPlayers,
	}))
	return nil
}

func (g *Game) handleGameStatusContractReady(event *types.Event) error {
	if event.EventType != types.EVENT_TYPE_ROOM_CONTRACT_CREATED {
		return fmt.Errorf("invalid event type: %d", event.EventType)
	}
	startRoundNum := 1
	evt := event.Data.(*types.RoomContractCreated)
	g.gameInfo.RoomContract = evt.RoomContractAddress
	g.setupNewRound()
	err := db.SaveGame(g.gameInfo)
	if err != nil {
		return err
	}
	gameReadyEvt := types.NewEvent(g.workerID(), types.EVENT_TYPE_GAME_READY, &types.GameReadyEvent{
		GameID:          g.gameInfo.ID,
		ContractAddress: evt.RoomContractAddress,
	})
	roundReadyEvt := types.NewEvent(g.workerID(), types.EVENT_TYPE_ROUND_READY, &types.RoundReadyEvent{
		GameID:      g.gameInfo.ID,
		RoundNumber: startRoundNum,
	})
	for _, player := range g.gamePlayers {
		g.workerMangerService.SendEvent(player.String(), gameReadyEvt)
		g.workerMangerService.SendEvent(player.String(), roundReadyEvt)
	}
	return nil
}
func (g *Game) handleGameStateWaittingCommitments(event *types.Event) error {
	if event.EventType != types.EVENT_TYPE_PLAYER_COMMITMENT_ON_CHAIN {
		return fmt.Errorf("invalid event type: %d", event.EventType)
	}
	evt := event.Data.(*types.PlayerCommitmentOnChain)
	// stale events
	if evt.RoundNumber != g.currentRound.RoundNumber {
		return nil
	}
	player := g.gamePlayers[evt.Address]
	player.status = player_commitment_on_chain
	player.roundPlayer.SubmittedCommitment = evt.Commitment
	// check if all player commitment on chain
	for _, player := range g.gamePlayers {
		if player.status != player_commitment_on_chain {
			return nil
		}
	}
	// all player commitment on chain, send EVENT_TYPE_COMMITMENTS_ON_CHAIN to players
	commitmentsOnChainEvt := types.NewEvent(g.workerID(), types.EVENT_TYPE_COMMITMENTS_ON_CHAIN, &types.CommitmentsOnChainEvent{
		GameID:      g.gameInfo.ID,
		RoundNumber: evt.RoundNumber,
	})
	for _, player := range g.gamePlayers {
		g.workerMangerService.SendEvent(player.String(), commitmentsOnChainEvt)
	}
	return nil
}

func (g *Game) handleGameStateCardSubmitted(event *types.Event) error {
	if event.EventType != types.EVENT_TYPE_PLAYER_CARDS_ON_CHAIN {
		return fmt.Errorf("invalid event type: %d", event.EventType)
	}
	// set player cards and player status
	evt := event.Data.(*types.PlayerCardsOnChain)
	// stale events
	if evt.RoundNumber != g.currentRound.RoundNumber {
		return nil
	}
	player := g.gamePlayers[evt.Address]
	player.status = player_cards_on_chain
	for _, card := range evt.Cards {
		player.roundPlayer.RoundSubmittedCards = append(player.roundPlayer.RoundSubmittedCards, &dao.RoundSubmittedCard{
			RoundID: player.roundPlayer.RoundID,
			CardID:  card,
		})
	}
	// check if all player cards on chain
	for _, player := range g.gamePlayers {
		if player.status != player_cards_on_chain {
			return nil
		}
	}
	// send CardsOnChainEvent and RoundCompletedEvent to all players
	cardsOnChainEvt := types.NewEvent(g.workerID(), types.EVENT_TYPE_CARDS_ON_CHAIN, &types.CardsOnChainEvent{
		GameID:      g.gameInfo.ID,
		RoundNumber: evt.RoundNumber,
	})
	for _, player := range g.gamePlayers {
		g.workerMangerService.SendEvent(player.String(), cardsOnChainEvt)
	}
	// TODO: calculate round info
	// TODO: check and calculate game info
	// send GameCompletedEvent to all players
	if false {
		g.handleGameEnd()
	} else {
		roundCompletedEvt := types.NewEvent(g.workerID(), types.EVENT_TYPE_ROUND_COMPLETED, &types.RoundCompletedEvent{
			GameID:    g.gameInfo.ID,
			RoundInfo: g.currentRound,
		})
		g.workerMangerService.SendEvent(player.String(), roundCompletedEvt)
		g.gameInfo.Status = proto.GameStatus_GAME_RUNNING
		g.setupNewRound()
	}
	err := g.saveGame()
	if err != nil {
		return err
	}
	return nil
}

func (g *Game) handleGameEnd() error {
	if g.gameInfo.Status != proto.GameStatus_GAME_RUNNING {
		return fmt.Errorf("invalid game status: %d", g.gameInfo.Status)
	}
	gameCompletedEvt := types.NewEvent(g.workerID(), types.EVENT_TYPE_GAME_COMPLETED, &types.GameCompletedEvent{
		GameID:   g.gameInfo.ID,
		GameInfo: g.gameInfo,
	})
	g.gameInfo.Status = proto.GameStatus_GAME_END
	for _, player := range g.gamePlayers {
		g.workerMangerService.SendEvent(player.String(), gameCompletedEvt)
	}
	return nil
}

func (g *Game) handleRound(event *types.Event) error {
	currentRound := g.currentRound
	switch currentRound.Status {
	case proto.RoundStatus_ROUND_WAITTING_COMMITMENTS:
		return g.handleGameStateWaittingCommitments(event)
	case proto.RoundStatus_ROUND_WAITTING_CARDS:
		return g.handleGameStateCardSubmitted(event)
	}
	return nil
}

func (g *Game) createSelf() {
	g.workerMangerService.SpwanWorker(g.workerID(), types.WORKER_TYPE_GAME, g)
}

func (g *Game) workerID() string {
	return fmt.Sprint(g.id)
}

func (g *Game) setupNewRound() {
	roundNum := 1
	if g.currentRound != nil {
		roundNum = g.currentRound.RoundNumber + 1
	}
	newRound := &dao.Round{
		MatchID:     g.gameInfo.ID,
		RoundNumber: roundNum,
		Status:      proto.RoundStatus_ROUND_WAITTING_COMMITMENTS,
	}
	for _, player := range g.gamePlayers {
		playerRoundInfo := &dao.PlayerRoundInfo{
			GamePlayerID: player.player.ID,
			GamePlayer:   *player.player,
		}
		newRound.PlayerRoundInfos = append(newRound.PlayerRoundInfos, playerRoundInfo)
		player.roundPlayer = playerRoundInfo
	}
	g.currentRound = newRound
	g.gameInfo.Rounds = append(g.gameInfo.Rounds, newRound)
}
