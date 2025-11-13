package game

import (
	"fmt"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

// validateCommitmentSubmission validates the commitment submission including round number and returns the 0-based index
func (g *Game) validateCommitmentSubmission(evt *types.PlayerCommitmentOnChain) (uint32, error) {
	// Check round number matches current round
	if evt.RoundNumber != g.Round.round.RoundNumber {
		log.Errorw("stale commitment event", "game id", g.gameInfo.ID, "expected round", g.Round.round.RoundNumber, "got round", evt.RoundNumber, "player address", evt.Address.TemporaryAddress)
		return 0, fmt.Errorf("round number mismatch: expected %d, got %d", g.Round.round.RoundNumber, evt.RoundNumber)
	}

	// CommitmentIndex is 1-based (1, 2, 3), validate range
	if evt.CommitmentIndex == 0 || evt.CommitmentIndex > 3 {
		log.Errorw("invalid commitment index", "game id", g.gameInfo.ID, "round number", evt.RoundNumber, "player address", evt.Address.TemporaryAddress, "commitment index", evt.CommitmentIndex)
		return 0, fmt.Errorf("commitment index must be between 1 and 3, got %d", evt.CommitmentIndex)
	}

	// Check if the provided index matches the current turn number
	expectedIndex := g.getCurrentTurnNumber()
	if evt.CommitmentIndex != expectedIndex {
		log.Errorw("commitment index mismatch", "game id", g.gameInfo.ID, "round number", evt.RoundNumber, "player address", evt.Address.TemporaryAddress, "expected index", expectedIndex, "got index", evt.CommitmentIndex)
		return 0, fmt.Errorf("expected commitment index %d, got %d", expectedIndex, evt.CommitmentIndex)
	}

	return evt.CommitmentIndex - 1, nil // Convert to 0-based
}

// validateCardSubmission validates the card submission including round number and returns the 0-based index, card entry, card ID, and error
// Returns (cardIdx, cardEntry, cardID, error). If card is already submitted, cardEntry will be nil.
func (g *Game) validateCardSubmission(evt *types.PlayerCardOnChain) (uint32, *dao.RoundSubmittedCard, uint, error) {
	// Check round number matches current round
	if evt.RoundNumber != g.Round.round.RoundNumber {
		log.Errorw("stale card event", "game id", g.gameInfo.ID, "expected round", g.Round.round.RoundNumber, "got round", evt.RoundNumber, "player address", evt.Address.TemporaryAddress)
		return 0, nil, 0, fmt.Errorf("round number mismatch: expected %d, got %d", g.Round.round.RoundNumber, evt.RoundNumber)
	}

	if evt.Card == 0 {
		return 0, nil, 0, fmt.Errorf("card ID cannot be zero")
	}

	// CardIndex is 1-based (1, 2, 3), validate range
	if evt.CardIndex == 0 || evt.CardIndex > 3 {
		log.Errorw("invalid card index", "game id", g.gameInfo.ID, "round number", evt.RoundNumber, "player address", evt.Address.TemporaryAddress, "card index", evt.CardIndex)
		return 0, nil, 0, fmt.Errorf("card index must be between 1 and 3, got %d", evt.CardIndex)
	}

	// Check if the provided index matches the current turn number
	expectedIndex := g.getCurrentTurnNumber()
	if evt.CardIndex != expectedIndex {
		log.Errorw("card index mismatch", "game id", g.gameInfo.ID, "round number", evt.RoundNumber, "player address", evt.Address.TemporaryAddress, "expected index", expectedIndex, "got index", evt.CardIndex)
		return 0, nil, 0, fmt.Errorf("expected card index %d, got %d", expectedIndex, evt.CardIndex)
	}

	// Get player's round info using getGamePlayer
	player, err := g.getGamePlayer(evt.Address.TemporaryAddress)
	if err != nil {
		log.Errorw("player not found", "game id", g.gameInfo.ID, "round number", evt.RoundNumber, "player address", evt.Address.TemporaryAddress, "error", err)
		return 0, nil, 0, fmt.Errorf("player not found: %w", err)
	}
	playerRoundInfo := player.roundPlayer

	cardIdx := evt.CardIndex - 1 // Convert to 0-based for internal use

	// Check if card index is valid
	if int(cardIdx) >= len(playerRoundInfo.SubmittedCards) {
		log.Errorw("card index out of range", "game id", g.gameInfo.ID, "round number", evt.RoundNumber, "player address", evt.Address.TemporaryAddress, "card index", cardIdx, "submitted cards length", len(playerRoundInfo.SubmittedCards))
		return 0, nil, 0, fmt.Errorf("card index %d out of range (submitted cards: %d)", cardIdx+1, len(playerRoundInfo.SubmittedCards))
	}

	// Verify commitment exists as a safety check
	cardEntry := playerRoundInfo.SubmittedCards[cardIdx]
	if len(cardEntry.SubmittedCommitment) == 0 {
		log.Errorw("commitment not submitted for card index", "game id", g.gameInfo.ID, "round number", evt.RoundNumber, "player address", evt.Address.TemporaryAddress, "card index", cardIdx)
		return 0, nil, 0, fmt.Errorf("commitment for card index %d must be submitted before card", cardIdx+1)
	}

	// Check if card is already submitted (CardID != 0)
	if cardEntry.CardID != 0 {
		log.Errorw("card already submitted for this index", "game id", g.gameInfo.ID, "round number", evt.RoundNumber, "player address", evt.Address.TemporaryAddress, "card index", cardIdx)
		return cardIdx, nil, 0, nil // Return nil cardEntry to indicate already submitted
	}

	// Get the card ID
	cardID := evt.Card
	return cardIdx, cardEntry, cardID, nil
}
