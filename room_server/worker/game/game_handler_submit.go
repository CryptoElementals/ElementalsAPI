package game

import (
	"fmt"

	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/utils"
	"github.com/ethereum/go-ethereum/common"
)

// ---- RPC ingress: signed commitment / card submissions (enqueued to tx pool after validation) ----

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

func (g *Game) handleSubmitPlayerCard(reqEvt *proto.SubmitPlayerCardRequest) error {
	if err := g.validatePlayerCard(reqEvt); err != nil {
		return err
	}
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

	calculatedCommitment, err := utils.SolidityPackedKeccak256(
		[]any{
			uint(reqEvt.Card),
			reqEvt.Salt,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to calculate commitment: %w", err)
	}

	storedCommitmentHash := common.BytesToHash(storedCommitment)
	if storedCommitmentHash != calculatedCommitment {
		return fmt.Errorf("commitment verification failed: stored commitment does not match calculated commitment from card and salt")
	}

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
