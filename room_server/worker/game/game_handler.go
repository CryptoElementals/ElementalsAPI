package game

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
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
		var _ *types.GetGameInfoRequest = evt
		return g.handleGetGameInfoRequest(event)
	case *types.GetBattleInfoRequest:
		return g.handleGetBattleInfoRequest(event, evt)
	case *types.GetGamePhaseRequest:
		var _ *types.GetGamePhaseRequest = evt
		return g.handleGetGamePhaseRequest(event)
	case *types.GetGameResultRequest:
		var _ *types.GetGameResultRequest = evt
		return g.handleGetGameResultRequest(event)
	case *types.SubmitPlayerCommitment:
		return g.handleSubmitPlayerCommitment(evt)
	case *types.SubmitPlayerCard:
		return g.handleSubmitPlayerCard(evt)
	}

	// Handle game state-specific events
	switch g.gameInfo.Status {
	case proto.GameStatus_GAME_INIT, proto.GameStatus_GAME_RUNNING:
		err := g.handleRound(event)
		if err != nil {
			log.Errorf("handleRound failed, err: %v", err)
			return err
		}
		return nil
	case proto.GameStatus_GAME_END:
		return errors.New("game has ended")
	}
	return fmt.Errorf("invalid game status: %d", g.gameInfo.Status)
}

