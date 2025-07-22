package game

import (
	"context"
	"errors"
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/room_server/worker"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

type gamePlayer struct {
	player      *dao.GamePlayerInfo
	roundPlayer *dao.PlayerRoundInfo
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
	gameInfo            *dao.Game
	gamePlayers         map[types.PlayerAddress]*gamePlayer
	currentRound        *dao.Round
	workerMangerService *worker.WorkerManager
}

func NewGame(ctx context.Context, players []types.PlayerAddress, workerMangerService *worker.WorkerManager) *Game {
	daoPlayers := make([]*dao.GamePlayerInfo, 0, len(players))
	gamePlayers := make(map[types.PlayerAddress]*gamePlayer)
	for _, player := range players {
		daoPlayer := player.ToDao()
		daoPlayers = append(daoPlayers, daoPlayer)
		gamePlayers[player] = &gamePlayer{
			player: daoPlayer,
		}
	}
	game := &Game{
		ctx: ctx,
		gameInfo: &dao.Game{
			Players: daoPlayers,
			Type:    types.GameTypePVP,
		},
		gamePlayers:         gamePlayers,
		workerMangerService: workerMangerService,
	}
	game.setupNewRound()
	return game
}

func NewGameFromGameInfo(ctx context.Context, workerMangerService *worker.WorkerManager, gameInfo *dao.Game) *Game {
	g := &Game{
		ctx:                 ctx,
		gameInfo:            gameInfo,
		gamePlayers:         make(map[types.PlayerAddress]*gamePlayer),
		workerMangerService: workerMangerService,
	}

	for _, playerInfo := range g.gameInfo.Players {
		addrKey := types.PlayerAddress{
			WalletAddress:    playerInfo.WalletAddress,
			TemporaryAddress: playerInfo.TemporaryAddress,
		}
		g.gamePlayers[addrKey] = &gamePlayer{
			player: playerInfo,
		}
	}
	if len(g.gameInfo.Rounds) != 0 {
		roundNum := uint32(0)
		for _, r := range g.gameInfo.Rounds {
			if r.RoundNumber > roundNum {
				roundNum = r.RoundNumber
				g.currentRound = r
			}
		}
		for _, roundPlayer := range g.currentRound.PlayerRoundInfos {
			addrKey := types.PlayerAddress{
				WalletAddress:    roundPlayer.WalletAddress,
				TemporaryAddress: roundPlayer.TemporaryAddress,
			}
			g.gamePlayers[addrKey].roundPlayer = roundPlayer
		}
	} else {
		g.setupNewRound()
	}

	return g
}

func (g *Game) saveGame() error {
	err := db.SaveGame(g.gameInfo)
	if err != nil {
		log.Errorf("SaveGame failed, err: %v", err)
		return err
	}
	return nil
}

func (g *Game) Handle(ctx context.Context, sender worker.EventSender, event *types.Event) error {
	switch g.gameInfo.Status {
	case proto.GameStatus_GAME_INIT, proto.GameStatus_GAME_RUNNING:
		return g.handleRound(event)
	}
	return fmt.Errorf("invalid game status: %d", g.gameInfo.Status)
}

func (g *Game) handleRound(event *types.Event) error {
	currentRound := g.currentRound
	switch currentRound.Status {
	case proto.RoundStatus_ROUND_WAITTING_BATTLE_CONFIRMATION:
		return g.handleWaittingRoundPlayersConfirmed(event)
	case proto.RoundStatus_ROUND_WAITTING_SETUP_ON_CHAIN:
		return g.handleGameStateWaittingSetupOnChain(event)
	case proto.RoundStatus_ROUND_WAITTING_COMMITMENTS:
		return g.handleGameStateWaittingCommitments(event)
	case proto.RoundStatus_ROUND_WAITTING_CARDS:
		return g.handleGameStateCardSubmitted(event)

	}
	return nil
}

func (g *Game) handleWaittingRoundPlayersConfirmed(event *types.Event) error {
	evt, err := types.AssertInterface[*types.PlayerReadyEvent](event)
	if err != nil {
		return err
	}
	// stale events
	if evt.RoundNumber != g.currentRound.RoundNumber {
		return nil
	}
	player := g.gamePlayers[evt.PlayerAddress]
	player.roundPlayer.PlayerReady = true
	// check if all players ready
	allPlayersReady := true
	for _, player := range g.gamePlayers {
		if !player.roundPlayer.PlayerReady {
			allPlayersReady = false
		}
	}
	if !allPlayersReady {
		return db.SavePlayerRoundInfo(player.roundPlayer)
	}
	allPlayers := make([]types.PlayerAddress, 0, len(g.gamePlayers))
	for _, player := range g.gamePlayers {
		allPlayers = append(allPlayers, player.PlayerAddress())
	}
	var newEvt *types.Event
	// the first round, we need to create contract
	if g.currentRound.RoundNumber == 1 {
		g.gameInfo.Status = proto.GameStatus_GAME_RUNNING
		newEvt = types.NewEvent(g.workerID(), &types.RequireContractCreationEvent{
			GameID:  g.gameInfo.ID,
			Players: allPlayers,
		})
	} else {
		if g.gameInfo.RoomContract == "" {
			return errors.New("room contract empty, need RequireContractCreationEvent but got RequireSetupNewRoundEvent")
		}
		// otherwise we need to setup new round on chain
		newEvt = types.NewEvent(g.workerID(), &types.RequireSetupNewRoundEvent{
			GameID:          g.gameInfo.ID,
			ContractAddress: g.gameInfo.RoomContract,
			RoundNumber:     uint32(g.currentRound.RoundNumber),
		})
	}
	g.currentRound.Status = proto.RoundStatus_ROUND_WAITTING_SETUP_ON_CHAIN
	err = g.saveGame()
	if err != nil {
		return err
	}
	g.workerMangerService.SendEvent(types.CHAIN_MANAGER_ID, newEvt)
	return nil
}

func (g *Game) handleGameStateWaittingSetupOnChain(event *types.Event) error {
	if g.currentRound.RoundNumber == 1 {
		return g.handleRoomContractCreated(event)
	}
	return g.handleNewRoundSetupOnChain(event)
}

func (g *Game) handleRoomContractCreated(event *types.Event) error {
	evt, err := types.AssertInterface[*types.RoomContractCreated](event)
	if err != nil {
		return err
	}
	g.gameInfo.RoomContract = evt.RoomContractAddress
	// just skip setup new round on chain for first round
	g.currentRound.Status = proto.RoundStatus_ROUND_WAITTING_COMMITMENTS
	err = g.saveGame()
	if err != nil {
		return err
	}
	gameReadyEvt := types.NewEvent(g.workerID(), &types.GameReadyEvent{
		GameID:          g.gameInfo.ID,
		ContractAddress: evt.RoomContractAddress,
	})
	roundReadyEvt := types.NewEvent(g.workerID(), &types.RoundReadyEvent{
		GameID:      g.gameInfo.ID,
		RoundNumber: g.currentRound.RoundNumber,
	})
	g.sendEventsToAllPlayers(gameReadyEvt, roundReadyEvt)
	return nil
}

func (g *Game) handleNewRoundSetupOnChain(event *types.Event) error {
	evt, err := types.AssertInterface[*types.NewRoundSetupComplete](event)
	if err != nil {
		return err
	}
	if evt.GameID != g.gameInfo.ID {
		return errors.New("invalid game id")
	}
	// stale event
	if evt.RoundNumber != uint32(g.currentRound.RoundNumber) {
		return nil
	}
	g.currentRound.Status = proto.RoundStatus_ROUND_WAITTING_COMMITMENTS
	err = db.SaveRound(g.currentRound)
	if err != nil {
		return err
	}
	g.sendEventsToAllPlayers(types.NewEvent(g.workerID(), &types.RoundReadyEvent{
		GameID:      g.gameInfo.ID,
		RoundNumber: evt.RoundNumber,
	}))
	return nil
}

func (g *Game) handleGameStateWaittingCommitments(event *types.Event) error {
	evt, err := types.AssertInterface[*types.PlayerCommitmentOnChain](event)
	if err != nil {
		return err
	}
	// stale events
	if evt.RoundNumber != g.currentRound.RoundNumber {
		return nil
	}
	player := g.gamePlayers[evt.Address]
	player.roundPlayer.SubmittedCommitment = evt.Commitment
	// check if all player commitment on chain
	allCommitmentsOnChain := true
	for _, player := range g.gamePlayers {
		if len(player.roundPlayer.SubmittedCommitment) == 0 {
			allCommitmentsOnChain = false
			break
		}
	}
	if !allCommitmentsOnChain {
		return db.SavePlayerRoundInfo(player.roundPlayer)
	}
	g.currentRound.Status = proto.RoundStatus_ROUND_WAITTING_CARDS
	err = db.SaveRound(g.currentRound)
	if err != nil {
		return err
	}
	// all player commitment on chain, send EVENT_TYPE_COMMITMENTS_ON_CHAIN to players
	commitmentsOnChainEvt := types.NewEvent(g.workerID(), &types.CommitmentsOnChainEvent{
		GameID:      g.gameInfo.ID,
		RoundNumber: evt.RoundNumber,
	})
	g.sendEventsToAllPlayers(commitmentsOnChainEvt)
	return nil
}

func (g *Game) handleGameStateCardSubmitted(event *types.Event) error {
	// set player cards and player status
	evt, err := types.AssertInterface[*types.PlayerCardsOnChain](event)
	if err != nil {
		return err
	}
	// stale events
	if evt.RoundNumber != g.currentRound.RoundNumber {
		return nil
	}
	player := g.gamePlayers[evt.Address]
	for _, card := range evt.Cards {
		player.roundPlayer.SubmittedCards = append(player.roundPlayer.SubmittedCards, &dao.RoundSubmittedCard{
			CardID: card,
		})
	}
	// check if all player cards on chain
	allCardsOnChain := true
	for _, player := range g.gamePlayers {
		if len(player.roundPlayer.SubmittedCards) == 0 {
			allCardsOnChain = false
			break
		}
	}
	if !allCardsOnChain {
		return db.SavePlayerRoundInfo(player.roundPlayer)
	}
	// TODO: calculate round info
	// TODO: check and calculate game info
	// send GameCompletedEvent to all players
	if g.currentRound.RoundNumber == 3 {
		return g.handleGameEnd()
	} else {
		g.currentRound.Status = proto.RoundStatus_ROUND_COMPLETED
		roundCompletedEvt := types.NewEvent(g.workerID(), &types.RoundCompletedEvent{
			GameID:    g.gameInfo.ID,
			RoundInfo: g.currentRound,
		})
		g.setupNewRound()
		err := g.saveGame()
		if err != nil {
			return err
		}
		g.sendEventsToAllPlayers(roundCompletedEvt)
		return nil
	}
}

// can go into game end from any other status
func (g *Game) handleGameEnd() error {
	if g.gameInfo.Status == proto.GameStatus_GAME_END {
		return fmt.Errorf("invalid game status: %d", g.gameInfo.Status)
	}
	gameCompletedEvt := types.NewEvent(g.workerID(), &types.GameCompletedEvent{
		GameID:   g.gameInfo.ID,
		GameInfo: g.gameInfo,
	})
	g.currentRound.Status = proto.RoundStatus_ROUND_COMPLETED
	g.gameInfo.Status = proto.GameStatus_GAME_END
	err := g.saveGame()
	if err != nil {
		return err
	}
	g.sendEventsToAllPlayers(gameCompletedEvt)
	g.stopWorker()
	return nil
}

func (g *Game) stopWorker() {
	g.workerMangerService.CloseWorker(g.workerID())
}

func (g *Game) createSelf() {
	g.workerMangerService.SpwanWorker(g.ctx, g.workerID(), types.WORKER_TYPE_GAME, g)
}

func (g *Game) workerID() string {
	return fmt.Sprint(g.gameInfo.ID)
}

func (g *Game) setupNewRound() {
	roundNum := uint32(1)
	if g.currentRound != nil {
		roundNum = g.currentRound.RoundNumber + 1
	}
	newRound := &dao.Round{
		GameID:      g.gameInfo.ID,
		RoundNumber: roundNum,
		Status:      proto.RoundStatus_ROUND_WAITTING_BATTLE_CONFIRMATION,
	}
	for _, player := range g.gamePlayers {
		playerRoundInfo := &dao.PlayerRoundInfo{
			WalletAddress:    player.player.TemporaryAddress,
			TemporaryAddress: player.player.WalletAddress,
			SubmittedCards:   make([]*dao.RoundSubmittedCard, 0),
		}
		newRound.PlayerRoundInfos = append(newRound.PlayerRoundInfos, playerRoundInfo)
		player.roundPlayer = playerRoundInfo
	}
	g.currentRound = newRound
	g.gameInfo.Rounds = append(g.gameInfo.Rounds, newRound)
}

func (g *Game) sendEventsToAllPlayers(events ...*types.Event) {
	for _, player := range g.gamePlayers {
		for _, event := range events {
			g.workerMangerService.SendEvent(player.String(), event)
		}
	}
}
