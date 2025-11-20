package game

import (
	"fmt"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
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

	if index == 0 || index > 3 {
		log.Errorw("invalid index", "type", indexType, "game id", g.gameInfo.ID, "round number", roundNumber, "player address", playerAddr, "index", index)
		return fmt.Errorf("%s index must be between 1 and 3, got %d", indexType, index)
	}

	expectedIndex := g.getCurrentTurnNumber()
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
	playerTurnInfo := player.getPlayerTurnInfoForTurn(turnNumber)
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

	if evt.Card == 0 {
		return fmt.Errorf("card ID cannot be zero")
	}

	if len(evt.Salt) == 0 {
		return fmt.Errorf("salt cannot be empty")
	}

	cardEntry, err := g.getAndValidateCardEntry(evt.Address.TemporaryAddress, evt.CardIndex)
	if err != nil {
		return err
	}

	if cardEntry.CardID != 0 {
		cardIdx := evt.CardIndex - 1
		log.Errorw("card already submitted", "game id", g.gameInfo.ID, "player address", evt.Address.TemporaryAddress, "card index", cardIdx)
		return fmt.Errorf("card already submitted for index %d", evt.CardIndex)
	}

	return nil
}
