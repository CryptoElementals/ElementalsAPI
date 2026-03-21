package game

import (
	"errors"
	"fmt"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/utils"
	"github.com/ethereum/go-ethereum/common"
)

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
	if req.RoundNumber != g.currentRound.round.RoundNumber {
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
	if g.currentRound.round.RoundNumber == 1 && g.currentRound.getCurrentTurnNumber() == 1 {
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

func (g *Game) handleRoomCreated(gameID uint, blockTime int64) error {
	defer g.sendTimerEventByCurrentRound()
	currentTurn := g.currentRound.getCurrentTurn()
	currentTurn.TurnStartAt = blockTime
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_COMMITMENTS)
	if err := g.saveRound(g.currentRound.round); err != nil {
		return err
	}

	players := make([]*proto.PlayerAddress, 0, len(g.currentRound.gamePlayers))
	for _, player := range g.currentRound.gamePlayers {
		addr := player.PlayerAddress()
		players = append(players, addr.ToProto())
	}
	g.publishProtoToAllPlayers(&proto.Event{
		Type: proto.EventType_TYPE_GAME_CREATED,
		Event: &proto.Event_GameReady{
			GameReady: &proto.GameReady{
				GameId:            uint32(g.gameInfo.ID),
				MaxRoundNum:       uint32(g.gameInfo.MaxRounds),
				MaxTurnNum:        3,
				InitialHP:         uint32(g.gameInfo.InitialHP),
				InitialMultiplier: uint32(g.gameInfo.InitialMultiplier),
				Players:           players,
			},
		},
	})
	g.publishProtoToAllPlayers(&proto.Event{
		Type: proto.EventType_TYPE_ROUND_READY,
		Event: &proto.Event_RoundReady{
			RoundReady: &proto.RoundReady{
				GameId:   uint32(g.gameInfo.ID),
				RoundNum: uint32(g.currentRound.round.RoundNumber),
			},
		},
	})
	g.publishProtoToAllPlayers(&proto.Event{
		Type: proto.EventType_TYPE_TURN_READY,
		Event: &proto.Event_TurnReady{
			TurnReady: &proto.TurnReady{
				GameId:                      uint32(g.gameInfo.ID),
				RoundNum:                    uint32(g.currentRound.round.RoundNumber),
				TurnNum:                     1,
				CommitmentSubmissionTimeout: g.gameInfo.CommitmentSubmissionTimeout,
			},
		},
	})
	return nil
}

func (g *Game) handleNewTurnSetupOnChain(gameID uint, blockTime int64, tx *proto.TxGameTurnSetupReady) error {
	defer g.sendTimerEventByCurrentRound()
	if gameID != g.gameInfo.ID {
		return errors.New("invalid game id")
	}
	if tx.RoundNumber != uint32(g.currentRound.round.RoundNumber) {
		return nil
	}
	if tx.TurnNumber != g.currentRound.getCurrentTurnNumber() {
		return nil
	}

	currentTurn := g.currentRound.getCurrentTurn()
	currentTurn.TurnStartAt = blockTime

	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_COMMITMENTS)
	if err := g.saveRound(g.currentRound.round); err != nil {
		return err
	}

	if tx.TurnNumber == 1 {
		g.publishProtoToAllPlayers(&proto.Event{
			Type: proto.EventType_TYPE_ROUND_READY,
			Event: &proto.Event_RoundReady{
				RoundReady: &proto.RoundReady{
					GameId:   uint32(g.gameInfo.ID),
					RoundNum: tx.RoundNumber,
				},
			},
		})
	}
	g.publishProtoToAllPlayers(&proto.Event{
		Type: proto.EventType_TYPE_TURN_READY,
		Event: &proto.Event_TurnReady{
			TurnReady: &proto.TurnReady{
				GameId:                      uint32(g.gameInfo.ID),
				RoundNum:                    tx.RoundNumber,
				TurnNum:                     tx.TurnNumber,
				CommitmentSubmissionTimeout: g.gameInfo.CommitmentSubmissionTimeout,
			},
		},
	})
	return nil
}

func (g *Game) handleGameStateWaittingCommitments(gameID uint, blockTime int64, tx *proto.TxCommitmentOnChain) error {
	commitmentIdx, err := g.validateCommitmentSubmission(tx)
	if err != nil {
		return err
	}

	var address types.PlayerAddress
	address.FromProto(tx.Address)
	player, err := g.getGamePlayer(address.TemporaryAddress)
	if err != nil {
		return err
	}

	playerTurnInfo := player.getCurrentPlayerTurnInfo()
	playerTurnInfo.TurnSubmittedCard.CommitmentHash = tx.Commitment
	playerTurnInfo.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_SUBMITTED

	if g.haveAllPlayersSubmittedCommitment() {
		turnNumber := commitmentIdx + 1
		g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_CARDS)
		err = g.saveRound(g.currentRound.round)
		if err != nil {
			return err
		}
		g.publishProtoToAllPlayers(&proto.Event{
			Type: proto.EventType_TYPE_COMMITMENTS_ON_CHAIN,
			Event: &proto.Event_CommitmentsOnChain{
				CommitmentsOnChain: &proto.CommitmentsOnChain{
					GameId:                uint32(g.gameInfo.ID),
					RoundNum:              tx.RoundNumber,
					TurnNum:               turnNumber,
					CardSubmissionTimeout: g.gameInfo.CardSubmissionTimeout,
				},
			},
		})
	} else {
		err = g.saveRound(g.currentRound.round)
		if err != nil {
			return err
		}
	}
	g.sendTimerEventByCurrentRound()
	return nil
}

