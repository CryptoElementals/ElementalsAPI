package game

import (
	"errors"
	"fmt"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/utils"
	"github.com/ethereum/go-ethereum/common"
)

func (g *Game) handleSurrenderEvent(event *types.SurrenderEvent) error {
	p, err := g.getGamePlayer(event.Address.TemporaryAddress)
	if err != nil {
		return err
	}
	p.status = playerStatusSurrendered
	return g.handleTurnEnd()
}

func (g *Game) handleWaittingPlayersConfirmed(evt *types.PlayerReadyEvent) error {
	// stale events - check both round and turn number
	if evt.RoundNumber != g.currentRound.round.RoundNumber {
		return nil
	}
	currentTurnNumber := g.currentRound.getCurrentTurnNumber()
	if evt.TurnNumber != currentTurnNumber {
		return nil
	}
	player, err := g.getGamePlayer(evt.PlayerAddress.TemporaryAddress)
	if err != nil {
		return err
	}
	// Update PlayerTurnInfo for current turn
	playerTurnInfo := player.getCurrentPlayerTurnInfo()
	playerTurnInfo.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_READY

	g.sendEventsToAllPlayers(types.NewEvent(g.workerID(), &types.RoundPartialReadyEvent{
		GameID:       g.gameInfo.ID,
		RoundNumber:  uint32(g.currentRound.round.RoundNumber),
		ReadyAddress: player.PlayerAddress(),
	}))
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

func (g *Game) handleRoomCreated(evt *types.RoomCreated) error {
	defer g.sendTimerEventByCurrentRound()
	// the turn 1 is already created in the round creation
	currentTurn := g.currentRound.getCurrentTurn()
	currentTurn.TurnStartAt = evt.TimeStamp
	// For the first turn of the first round, the contract creation already handles the turn setup
	// so we transition directly to waiting for commitments
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_COMMITMENTS)
	if err := g.saveRound(g.currentRound.round); err != nil {
		return err
	}

	// Send game ready event (only once when contract is created)
	allPlayers := make([]types.PlayerAddress, 0, len(g.currentRound.gamePlayers))
	for _, player := range g.currentRound.gamePlayers {
		allPlayers = append(allPlayers, player.PlayerAddress())
	}
	gameReadyEvt := types.NewEvent(g.workerID(), &types.GameReadyEvent{
		GameID:            g.gameInfo.ID,
		MaxRoundNum:       uint32(g.gameInfo.MaxRounds),
		MaxTurnNum:        3, // 3 turns per round
		InitialHP:         uint32(g.gameInfo.InitialHP),
		InitialMultiplier: uint32(g.gameInfo.InitialMultiplier),
		Players:           allPlayers,
	})
	// Send turn ready event for the first turn
	turnReadyEvt := types.NewEvent(g.workerID(), &types.TurnReadyEvent{
		GameID:                      g.gameInfo.ID,
		RoundNumber:                 uint32(g.currentRound.round.RoundNumber),
		TurnNumber:                  1,
		CommitmentSubmissionTimeout: g.gameInfo.CommitmentSubmissionTimeout,
	})
	// Also send round ready event for the first turn
	roundReadyEvt := types.NewEvent(g.workerID(), &types.RoundReadyEvent{
		GameID:         g.gameInfo.ID,
		RoundNumber:    uint32(g.currentRound.round.RoundNumber),
		RoundStartedAt: evt.TimeStamp,
	})
	g.sendEventsToAllPlayers(gameReadyEvt, roundReadyEvt, turnReadyEvt)
	return nil
}

