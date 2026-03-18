package game

import (
	"fmt"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

// validateCommitmentSubmission validates the commitment submission including round number and returns the 0-based index
func (g *Game) validateCommitmentSubmission(evt *types.PlayerCommitmentOnChain) (uint32, error) {
	if err := g.validateRoundAndIndex(evt.RoundNumber, evt.CommitmentIndex, evt.Address.TemporaryAddress, "commitment"); err != nil {
		return 0, err
	}
	return evt.CommitmentIndex - 1, nil // Convert to 0-based
}

// validateCardSubmission validates the card submission including round number and returns the 0-based index, card entry, card ID, and error
// Returns (cardIdx, cardEntry, cardID, error). If card is already submitted, cardEntry will be nil.
func (g *Game) validateCardSubmission(evt *types.PlayerCardOnChain) (uint32, *dao.TurnSubmittedCard, uint, error) {
	if err := g.validateRoundAndIndex(evt.RoundNumber, evt.CardIndex, evt.Address.TemporaryAddress, "card"); err != nil {
		return 0, nil, 0, err
	}

	if evt.Card == 0 {
		return 0, nil, 0, fmt.Errorf("card ID cannot be zero")
	}

	cardEntry, err := g.getAndValidateCardEntry(evt.Address.TemporaryAddress, evt.CardIndex)
	if err != nil {
		return 0, nil, 0, err
	}

	// Check if card is already submitted (CardID != 0)
	if cardEntry.CardID != 0 {
		cardIdx := evt.CardIndex - 1
		log.Errorw("card already submitted", "game id", g.gameInfo.ID, "player address", evt.Address.TemporaryAddress, "card index", cardIdx)
		return cardIdx, nil, 0, nil // Return nil cardEntry to indicate already submitted
	}

	// Return cardID from evt since cardEntry might not have CardID set yet
	return evt.CardIndex - 1, cardEntry, evt.Card, nil
}

// validateRoundAndIndex validates round number and index (1-based, 1-3) against current round and turn
func (g *Game) validateRoundAndIndex(roundNumber, index uint32, playerAddr string, indexType string) error {
	if roundNumber != g.currentRound.round.RoundNumber {
		log.Errorw("stale event", "type", indexType, "game id", g.gameInfo.ID, "expected round", g.currentRound.round.RoundNumber, "got round", roundNumber, "player address", playerAddr)
		return fmt.Errorf("round number mismatch: expected %d, got %d", g.currentRound.round.RoundNumber, roundNumber)
	}

	expectedIndex := g.currentRound.getCurrentTurnNumber()
	if index != expectedIndex {
		log.Errorw("index mismatch", "type", indexType, "game id", g.gameInfo.ID, "round number", roundNumber, "player address", playerAddr, "expected index", expectedIndex, "got index", index)
		return fmt.Errorf("expected %s index %d, got %d", indexType, expectedIndex, index)
	}

	return nil
}

// getAndValidateCardEntry gets the player's card entry and validates it exists with a commitment
func (g *Game) getAndValidateCardEntry(playerAddr string, cardIndex uint32) (*dao.TurnSubmittedCard, error) {
	player, err := g.getGamePlayer(playerAddr)
	if err != nil {
		log.Errorw("player not found", "game id", g.gameInfo.ID, "player address", playerAddr, "error", err)
		return nil, fmt.Errorf("player not found: %w", err)
	}

	turnNumber := cardIndex // cardIndex is 1-based, same as turnNumber
	var _ uint32 = turnNumber
	playerTurnInfo := player.getCurrentPlayerTurnInfo()
	if playerTurnInfo == nil || playerTurnInfo.TurnSubmittedCard == nil {
		log.Errorw("commitment not submitted", "game id", g.gameInfo.ID, "player address", playerAddr, "card index", cardIndex)
		return nil, fmt.Errorf("commitment for card index %d must be submitted before card", cardIndex)
	}

	if len(playerTurnInfo.TurnSubmittedCard.CommitmentHash) == 0 {
		log.Errorw("commitment not submitted", "game id", g.gameInfo.ID, "player address", playerAddr, "card index", cardIndex)
		return nil, fmt.Errorf("commitment for card index %d must be submitted before card", cardIndex)
	}

	return playerTurnInfo.TurnSubmittedCard, nil
}

// validatePlayerCommitment validates the commitment using similar logic as validateCommitmentSubmission
func (g *Game) validatePlayerCommitment(evt *types.SubmitPlayerCommitment) error {
	if err := g.validateRoundAndIndex(evt.RoundNumber, evt.CommitmentIndex, evt.Address.TemporaryAddress, "commitment"); err != nil {
		return err
	}

	// check if the turn is waiting for commitments
	if g.currentRound.getTurnStatus() != proto.TurnStatus_TURN_WAITTING_COMMITMENTS {
		log.Errorw("turn is not waiting for commitments", "game id", g.gameInfo.ID, "player address", evt.Address.TemporaryAddress, "turn status", g.currentRound.getTurnStatus())
		return fmt.Errorf("turn is not waiting for commitments")
	}

	if len(evt.Commitment) == 0 {
		return fmt.Errorf("commitment cannot be empty")
	}

	return nil
}

// validatePlayerCard validates the card using similar logic as validateCardSubmission
func (g *Game) validatePlayerCard(evt *types.SubmitPlayerCard) error {
	if err := g.validateRoundAndIndex(evt.RoundNumber, evt.CardIndex, evt.Address.TemporaryAddress, "card"); err != nil {
		return err
	}

	// check if the turn is waiting for card
	if g.currentRound.getTurnStatus() != proto.TurnStatus_TURN_WAITTING_CARDS {
		log.Errorw("turn is not waiting for cards", "game id", g.gameInfo.ID, "player address", evt.Address.TemporaryAddress, "turn status", g.currentRound.getTurnStatus())
		return fmt.Errorf("turn is not waiting for cards")
	}

	if evt.Card == 0 {
		return fmt.Errorf("card ID cannot be zero")
	}

	if len(evt.Salt) == 0 {
		return fmt.Errorf("salt cannot be empty")
	}

	return nil
}
