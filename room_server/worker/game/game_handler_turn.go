package game

import (
	"fmt"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

// ---- Player-driven turn flow (surrender, battle confirm, turn completion) ----

func (g *Game) handleSurrender(req *proto.SurrenderRequest) error {
	if req.Address == nil {
		return fmt.Errorf("missing player address")
	}
	p, err := g.getGamePlayer(req.Address.TemporaryAddress)
	if err != nil {
		return err
	}
	p.status = playerStatusSurrendered
	return g.handleTurnEnd()
}

func (g *Game) handleConfirmBattle(req *proto.ConfirmBattleRequest) error {
	if req.PlayerAddress == nil {
		return fmt.Errorf("missing player address")
	}
	// stale events - check both round and turn number
	if req.RoundNumber != g.currentRound.roundNumber {
		return nil
	}
	currentTurnNumber := g.currentRound.getCurrentTurnNumber()
	if req.TurnNumber != currentTurnNumber {
		return nil
	}
	player, err := g.getGamePlayer(req.PlayerAddress.TemporaryAddress)
	if err != nil {
		return err
	}
	// Update PlayerTurnInfo for current turn
	playerTurnInfo := player.getCurrentPlayerTurnInfo()
	playerTurnInfo.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_READY

	g.publishPartialReadyToOtherPlayers(player.PlayerAddress())
	if !g.areAllPlayersReady() {
		err = g.saveGame()
		if err != nil {
			return err
		}
		return nil
	}

	// All players confirmed battle for this turn
	// The first round, first turn needs to create contract first
	if g.currentRound.roundNumber == 1 && g.currentRound.getCurrentTurnNumber() == 1 {
		g.gameInfo.Status = proto.GameStatus_GAME_RUNNING
		allPlayers := make([]types.PlayerAddress, 0, len(g.currentRound.gamePlayers))
		for _, player := range g.currentRound.gamePlayers {
			allPlayers = append(allPlayers, player.PlayerAddress())
		}
		err := g.sendContractCreation(allPlayers)
		if err != nil {
			g.handleGameAbortInternalError()
			return err
		}
		g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN)
	} else {
		// For all other turns, setup new turn on chain
		err := g.setupNewTurn()
		if err != nil {
			g.handleGameAbortInternalError()
			return err
		}
	}

	err = g.saveGame()
	if err != nil {
		return err
	}
	g.sendTimerEventByCurrentRound()
	return nil
}

// areAllPlayersReady checks if all players have confirmed battle for the current turn
func (g *Game) areAllPlayersReady() bool {
	for _, p := range g.currentRound.gamePlayers {
		if !p.isPlayerReady() {
			return false
		}
	}
	return true
}

// haveAllPlayersSubmittedCommitment checks if all players have submitted a commitment for the current turn
func (g *Game) haveAllPlayersSubmittedCommitment() bool {
	for _, p := range g.currentRound.gamePlayers {
		playerTurnInfo := p.getCurrentPlayerTurnInfo()
		if playerTurnInfo == nil || playerTurnInfo.TurnSubmittedCard == nil {
			return false
		}
		if len(playerTurnInfo.TurnSubmittedCard.CommitmentHash) == 0 {
			return false
		}
	}
	return true
}

// haveAllPlayersSubmittedCard checks if all players have submitted a card for the current turn
func (g *Game) haveAllPlayersSubmittedCard() bool {
	for _, p := range g.currentRound.gamePlayers {
		playerTurnInfo := p.getCurrentPlayerTurnInfo()
		if playerTurnInfo == nil || playerTurnInfo.TurnSubmittedCard == nil {
			return false
		}
		if playerTurnInfo.TurnSubmittedCard.CardID == 0 {
			return false
		}
	}
	return true
}

