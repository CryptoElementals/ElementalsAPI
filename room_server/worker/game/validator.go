package game

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

// validateCommitmentSubmission validates the commitment submission including round number and returns the 0-based index.
// Proto TurnNumber is the commitment index (1-based).
func (g *Game) validateCommitmentSubmission(tx *proto.TxCommitmentOnChain) (uint32, error) {
	var address types.PlayerAddress
	address.FromProto(tx.Address)
	if err := g.validateRoundAndIndex(tx.RoundNumber, tx.TurnNumber, address.TemporaryAddress, "commitment"); err != nil {
		return 0, err
	}
	return tx.TurnNumber - 1, nil
}

// validateCardSubmission validates the card submission including round number and returns the 0-based index, card entry, card ID, and error.
// Proto TurnNumber is the card index (1-based). When the reveal was already stored (e.g. via SubmitPlayerCard RPC), the on-chain tx must match.
func (g *Game) validateCardSubmission(tx *proto.TxCardOnChain) (uint32, *dao.TurnSubmittedCard, uint, error) {
	var address types.PlayerAddress
	address.FromProto(tx.Address)
	if err := g.validateRoundAndIndex(tx.RoundNumber, tx.TurnNumber, address.TemporaryAddress, "card"); err != nil {
		return 0, nil, 0, err
	}

	if tx.CardId == 0 {
		return 0, nil, 0, fmt.Errorf("card ID cannot be zero")
	}

	cardEntry, err := g.getAndValidateCardEntry(address.TemporaryAddress, tx.TurnNumber)
	if err != nil {
		return 0, nil, 0, err
	}

	if cardEntry.CardID != 0 {
		if cardEntry.CardID != uint32(tx.CardId) {
			return 0, nil, 0, fmt.Errorf("on-chain card id does not match stored reveal")
		}
		if !bytes.Equal(cardEntry.Salt, tx.Salt) {
			return 0, nil, 0, fmt.Errorf("on-chain salt does not match stored reveal")
		}
		cardIdx := tx.TurnNumber - 1
		return cardIdx, cardEntry, uint(tx.CardId), nil
	}

	return tx.TurnNumber - 1, cardEntry, uint(tx.CardId), nil
}

// validateRoundAndIndex validates round number and index (1-based, 1-3) against current round and turn
func (g *Game) validateRoundAndIndex(roundNumber, index uint32, playerAddr string, indexType string) error {
	if roundNumber != g.currentRound.roundNumber {
		log.Errorw("stale event", "type", indexType, "game id", g.gameInfo.ID, "expected round", g.currentRound.roundNumber, "got round", roundNumber, "player address", playerAddr)
		return fmt.Errorf("round number mismatch: expected %d, got %d", g.currentRound.roundNumber, roundNumber)
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

// validatePlayerCommitment validates the commitment using similar logic as validateCommitmentSubmission.
// Proto TurnNumber is the commitment index (1–3).
func (g *Game) validatePlayerCommitment(req *proto.SubmitPlayerCommitmentRequest) error {
	if req.Address == nil {
		return fmt.Errorf("missing address")
	}
	if err := g.validateRoundAndIndex(req.RoundNumber, req.TurnNumber, req.Address.TemporaryAddress, "commitment"); err != nil {
		return err
	}

	// check if the turn is waiting for commitments
	if g.currentRound.getTurnStatus() != proto.TurnStatus_TURN_WAITTING_COMMITMENTS {
		log.Errorw("turn is not waiting for commitments", "game id", g.gameInfo.ID, "player address", req.Address.TemporaryAddress, "turn status", g.currentRound.getTurnStatus())
		return fmt.Errorf("turn is not waiting for commitments")
	}

	if len(req.Commitment) == 0 {
		return fmt.Errorf("commitment cannot be empty")
	}

	return nil
}

// validatePlayerCard validates the card using similar logic as validateCardSubmission.
// Proto TurnNumber is the card index (1–3).
func (g *Game) validatePlayerCard(req *proto.SubmitPlayerCardRequest) error {
	if req.Address == nil {
		return fmt.Errorf("missing address")
	}
	if err := g.validateRoundAndIndex(req.RoundNumber, req.TurnNumber, req.Address.TemporaryAddress, "card"); err != nil {
		return err
	}

	// check if the turn is waiting for card
	if g.currentRound.getTurnStatus() != proto.TurnStatus_TURN_WAITTING_CARDS {
		log.Errorw("turn is not waiting for cards", "game id", g.gameInfo.ID, "player address", req.Address.TemporaryAddress, "turn status", g.currentRound.getTurnStatus())
		return fmt.Errorf("turn is not waiting for cards")
	}

	if req.Card == 0 {
		return fmt.Errorf("card ID cannot be zero")
	}

	if len(req.Salt) == 0 {
		return fmt.Errorf("salt cannot be empty")
	}

	return nil
}

func validateSubmitPlayerCommitmentRequest(req *proto.SubmitPlayerCommitmentRequest) error {
	if req == nil {
		return errors.New("nil SubmitPlayerCommitmentRequest")
	}
	if req.Address == nil {
		return errors.New("missing address")
	}
	return nil
}

func validateSubmitPlayerCardRequest(req *proto.SubmitPlayerCardRequest) error {
	if req == nil {
		return errors.New("nil SubmitPlayerCardRequest")
	}
	if req.Address == nil {
		return errors.New("missing address")
	}
	return nil
}