func (g *Game) handleNewTurnSetupOnChain(evt *types.NewTurnSetupComplete) error {
	defer g.sendTimerEventByCurrentRound()
	if evt.GameID != g.gameInfo.ID {
		return errors.New("invalid game id")
	}
	// stale event - check round and turn number
	if evt.RoundNumber != uint32(g.currentRound.round.RoundNumber) {
		return nil
	}
	if evt.TurnNumber != g.currentRound.getCurrentTurnNumber() {
		return nil
	}

	// turn should already be created
	currentTurn := g.currentRound.getCurrentTurn()
	currentTurn.TurnStartAt = evt.TimeStamp

	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_COMMITMENTS)
	if err := g.saveRound(g.currentRound.round); err != nil {
		return err
	}

	// Send turn ready event
	turnReadyEvt := types.NewEvent(g.workerID(), &types.TurnReadyEvent{
		GameID:                      g.gameInfo.ID,
		RoundNumber:                 evt.RoundNumber,
		TurnNumber:                  evt.TurnNumber,
		CommitmentSubmissionTimeout: g.gameInfo.CommitmentSubmissionTimeout,
	})
	// For the first turn of a round, also send round ready event
	if evt.TurnNumber == 1 {
		roundReadyEvt := types.NewEvent(g.workerID(), &types.RoundReadyEvent{
			GameID:         g.gameInfo.ID,
			RoundNumber:    evt.RoundNumber,
			RoundStartedAt: evt.TimeStamp,
		})
		g.sendEventsToAllPlayers(roundReadyEvt, turnReadyEvt)
	} else {
		g.sendEventsToAllPlayers(turnReadyEvt)
	}
	return nil
}

func (g *Game) handleGameStateWaittingCommitments(evt *types.PlayerCommitmentOnChain) error {
	commitmentIdx, err := g.validateCommitmentSubmission(evt)
	if err != nil {
		return err
	}

	player, err := g.getGamePlayer(evt.Address.TemporaryAddress)
	if err != nil {
		return err
	}

	// it should already be created in the turn creation
	playerTurnInfo := player.getCurrentPlayerTurnInfo()
	playerTurnInfo.TurnSubmittedCard.CommitmentHash = evt.Commitment
	playerTurnInfo.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_SUBMITTED

	// If all players have submitted this commitment index, allow card submission for this index
	if g.haveAllPlayersSubmittedCommitment() {
		// Send CommitmentsOnChain event to notify players they can submit cards for this index
		// commitmentIdx is 0-based, but TurnNumber is 1-based (1, 2, 3)
		turnNumber := commitmentIdx + 1
		commitmentsOnChainEvt := types.NewEvent(g.workerID(), &types.CommitmentsOnChainEvent{
			GameID:                g.gameInfo.ID,
			RoundNumber:           evt.RoundNumber,
			TurnNumber:            turnNumber,
			CardSubmissionTimeout: g.gameInfo.CardSubmissionTimeout,
		})
		// Change status to allow card submission for this turn
		g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_CARDS)
		err = g.saveRound(g.currentRound.round)
		if err != nil {
			return err
		}
		g.sendEventsToAllPlayers(commitmentsOnChainEvt)
	} else {
		err = g.saveRound(g.currentRound.round)
		if err != nil {
			return err
		}
	}
	g.sendTimerEventByCurrentRound()
	return nil
}

