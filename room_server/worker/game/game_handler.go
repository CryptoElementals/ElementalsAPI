package game

import (
	"context"
	"errors"
	"fmt"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/utils"
	"github.com/ethereum/go-ethereum/common"
)

// Handle is the main entry point for handling events
// Note: Events are processed sequentially by the worker, so no lock is needed
func (g *Game) Handle(ctx context.Context, event *types.Event) error {
	// Handle request events that work regardless of game status using type switch
	switch evt := event.Data.(type) {
	case *timerEvent:
		g.handleTimerEvent(evt)
		return nil
	case *types.GetGameInfoRequest:
		return g.handleGetGameInfoRequest(event)
	case *types.GetBattleInfoRequest:
		return g.handleGetBattleInfoRequest(event, evt)
	case *types.GetGamePhaseRequest:
		return g.handleGetGamePhaseRequest(event)
	case *types.SyncGamePhaseRequest:
		return g.handleSyncGamePhaseRequest(evt)
	case *types.GetGameResultRequest:
		return g.handleGetGameResultRequest(event)
	case *types.SubmitPlayerCommitment:
		return g.handleSubmitPlayerCommitment(evt)
	case *types.SubmitPlayerCard:
		return g.handleSubmitPlayerCard(evt)
	}

	// Handle game state-specific events
	switch g.gameInfo.Status {
	case proto.GameStatus_GAME_INIT, proto.GameStatus_GAME_RUNNING:
		return g.handleTurn(event)
	case proto.GameStatus_GAME_END:
		return errors.New("game has ended")
	}
	return fmt.Errorf("invalid game status: %d", g.gameInfo.Status)
}

func (g *Game) handleTurn(event *types.Event) error {
	if surrenderEvt, err := types.AssertInterface[*types.SurrenderEvent](event); err == nil {
		return g.handleSurrenderEvent(surrenderEvt)
	}

	switch g.currentRound.turnStatus {
	case proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION:
		return g.handleWaittingPlayersConfirmed(event)
	case proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN:
		return g.handleGameStateWaittingSetupOnChain(event)
	case proto.TurnStatus_TURN_WAITTING_COMMITMENTS:
		return g.handleGameStateWaittingCommitments(event)
	case proto.TurnStatus_TURN_WAITTING_CARDS:
		return g.handleGameStateCardSubmitted(event)
	}
	return nil
}

func (g *Game) handleSurrenderEvent(event *types.SurrenderEvent) error {
	p, err := g.getGamePlayer(event.Address.TemporaryAddress)
	if err != nil {
		return err
	}
	// Mark surrender in current turn's PlayerTurnInfo
	playerTurnInfo := p.getCurrentPlayerTurnInfo()
	playerTurnInfo.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_UNKNOWN // Use a status to indicate surrender
	// Check if game is over
	isGameComplete := g.gameInfo.GameResult != nil
	return g.completeRoundAndCheckGameEnd(proto.RoundCompleteReason_ROUND_COMPLETE_PLAYER_SURRENDER, isGameComplete)
}