func (g *Game) handleGameStateCardSubmitted(gameID uint, blockTime int64, tx *proto.TxCardOnChain) error {
	_, cardEntry, cardID, err := g.validateCardSubmission(tx)
	if err != nil {
		return err
	}
	if cardEntry == nil {
		return nil
	}

	var address types.PlayerAddress
	address.FromProto(tx.Address)
	player, err := g.getGamePlayer(address.TemporaryAddress)
	if err != nil {
		return err
	}

	playerTurnInfo := player.getCurrentPlayerTurnInfo()
	playerTurnInfo.TurnSubmittedCard.CardID = uint32(cardID)
	playerTurnInfo.TurnSubmittedCard.Salt = tx.Salt
	playerTurnInfo.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED

	if g.haveAllPlayersSubmittedCard() {
		return g.handleTurnEnd()
	}
	return g.saveRound(g.currentRound.round)
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
	turnNumber := g.currentRound.getCurrentTurnNumber()
	for _, p := range g.currentRound.gamePlayers {
		var _ uint32 = turnNumber
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
	turnNumber := g.currentRound.getCurrentTurnNumber()
	for _, p := range g.currentRound.gamePlayers {
		var _ uint32 = turnNumber
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
	roundNumber := g.currentRound.round.RoundNumber

	isGameOver, gameResult, err := g.currentRound.executeCardIndex()
	if err != nil {
		return fmt.Errorf("failed to execute card index %d: %v", cardIdx, err)
	}
	g.gameInfo.GameResult = gameResult

	currentTurn := g.currentRound.getCurrentTurn()
	currentTurn.RoundID = g.currentRound.round.ID

	isRoundComplete := turnNumber >= 3 || isGameOver
	isGameComplete := isGameOver

	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_COMPLETED)

	// Build proto TurnCompleted directly
	turnCompletedEvt := g.buildTurnCompletedEvent(roundNumber, turnNumber, isRoundComplete, isGameComplete, gameResult)
	confirmationTimeout := g.gameInfo.ConfirmationTimeout
	turnCompletedEvt.GetTurnCompleted().ConfirmationTimeout = &confirmationTimeout
	if isGameComplete {
		gameContinueTimeout := g.gameInfo.GameContinueTimeout
		turnCompletedEvt.GetTurnCompleted().GameContinueTimeout = &gameContinueTimeout
	}

	if isRoundComplete {
		g.currentRound.round.CompleteReason = proto.RoundCompleteReason_ROUND_COMPLETE_NORMAL
		g.currentRound.setTurnStatus(proto.TurnStatus_TURN_ROUND_COMPLETED)

		if isGameComplete {
			g.currentRound.round.IsLastRound = true
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
	if err = g.saveRound(g.currentRound.round); err != nil {
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

// handleSubmitPlayerCommitment handles the SubmitPlayerCommitment event: validate then enqueue to tx pool
func (g *Game) handleSubmitPlayerCommitment(reqEvt *proto.SubmitPlayerCommitmentRequest) error {
	if err := g.validatePlayerCommitment(reqEvt); err != nil {
		return err
	}
	valid, err := utils.Verify(
		[]any{g.gameInfo.ID, reqEvt.RoundNumber, reqEvt.TurnNumber, reqEvt.Commitment},
		reqEvt.Signature,
		common.HexToAddress(reqEvt.Address.TemporaryAddress))
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("invalid signature")
	}
	return g.txPoolEnqueuer.AddCommitment(reqEvt)
}

// handleSubmitPlayerCard handles the SubmitPlayerCard event
func (g *Game) handleSubmitPlayerCard(reqEvt *proto.SubmitPlayerCardRequest) error {
	// Validate the event - return error if validation fails, nil if valid
	// The worker will automatically send the error to AckChan if present
	if err := g.validatePlayerCard(reqEvt); err != nil {
		return err
	}
	// Get commitment for this round and turn and verify the commitment
	player, err := g.getGamePlayer(reqEvt.Address.TemporaryAddress)
	if err != nil {
		return fmt.Errorf("failed to get player: %w", err)
	}

	playerTurnInfo := player.getCurrentPlayerTurnInfo()
	if playerTurnInfo == nil || playerTurnInfo.TurnSubmittedCard == nil {
		return fmt.Errorf("player turn info or submitted card not found")
	}

	storedCommitment := playerTurnInfo.TurnSubmittedCard.CommitmentHash
	if len(storedCommitment) == 0 {
		return fmt.Errorf("commitment not found for this round and turn")
	}

	// Calculate commitment from submitted card and salt
	calculatedCommitment, err := utils.SolidityPackedKeccak256(
		[]any{
			uint(reqEvt.Card),
			reqEvt.Salt,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to calculate commitment: %w", err)
	}

	// Verify that the calculated commitment matches the stored commitment
	storedCommitmentHash := common.BytesToHash(storedCommitment)
	if storedCommitmentHash != calculatedCommitment {
		return fmt.Errorf("commitment verification failed: stored commitment does not match calculated commitment from card and salt")
	}

	// verify signature for: game id, round number, card index, card, salt
	valid, err := utils.Verify(
		[]any{g.gameInfo.ID, reqEvt.RoundNumber, reqEvt.TurnNumber, uint(reqEvt.Card), reqEvt.Salt},
		reqEvt.Signature,
		common.HexToAddress(reqEvt.Address.TemporaryAddress))
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("invalid signature")
	}
	return g.txPoolEnqueuer.AddCard(reqEvt)
}