func (g *Game) handleGameStateCardSubmitted(evt *types.PlayerCardOnChain) error {
	_, cardEntry, cardID, err := g.validateCardSubmission(evt)
	if err != nil {
		return err
	}
	if cardEntry == nil {
		// Card already submitted, nothing to do
		return nil
	}

	player, err := g.getGamePlayer(evt.Address.TemporaryAddress)
	if err != nil {
		return err
	}

	// Update the current turn's PlayerTurnInfo with CardID and Salt
	playerTurnInfo := player.getCurrentPlayerTurnInfo()
	playerTurnInfo.TurnSubmittedCard.CardID = uint32(cardID)
	playerTurnInfo.TurnSubmittedCard.Salt = evt.Salt
	playerTurnInfo.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED

	// If all players have submitted this card index, handle turn end
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
	// Get card index from current turn number (turn number is 1-based, cardIdx is 0-based)
	turnNumber := g.currentRound.getCurrentTurnNumber()
	cardIdx := int(turnNumber) - 1
	roundNumber := g.currentRound.round.RoundNumber

	// Execute battles for this card index
	isGameOver, gameResult, err := g.currentRound.executeCardIndex()
	if err != nil {
		return fmt.Errorf("failed to execute card index %d: %v", cardIdx, err)
	}
	g.gameInfo.GameResult = gameResult

	// Get or create current Turn record
	currentTurn := g.currentRound.getCurrentTurn()
	currentTurn.RoundID = g.currentRound.round.ID

	// Build PlayerTurnInfo for this turn from playerTurnInfos and save to Turn record
	playerTurnInfos := make([]*types.PlayerTurnInfo, 0, len(g.currentRound.gamePlayers))
	for _, p := range g.currentRound.gamePlayers {
		// Get PlayerTurnInfo for current turn
		playerTurnInfo := p.getCurrentPlayerTurnInfo()
		// Use TurnSubmittedCard directly
		playerTurnInfos = append(playerTurnInfos, &types.PlayerTurnInfo{
			PlayerAddress: p.PlayerAddress(),
			SubmittedCard: playerTurnInfo.TurnSubmittedCard,
		})
	}

	// Determine if round and/or game is complete
	isRoundComplete := turnNumber >= 3 || isGameOver
	isGameComplete := isGameOver

	// Mark this turn as completed (persisted on the Turn record).
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_COMPLETED)

	// Build event payload now, but send only after successful save.
	confirmationTimeout := g.gameInfo.ConfirmationTimeout
	gameContinueTimeout := g.gameInfo.GameContinueTimeout
	turnCompletedEvt := &types.TurnCompletedEvent{
		GameID:              g.gameInfo.ID,
		RoundNumber:         roundNumber,
		TurnNumber:          turnNumber,
		IsRoundComplete:     isRoundComplete,
		IsGameComplete:      isGameComplete,
		PlayerTurnInfo:      playerTurnInfos,
		GameResult:          gameResult, // will be nil if game is not complete
		ConfirmationTimeout: &confirmationTimeout,
	}
	if isGameComplete {
		turnCompletedEvt.GameContinueTimeout = &gameContinueTimeout
	}

	// If round is complete, mark round completion and persist first.
	if isRoundComplete {
		g.currentRound.round.CompleteReason = proto.RoundCompleteReason_ROUND_COMPLETE_NORMAL
		g.currentRound.setTurnStatus(proto.TurnStatus_TURN_ROUND_COMPLETED)

		if isGameComplete {
			g.currentRound.round.IsLastRound = true
			g.gameInfo.Status = proto.GameStatus_GAME_END
			if err := g.saveGame(); err != nil {
				return err
			}
			// Send turn-completed only after game/round state is saved.
			g.sendEventsToAllPlayers(types.NewEvent(g.workerID(), turnCompletedEvt))
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

		// Game continues: create new round and persist, then notify.
		g.setupNewRound()
		if err := g.saveGame(); err != nil {
			return err
		}
		g.sendEventsToAllPlayers(types.NewEvent(g.workerID(), turnCompletedEvt))
		return nil
	}

	// Otherwise, prepare for the next turn
	g.incrementTurnNumber()
	// Create the next turn record immediately after turn completion
	g.currentRound.createNewTurn()
	if err = g.saveRound(g.currentRound.round); err != nil {
		return err
	}
	g.sendEventsToAllPlayers(types.NewEvent(g.workerID(), turnCompletedEvt))
	g.sendTimerEventByCurrentRound()
	return nil
}

// handleSubmitPlayerCommitment handles the SubmitPlayerCommitment event: validate then enqueue to tx pool
func (g *Game) handleSubmitPlayerCommitment(reqEvt *types.SubmitPlayerCommitment) error {
	if err := g.validatePlayerCommitment(reqEvt); err != nil {
		return err
	}
	valid, err := utils.Verify(
		[]any{g.gameInfo.ID, reqEvt.RoundNumber, reqEvt.CommitmentIndex, reqEvt.Commitment},
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
func (g *Game) handleSubmitPlayerCard(reqEvt *types.SubmitPlayerCard) error {
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
			reqEvt.Card,
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
		[]any{g.gameInfo.ID, reqEvt.RoundNumber, reqEvt.CardIndex, reqEvt.Card, reqEvt.Salt},
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