func (g *Game) handleWaittingPlayersConfirmed(event *types.Event) error {
	evt, err := types.AssertInterface[*types.PlayerReadyEvent](event)
	if err != nil {
		return err
	}
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
		g.currentRound.turnStatus = proto.TurnStatus_TURN_WAITTING_SETUP_ON_CHAIN
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

func (g *Game) handleGameStateWaittingSetupOnChain(event *types.Event) error {
	defer g.sendTimerEventByCurrentRound()
	// First round, first turn needs contract creation
	if g.currentRound.round.RoundNumber == 1 && g.currentRound.getCurrentTurnNumber() == 1 {
		return g.handleRoomCreated(event)
	}
	// For all other turns, handle turn setup completion
	return g.handleNewTurnSetupOnChain(event)
}

func (g *Game) handleRoomCreated(event *types.Event) error {
	evt, err := types.AssertInterface[*types.RoomCreated](event)
	if err != nil {
		return err
	}
	// the turn 1 is already created in the round creation
	currentTurn := g.currentRound.getCurrentTurn()
	currentTurn.TurnStartAt = evt.TimeStamp
	// For the first turn of the first round, the contract creation already handles the turn setup
	// so we transition directly to waiting for commitments
	g.currentRound.turnStatus = proto.TurnStatus_TURN_WAITTING_COMMITMENTS
	err = g.saveRound(g.currentRound.round)
	if err != nil {
		return err
	}

	// Send game ready event (only once when contract is created)
	gameReadyEvt := types.NewEvent(g.workerID(), &types.GameReadyEvent{
		GameID: g.gameInfo.ID,
	})
	// Send turn ready event for the first turn
	turnReadyEvt := types.NewEvent(g.workerID(), &types.TurnReadyEvent{
		GameID:      g.gameInfo.ID,
		RoundNumber: uint32(g.currentRound.round.RoundNumber),
		TurnNumber:  1,
	})
	// Also send round ready event for the first turn
	roundReadyEvt := types.NewEvent(g.workerID(), &types.RoundReadyEvent{
		GameID:         g.gameInfo.ID,
		RoundNumber:    uint32(g.currentRound.round.RoundNumber),
		RoundStartedAt: evt.TimeStamp,
		RoundTimeout:   g.gameInfo.RoundTimeout,
	})
	g.sendEventsToAllPlayers(gameReadyEvt, roundReadyEvt, turnReadyEvt)
	return nil
}

func (g *Game) handleNewTurnSetupOnChain(event *types.Event) error {
	evt, err := types.AssertInterface[*types.NewTurnSetupComplete](event)
	if err != nil {
		return err
	}
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

	g.currentRound.turnStatus = proto.TurnStatus_TURN_WAITTING_COMMITMENTS
	err = g.saveRound(g.currentRound.round)
	if err != nil {
		return err
	}

	// Send turn ready event
	turnReadyEvt := types.NewEvent(g.workerID(), &types.TurnReadyEvent{
		GameID:      g.gameInfo.ID,
		RoundNumber: evt.RoundNumber,
		TurnNumber:  evt.TurnNumber,
	})
	// For the first turn of a round, also send round ready event
	if evt.TurnNumber == 1 {
		roundReadyEvt := types.NewEvent(g.workerID(), &types.RoundReadyEvent{
			GameID:         g.gameInfo.ID,
			RoundNumber:    evt.RoundNumber,
			RoundStartedAt: evt.TimeStamp,
			RoundTimeout:   g.gameInfo.RoundTimeout,
		})
		g.sendEventsToAllPlayers(roundReadyEvt, turnReadyEvt)
	} else {
		g.sendEventsToAllPlayers(turnReadyEvt)
	}
	return nil
}

func (g *Game) handleGameStateWaittingCommitments(event *types.Event) error {
	evt, err := types.AssertInterface[*types.PlayerCommitmentOnChain](event)
	if err != nil {
		return err
	}

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
			GameID:      g.gameInfo.ID,
			RoundNumber: evt.RoundNumber,
			TurnNumber:  turnNumber,
		})
		g.sendEventsToAllPlayers(commitmentsOnChainEvt)

		// Change status to allow card submission for this turn
		g.currentRound.turnStatus = proto.TurnStatus_TURN_WAITTING_CARDS

	}
	err = g.saveRound(g.currentRound.round)
	if err != nil {
		return err
	}
	g.sendTimerEventByCurrentRound()
	return nil
}

func (g *Game) handleGameStateCardSubmitted(event *types.Event) error {
	// set player cards and player status
	evt, err := types.AssertInterface[*types.PlayerCardOnChain](event)
	if err != nil {
		return err
	}

	cardIdx, cardEntry, cardID, err := g.validateCardSubmission(evt)
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
	if func() bool {
		var _ uint32 = cardIdx
		return g.haveAllPlayersSubmittedCard()
	}() {
		return g.handleTurnEnd()
	}
	err = g.saveRound(g.currentRound.round)
	if err != nil {
		return err
	}
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

	// Send event to both players with TurnCompletedEvent (includes round/game completion info)
	turnCompletedEvt := &types.TurnCompletedEvent{
		GameID:          g.gameInfo.ID,
		RoundNumber:     roundNumber,
		TurnNumber:      turnNumber,
		IsRoundComplete: isRoundComplete,
		IsGameComplete:  isGameComplete,
		PlayerTurnInfo:  playerTurnInfos,
		GameResult:      gameResult, // will be nil if game is not complete
	}
	g.sendEventsToAllPlayers(types.NewEvent(g.workerID(), turnCompletedEvt))

	// If round or game is complete, handle it
	if isRoundComplete {
		return g.completeRoundAndCheckGameEnd(proto.RoundCompleteReason_ROUND_COMPLETE_NORMAL, isGameComplete)
	}

	// Otherwise, prepare for the next turn
	g.incrementTurnNumber()
	// Create the next turn record immediately after turn completion
	g.currentRound.createNewTurn()
	if err = g.saveRound(g.currentRound.round); err != nil {
		return err
	}
	g.sendTimerEventByCurrentRound()
	return nil
}

// completeRoundAndCheckGameEnd handles round completion and checks if game should end
// This function consolidates the logic from handleRoundEnd and handleGameEnd
func (g *Game) completeRoundAndCheckGameEnd(reason proto.RoundCompleteReason, isGameComplete bool) error {
	// Mark round as completed
	g.currentRound.round.CompleteReason = reason
	g.currentRound.turnStatus = proto.TurnStatus_TURN_ROUND_COMPLETED

	// If game is complete, handle game end
	if isGameComplete {
		g.currentRound.round.IsLastRound = true
		g.gameInfo.Status = proto.GameStatus_GAME_END
		err := g.saveGame()
		if err != nil {
			return err
		}
		// Call HandleGameCompletedEvent for game result settlement
		completeEvt := &types.GameCompletedEvent{
			GameID:   g.gameInfo.ID,
			GameInfo: g.gameInfo,
		}
		if err := g.gameContextHandler.HandleGameCompletedEvent(completeEvt); err != nil {
			log.Errorw("handle game complete event failed", "err", err, "game id", g.gameInfo.ID)
		}
		g.stopGame()
		return nil
	}

	// Game continues, setup new round
	g.setupNewRound()
	err := g.saveGame()
	if err != nil {
		return err
	}
	return nil
}

