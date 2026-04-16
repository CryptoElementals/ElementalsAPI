package game

import (
	"fmt"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
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
	if g.currentRound.getTurnStatus() != proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION {
		return nil
	}
	player, err := g.getGamePlayer(req.PlayerAddress.TemporaryAddress)
	if err != nil {
		return err
	}
	// Update PlayerTurnInfo for current turn
	playerTurnInfo := player.getCurrentPlayerTurnInfo()
	playerTurnInfo.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_READY

	if !g.areAllPlayersReady() {
		if err := g.persistPlayerTurnInfo(playerTurnInfo); err != nil {
			return err
		}
		readyAddr := player.PlayerAddress()
		return g.afterTx(func() error {
			g.publishPartialReadyToOtherPlayers(readyAddr)
			return nil
		})
	}

	// All players confirmed battle for this turn — persist before enqueuing chain work.
	if g.currentRound.roundNumber == 1 && g.currentRound.getCurrentTurnNumber() == 1 {
		g.gameInfo.Status = proto.GameStatus_GAME_RUNNING
		g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN)
	} else {
		g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN)
	}

	if err := g.persistConfirmBattleAllReady(); err != nil {
		return err
	}

	return g.afterTx(func() error {
		if err := g.sendTurnReady(); err != nil {
			g.handleGameAbortInternalError()
			return err
		}
		g.sendTimerEventByCurrentRound()
		return nil
	})
}

// bootstrapFirstTurnAfterQueueConfirmations applies in-DB state and chain enqueue for round 1 / turn 1
// after both players already confirmed via queue-side game_match (skips redundant client ConfirmBattle).
func (g *Game) bootstrapFirstTurnAfterQueueConfirmations() error {
	if g.currentRound.roundNumber != 1 || g.currentRound.getCurrentTurnNumber() != 1 {
		return fmt.Errorf("bootstrapFirstTurnAfterQueueConfirmations: expected round 1 turn 1")
	}
	for _, pl := range g.currentRound.gamePlayers {
		pti := pl.getCurrentPlayerTurnInfo()
		if pti == nil {
			continue
		}
		pti.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_READY
		if err := g.persistPlayerTurnInfo(pti); err != nil {
			return err
		}
	}
	g.gameInfo.Status = proto.GameStatus_GAME_RUNNING
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN)
	if err := g.persistConfirmBattleAllReady(); err != nil {
		return err
	}
	return g.afterTx(func() error {
		allPlayers := make([]types.PlayerAddress, 0, len(g.currentRound.gamePlayers))
		for _, pl := range g.currentRound.gamePlayers {
			allPlayers = append(allPlayers, pl.PlayerAddress())
		}
		if err := g.sendContractCreation(allPlayers); err != nil {
			g.handleGameAbortInternalError()
			return err
		}
		g.sendTimerEventByCurrentRound()
		return nil
	})
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

	turnsThisRound := dao.TurnsPerRoundForGame(g.gameInfo, roundNumber)
	isRoundComplete := turnNumber >= turnsThisRound || isGameOver
	isGameComplete := isGameOver

	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_COMPLETED)

	isNextRoundExtra := isRoundComplete && !isGameComplete && g.currentRound.isNextRoundExtra()

	// Build proto TurnCompleted directly
	turnCompletedEvt := g.buildTurnCompletedEvent(roundNumber, turnNumber, isRoundComplete, isGameComplete, isNextRoundExtra, gameResult)
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
			if err := g.persistTurnEndGameOver(); err != nil {
				return err
			}
			evt := turnCompletedEvt
			return g.afterTx(func() error {
				g.publishProtoToAllPlayers(evt)
				completeEvt := &types.GameCompletedEvent{GameID: g.gameInfo.ID, GameType: g.gameInfo.Type}
				if err := g.completeGameAndNotify(completeEvt); err != nil {
					log.Errorw("handle game complete event failed", "err", err, "game id", g.gameInfo.ID)
					return err
				}
				g.stopGame()
				return nil
			})
		}

		completedTurn := g.currentRound.getCurrentTurn()
		g.setupNewRound()
		newTurn := g.currentRound.getCurrentTurn()
		if err := g.persistCompletedTurnAndNewTurn(completedTurn, newTurn); err != nil {
			return err
		}
		evt := turnCompletedEvt
		return g.afterTx(func() error {
			g.publishProtoToAllPlayers(evt)
			g.sendTimerEventByCurrentRound()
			return nil
		})
	}

	completedTurn := g.currentRound.getCurrentTurn()
	g.incrementTurnNumber()
	g.currentRound.createNewTurn()
	newTurn := g.currentRound.getCurrentTurn()
	if err := g.persistCompletedTurnAndNewTurn(completedTurn, newTurn); err != nil {
		return err
	}
	evt := turnCompletedEvt
	return g.afterTx(func() error {
		g.publishProtoToAllPlayers(evt)
		g.sendTimerEventByCurrentRound()
		return nil
	})
}

// buildTurnCompletedEvent constructs a proto.Event for TYPE_TURN_COMPLETE from current game state.
func (g *Game) buildTurnCompletedEvent(roundNumber uint32, turnNumber uint32, isRoundComplete, isGameComplete, isNextRoundExtra bool, gameResult interface{}) *proto.Event {
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
		GameId:             g.gameInfo.ID,
		RoundNum:           roundNumber,
		TurnNum:            turnNumber,
		IsRoundComplete:    isRoundComplete,
		IsGameComplete:     isGameComplete,
		IsNextRoundExtra:   isNextRoundExtra,
		PlayerTurnInfos:    playerTurnInfos,
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
