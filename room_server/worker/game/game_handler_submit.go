package game

import (
	"bytes"
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
	player, err := g.getGamePlayer(reqEvt.Address.TemporaryAddress)
	if err != nil {
		return fmt.Errorf("failed to get player: %w", err)
	}
	pti := player.getCurrentPlayerTurnInfo()
	if pti == nil || pti.TurnSubmittedCard == nil {
		return fmt.Errorf("player turn info or submitted card not found")
	}
	if pti.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_ON_CHAIN {
		return fmt.Errorf("commitment already confirmed on chain")
	}
	if len(pti.TurnSubmittedCard.CommitmentHash) > 0 {
		if !bytes.Equal(pti.TurnSubmittedCard.CommitmentHash, reqEvt.Commitment) {
			return fmt.Errorf("commitment already submitted with a different hash")
		}
		if pti.PlayerStatus != proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_SUBMITTED {
			pti.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_SUBMITTED
			if err := g.persistPlayerTurnInfo(pti); err != nil {
				return err
			}
		}
	} else {
		pti.TurnSubmittedCard.CommitmentHash = append([]byte(nil), reqEvt.Commitment...)
		pti.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_SUBMITTED
		if err := g.persistPlayerTurnInfo(pti); err != nil {
			return err
		}
	}
	req := reqEvt
	return g.afterTx(func() error {
		return g.txPoolEnqueuer.AddCommitment(req)
	})
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
	cardID := uint32(reqEvt.Card)
	if playerTurnInfo.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_CARD_ON_CHAIN {
		return fmt.Errorf("card already confirmed on chain")
	}
	if playerTurnInfo.TurnSubmittedCard.CardID != 0 {
		if playerTurnInfo.TurnSubmittedCard.CardID != cardID || !bytes.Equal(playerTurnInfo.TurnSubmittedCard.Salt, reqEvt.Salt) {
			return fmt.Errorf("card already submitted with a different reveal")
		}
	} else {
		playerTurnInfo.TurnSubmittedCard.CardID = cardID
		playerTurnInfo.TurnSubmittedCard.Salt = append([]byte(nil), reqEvt.Salt...)
		playerTurnInfo.PlayerStatus = proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED
		if err := g.persistPlayerTurnInfo(playerTurnInfo); err != nil {
			return err
		}
	}
	req := reqEvt
	return g.afterTx(func() error {
		return g.txPoolEnqueuer.AddCard(req)
	})
}