// handleGetGameInfoRequest handles the GetGameInfoRequest event and sends back a response
func (g *Game) handleGetGameInfoRequest(event *types.Event) error {
	// Get the game info (lock is already held by Handle)
	gameProto := conversion.DbGameInfoToProtoGameInfo(g.gameInfo)

	// Send response through AckChan
	if event.AckChan != nil {
		event.AckChan <- gameProto
	}
	return nil
}

// handleGetBattleInfoRequest handles the GetBattleInfoRequest event and sends back a response
func (g *Game) handleGetBattleInfoRequest(event *types.Event, reqEvt *types.GetBattleInfoRequest) error {
	// Get the battle info (lock is already held by Handle)
	var gameRes *proto.GameResult
	if g.gameInfo.GameResult != nil {
		gameRes = conversion.DbGameResultToProtoGameResult(g.gameInfo.GameResult)
		gameRes.GameContinueTimeout = uint64(g.gameInfo.GameArgs.ContinueTimeout)
	}
	var roundRes *proto.RoundResult
	for _, round := range g.gameInfo.Rounds {
		if round.RoundNumber == reqEvt.RoundNumber {
			roundRes = conversion.DbRoundToRoundResult(round)
			roundRes.RoundConfirmTimeout = uint64(g.gameInfo.GameArgs.RoundConfirmTimeout)
			break
		}
	}

	// Send response through AckChan
	if event.AckChan != nil {
		event.AckChan <- &types.GetBattleInfoResponse{
			RoundResult: roundRes,
			GameResult:  gameRes,
		}
	}
	return nil
}

// handleGetGamePhaseRequest handles the GetGamePhaseRequest event and sends back a response
func (g *Game) handleGetGamePhaseRequest(event *types.Event) error {
	// Get the game phase (lock is already held by Handle)
	turnNumber := g.currentRound.getCurrentTurnNumber()
	turnStartAt := int64(0)
	currentTurn := g.currentRound.getCurrentTurn()
	if currentTurn != nil && currentTurn.TurnStartAt > 0 {
		turnStartAt = currentTurn.TurnStartAt
	} else {
		turnStartAt = g.currentRound.round.CreatedAt.Unix()
	}
	gamePhase := conversion.DbGameToProtoGamePhase(g.gameInfo, g.currentRound.round, turnNumber, turnStartAt)

	// Send response through AckChan
	if event.AckChan != nil {
		event.AckChan <- gamePhase
	}
	return nil
}

// handleSyncGamePhaseRequest handles the SyncGamePhaseRequest event and sends game phase directly to receiver
func (g *Game) handleSyncGamePhaseRequest(reqEvt *types.SyncGamePhaseRequest) error {
	// Get the game phase (lock is already held by Handle)
	turnNumber := g.currentRound.getCurrentTurnNumber()
	turnStartAt := int64(0)
	currentTurn := g.currentRound.getCurrentTurn()
	if currentTurn != nil && currentTurn.TurnStartAt > 0 {
		turnStartAt = currentTurn.TurnStartAt
	} else {
		turnStartAt = g.currentRound.round.CreatedAt.Unix()
	}
	gamePhase := conversion.DbGameToProtoGamePhase(g.gameInfo, g.currentRound.round, turnNumber, turnStartAt)

	// Send game phase directly to receiver via workerManager
	if reqEvt.Receiver != nil {
		syncEvt := types.NewEvent(types.GAME_MANAGER_ID, &types.GamePhaseSyncEvent{
			GamePhase: gamePhase,
		})
		g.workerMangerService.SendEvent(reqEvt.Receiver.String(), syncEvt)
	}
	return nil
}

// handleGetGameResultRequest handles the GetGameResultRequest event and sends back a response
func (g *Game) handleGetGameResultRequest(event *types.Event) error {
	// Get the game result (lock is already held by Handle)
	var gameRes *proto.GameResult
	if g.gameInfo.GameResult != nil {
		gameRes = conversion.DbGameResultToProtoGameResult(g.gameInfo.GameResult)
	}

	// Send response through AckChan
	if event.AckChan != nil {
		event.AckChan <- gameRes
	}
	return nil
}

// handleSubmitPlayerCommitment handles the SubmitPlayerCommitment event
func (g *Game) handleSubmitPlayerCommitment(reqEvt *types.SubmitPlayerCommitment) error {
	// Validate the event - return error if validation fails, nil if valid
	// The worker will automatically send the error to AckChan if present
	if err := g.validatePlayerCommitment(reqEvt); err != nil {
		return err
	}
	// verify signature for: game id, round number, commitment index, commitment
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
	return nil
}

// handleSubmitPlayerCard handles the SubmitPlayerCard event
func (g *Game) handleSubmitPlayerCard(reqEvt *types.SubmitPlayerCard) error {
	// Validate the event - return error if validation fails, nil if valid
	// The worker will automatically send the error to AckChan if present
	if err := g.validatePlayerCard(reqEvt); err != nil {
		return err
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
	return nil
}