func (g *Game) handleRound(event *types.Event) error {
	currentRound := g.currentRound.round
	if surrenderEvt, err := types.AssertInterface[*types.SurrenderEvent](event); err == nil {
		return g.handleSurrenderEvent(surrenderEvt)
	}

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

func (g *Game) handleSurrenderEvent(event *types.SurrenderEvent) error {
	p, err := g.getGamePlayer(event.Address.TemporaryAddress)
	if err != nil {
		return err
	}
	p.roundPlayer.Surrendered = true
	g.savePlayerRoundInfo(p.roundPlayer)
	return g.handleRoundEnd(proto.RoundCompleteReason_ROUND_COMPLETE_PLAYER_SURRENDER)
}

func (g *Game) handleWaittingRoundPlayersConfirmed(event *types.Event) error {
	evt, err := types.AssertInterface[*types.PlayerReadyEvent](event)
	if err != nil {
		return err
	}
	// stale events
	if evt.RoundNumber != g.currentRound.round.RoundNumber {
		return nil
	}
	// might be a chain error, ignore it
	player, err := g.getGamePlayer(evt.PlayerAddress.TemporaryAddress)
	if err != nil {
		log.Errorf("getGamePlayer failed, err: %v", err)
		return err
	}
	player.roundPlayer.PlayerReady = true
	g.sendEventsToAllPlayers(types.NewEvent(g.workerID(), &types.RoundPartialReadyEvent{
		GameID:       g.gameInfo.ID,
		RoundNumber:  uint32(g.currentRound.round.RoundNumber),
		ReadyAddress: player.PlayerAddress(),
	}))
	if !g.areAllPlayersReady() {
		g.savePlayerRoundInfo(player.roundPlayer)
		return nil
	}

	// All players confirmed battle for this turn
	// The first round, first turn needs to create contract first
	if g.currentRound.round.RoundNumber == 1 && g.getCurrentTurnNumber() == 1 {
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
		g.currentRound.round.Status = proto.RoundStatus_ROUND_WAITTING_SETUP_ON_CHAIN
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
	if g.currentRound.round.RoundNumber == 1 && g.getCurrentTurnNumber() == 1 {
		return g.handleRoomContractCreated(event)
	}
	// For all other turns, handle turn setup completion
	return g.handleNewTurnSetupOnChain(event)
}

func (g *Game) handleRoomContractCreated(event *types.Event) error {
	evt, err := types.AssertInterface[*types.RoomContractCreated](event)
	if err != nil {
		return err
	}
	g.gameInfo.RoomContract = evt.RoomContractAddress
	g.currentRound.round.SetupOnChainAt = evt.TimeStamp
	err = g.saveGame()
	if err != nil {
		return err
	}

	// Send game ready event (only once when contract is created)
	gameReadyEvt := types.NewEvent(g.workerID(), &types.GameReadyEvent{
		GameID:          g.gameInfo.ID,
		ContractAddress: evt.RoomContractAddress,
	})
	g.sendEventsToAllPlayers(gameReadyEvt)

	// For the first turn of the first round, the contract creation already handles the turn setup
	// so we transition directly to waiting for commitments
	g.currentRound.round.Status = proto.RoundStatus_ROUND_WAITTING_COMMITMENTS
	err = g.saveRound(g.currentRound.round)
	if err != nil {
		return err
	}

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
	g.sendEventsToAllPlayers(roundReadyEvt, turnReadyEvt)
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
	if evt.TurnNumber != g.getCurrentTurnNumber() {
		return nil
	}
	g.currentRound.round.Status = proto.RoundStatus_ROUND_WAITTING_COMMITMENTS
	g.currentRound.round.SetupOnChainAt = evt.TimeStamp
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

	// Store commitment directly in SubmittedCards
	// Since commitmentIdx == len(SubmittedCards) (validated above), append a new entry
	player.roundPlayer.SubmittedCards = append(player.roundPlayer.SubmittedCards, &dao.RoundSubmittedCard{
		CardNumber:          commitmentIdx + 1,
		SubmittedCommitment: evt.Commitment,
	})

	g.savePlayerRoundInfo(player.roundPlayer)

	// If all players have submitted this commitment index, allow card submission for this index
	if g.haveAllPlayersSubmittedCommitment(commitmentIdx) {
		// Send CommitmentsOnChain event to notify players they can submit cards for this index
		commitmentsOnChainEvt := types.NewEvent(g.workerID(), &types.CommitmentsOnChainEvent{
			GameID:      g.gameInfo.ID,
			RoundNumber: evt.RoundNumber,
		})
		g.sendEventsToAllPlayers(commitmentsOnChainEvt)

		// Change status to allow card submission for this turn
		g.currentRound.round.Status = proto.RoundStatus_ROUND_WAITTING_CARDS
		err = g.saveRound(g.currentRound.round)
		if err != nil {
			return err
		}
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

	// Update the existing card entry with CardID and Salt
	cardEntry.CardID = cardID
	cardEntry.Salt = evt.Salt

	g.savePlayerRoundInfo(player.roundPlayer)

	// If all players have submitted this card index, handle turn end
	if g.haveAllPlayersSubmittedCard(cardIdx) {
		return g.handleTurnEnd()
	}

	return nil
}

// areAllPlayersReady checks if all players have confirmed battle for the current round
func (g *Game) areAllPlayersReady() bool {
	for _, p := range g.currentRound.gamePlayers {
		if !p.roundPlayer.PlayerReady {
			return false
		}
	}
	return true
}

// haveAllPlayersSubmittedCommitment checks if all players have submitted a commitment at the given index
func (g *Game) haveAllPlayersSubmittedCommitment(commitmentIdx uint32) bool {
	for _, p := range g.currentRound.gamePlayers {
		if int(commitmentIdx) >= len(p.roundPlayer.SubmittedCards) {
			return false
		}
		card := p.roundPlayer.SubmittedCards[commitmentIdx]
		if len(card.SubmittedCommitment) == 0 {
			return false
		}
	}
	return true
}

// haveAllPlayersSubmittedCard checks if all players have submitted a card at the given index
func (g *Game) haveAllPlayersSubmittedCard(cardIdx uint32) bool {
	for _, p := range g.currentRound.gamePlayers {
		if int(cardIdx) >= len(p.roundPlayer.SubmittedCards) {
			return false
		}
		if p.roundPlayer.SubmittedCards[cardIdx].CardID == 0 {
			return false
		}
	}
	return true
}

// handleTurnEnd handles the end of a turn: executes the card, sends events, and checks if round/game ends
func (g *Game) handleTurnEnd() error {
	// Get card index from current turn number (turn number is 1-based, cardIdx is 0-based)
	turnNumber := g.getCurrentTurnNumber()
	cardIdx := int(turnNumber) - 1
	roundNumber := g.currentRound.round.RoundNumber

	// Execute battles for this card index
	isGameOver, gameResult, err := g.currentRound.ExecuteCardIndex(cardIdx)
	if err != nil {
		return fmt.Errorf("failed to execute card index %d: %v", cardIdx, err)
	}
	g.gameInfo.GameResult = gameResult
	// Build PlayerTurnInfo for this turn
	playerTurnInfos := make([]*types.PlayerTurnInfo, 0, len(g.currentRound.gamePlayers))
	for _, p := range g.currentRound.gamePlayers {
		if cardIdx < len(p.roundPlayer.SubmittedCards) {
			playerTurnInfos = append(playerTurnInfos, &types.PlayerTurnInfo{
				PlayerAddress: p.PlayerAddress(),
				SubmittedCard: p.roundPlayer.SubmittedCards[cardIdx],
			})
		}
	}

	// Send events to both players (CardsOnChainEvent and TurnCompletedEvent)
	g.sendEventsToAllPlayers(
		types.NewEvent(g.workerID(), &types.CardsOnChainEvent{
			GameID:      g.gameInfo.ID,
			RoundNumber: roundNumber,
		}),
		types.NewEvent(g.workerID(), &types.TurnCompletedEvent{
			GameID:         g.gameInfo.ID,
			RoundNumber:    roundNumber,
			TurnNumber:     turnNumber,
			PlayerTurnInfo: playerTurnInfos,
		}),
	)

	// Check if we've reached 3 turns in this round or game is over
	if isGameOver || turnNumber >= 3 {
		// All 3 turns completed, handle round end
		return g.handleRoundEnd(proto.RoundCompleteReason_ROUND_COMPLETE_NORMAL)
	}

	// Otherwise, prepare for the next turn
	g.incrementTurnNumber()
	for _, p := range g.currentRound.gamePlayers {
		p.roundPlayer.PlayerReady = false
	}
	g.currentRound.round.Status = proto.RoundStatus_ROUND_WAITTING_BATTLE_CONFIRMATION
	err = g.saveRound(g.currentRound.round)
	if err != nil {
		return err
	}
	g.sendTimerEventByCurrentRound()
	return nil
}

// handleRoundEnd handles the end of a round: gets game result and checks if game ends
func (g *Game) handleRoundEnd(reason proto.RoundCompleteReason) error {
	g.currentRound.round.CompleteReason = reason
	g.currentRound.round.RoundEndTime = time.Now().Unix()

	// Check if game is over (game result is set when game ends)
	isGameOver := g.gameInfo.GameResult != nil

	// Start a new round
	g.currentRound.round.Status = proto.RoundStatus_ROUND_COMPLETED
	roundCompletedEvt := types.NewEvent(g.workerID(), &types.RoundCompletedEvent{
		GameID:    g.gameInfo.ID,
		RoundInfo: g.currentRound.round,
	})
	g.sendEventsToAllPlayers(roundCompletedEvt)

	if isGameOver {
		return g.handleGameEnd()
	}
	g.setupNewRound()
	err := g.saveGame()
	if err != nil {
		return err
	}
	return nil
}

// sendGameCompletedEventAndStop sends game completed event and stops the game
func (g *Game) sendGameCompletedEventAndStop() {
	completeEvt := &types.GameCompletedEvent{
		GameID:   g.gameInfo.ID,
		GameInfo: g.gameInfo,
	}
	gameCompletedEvt := types.NewEvent(g.workerID(), completeEvt)
	if err := g.gameContextHandler.HandleGameCompletedEvent(completeEvt); err != nil {
		log.Errorw("handle game complete event failed", "err", err, "game id", g.gameInfo.ID)
	}
	g.sendEventsToAllPlayers(gameCompletedEvt)
	g.stopGame()
}

// can go into game end from any other status
func (g *Game) handleGameEnd() error {
	g.currentRound.round.Status = proto.RoundStatus_ROUND_COMPLETED
	g.currentRound.round.IsLastRound = true
	g.gameInfo.Status = proto.GameStatus_GAME_END
	err := g.saveGame()
	if err != nil {
		return err
	}
	g.sendGameCompletedEventAndStop()
	return nil
}

// can go into game end from any other status
func (g *Game) handleGameAbortInit() error {
	log.Infow("game aborted", "game id", g.gameInfo.ID)
	if g.gameInfo.Status != proto.GameStatus_GAME_INIT {
		return fmt.Errorf("invalid game status: %d", g.gameInfo.Status)
	}
	g.currentRound.round.IsLastRound = true
	g.gameInfo.Status = proto.GameStatus_GAME_ABORTED
	g.gameInfo.GameResult = g.abortedGameResult()
	err := g.saveGame()
	if err != nil {
		return err
	}
	g.sendGameCompletedEventAndStop()
	return nil
}

// can go into game end from any other status
func (g *Game) handleGameAbortInternalError() error {
	log.Infow("game aborted with internal error", "game id", g.gameInfo.ID)
	if g.currentRound.round != nil {
		g.currentRound.round.IsLastRound = true
		g.currentRound.round.Status = proto.RoundStatus_ROUND_COMPLETED
	}

	g.gameInfo.Status = proto.GameStatus_GAME_ABORTED
	g.gameInfo.GameResult = g.abortedGameResult()
	err := g.saveGame()
	if err != nil {
		return err
	}
	g.sendGameCompletedEventAndStop()
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
	gamePhase := conversion.DbGameToProtoGamePhase(g.gameInfo, g.currentRound.round)

	// Send response through AckChan
	if event.AckChan != nil {
		event.AckChan <- gamePhase
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
	return g.validatePlayerCommitment(reqEvt)
}

// handleSubmitPlayerCard handles the SubmitPlayerCard event
func (g *Game) handleSubmitPlayerCard(reqEvt *types.SubmitPlayerCard) error {
	// Validate the event - return error if validation fails, nil if valid
	// The worker will automatically send the error to AckChan if present
	return g.validatePlayerCard(reqEvt)
}