// handleTurnEnd handles the end of a turn: executes the card, sends events, and checks if round/game ends
func (g *Game) handleTurnEnd() error {
	turnNumber := g.currentRound.getCurrentTurnNumber()
	cardIdx := int(turnNumber) - 1
	roundNumber := g.currentRound.roundNumber

	isGameOver, gameResult, err := g.currentRound.executeCardIndex()
	if err != nil {
		return fmt.Errorf("failed to execute card index %d: %v", cardIdx, err)
	}
	g.gameInfo.GameResult = gameResult

	isRoundComplete := turnNumber >= uint32(g.gameInfo.GameArgs.MaxTurnsPerRound) || isGameOver
	isGameComplete := isGameOver

	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_COMPLETED)

	// Build proto TurnCompleted directly
	turnCompletedEvt := g.buildTurnCompletedEvent(roundNumber, turnNumber, isRoundComplete, isGameComplete, gameResult)
	confirmationTimeout := g.gameInfo.GameArgs.ConfirmationTimeout
	gameContinueTimeout := g.gameInfo.GameArgs.GameContinueTimeout
	turnCompletedEvt.GetTurnCompleted().ConfirmationTimeout = &confirmationTimeout
	if isGameComplete {
		turnCompletedEvt.GetTurnCompleted().GameContinueTimeout = &gameContinueTimeout
	}

	if isRoundComplete {
		g.currentRound.completeReason = proto.RoundCompleteReason_ROUND_COMPLETE_NORMAL
		g.currentRound.setTurnStatus(proto.TurnStatus_TURN_ROUND_COMPLETED)

		if isGameComplete {
			g.currentRound.isLastRound = true
			g.gameInfo.Status = proto.GameStatus_GAME_END
			if err := g.saveGame(); err != nil {
				return err
			}
			g.publishProtoToAllPlayers(turnCompletedEvt)
			completeEvt := &types.GameCompletedEvent{
				GameID:   g.gameInfo.ID,
				GameInfo: g.gameInfo,
			}
			if err := g.completeGameAndNotify(completeEvt); err != nil {
				log.Errorw("handle game complete event failed", "err", err, "game id", g.gameInfo.ID)
			}
			g.stopGame()
			return nil
		}

		g.setupNewRound()
		if err := g.saveGame(); err != nil {
			return err
		}
		g.publishProtoToAllPlayers(turnCompletedEvt)
		return nil
	}

	g.incrementTurnNumber()
	g.currentRound.createNewTurn()
	if err = g.saveRound(); err != nil {
		return err
	}
	g.publishProtoToAllPlayers(turnCompletedEvt)
	g.sendTimerEventByCurrentRound()
	return nil
}

// buildTurnCompletedEvent constructs a proto.Event for TYPE_TURN_COMPLETE from current game state.
func (g *Game) buildTurnCompletedEvent(roundNumber uint32, turnNumber uint32, isRoundComplete, isGameComplete bool, gameResult interface{}) *proto.Event {
	playerTurnInfos := make([]*proto.PlayerTurnInfo, 0, len(g.currentRound.gamePlayers))
	for _, p := range g.currentRound.gamePlayers {
		playerTurnInfo := p.getCurrentPlayerTurnInfo()
		if playerTurnInfo != nil && playerTurnInfo.TurnSubmittedCard != nil {
			addr := p.PlayerAddress()
			playerTurnInfos = append(playerTurnInfos, &proto.PlayerTurnInfo{
				PlayerAddress: addr.ToProto(),
				SubmittedCard: conversion.TurnSubmittedCardToProtoRoundSubmittedCard(playerTurnInfo.TurnSubmittedCard, turnNumber),
			})
		}
	}

	turnCompleted := &proto.TurnCompleted{
		GameId:          uint32(g.gameInfo.ID),
		RoundNum:        roundNumber,
		TurnNum:         turnNumber,
		IsRoundComplete: isRoundComplete,
		IsGameComplete:  isGameComplete,
		PlayerTurnInfos: playerTurnInfos,
	}

	if isGameComplete && g.gameInfo.GameResult != nil {
		turnCompleted.GameResult = conversion.DbGameResultToProtoGameResult(g.gameInfo.GameResult)
	}

	return &proto.Event{
		Type: proto.EventType_TYPE_TURN_COMPLETE,
		Event: &proto.Event_TurnCompleted{
			TurnCompleted: turnCompleted,
		},
	}
}
