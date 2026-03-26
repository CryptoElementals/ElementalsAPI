package game

import (
	"errors"

	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

// ---- On-chain / contract callbacks (room created, turn setup, commitments, cards observed on chain) ----

func (g *Game) handleRoomCreated(gameID uint, blockTime int64) error {
	defer g.sendTimerEventByCurrentRound()
	currentTurn := g.currentRound.getCurrentTurn()
	currentTurn.TurnStartAt = blockTime
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_COMMITMENTS)
	if err := g.saveRound(); err != nil {
		return err
	}

	players := make([]*proto.PlayerAddress, 0, len(g.currentRound.gamePlayers))
	for _, player := range g.currentRound.gamePlayers {
		addr := player.PlayerAddress()
		players = append(players, addr.ToProto())
	}
	ga := g.gameInfo.GameArgs
	g.publishProtoToAllPlayers(&proto.Event{
		Type: proto.EventType_TYPE_GAME_CREATED,
		Event: &proto.Event_GameReady{
			GameReady: &proto.GameReady{
				GameId:            uint32(g.gameInfo.ID),
				MaxRoundNum:       uint32(ga.MaxRounds),
				MaxTurnNum:        uint32(ga.MaxTurnsPerRound),
				InitialHP:         uint32(ga.InitialHP),
				InitialMultiplier: uint32(ga.InitialMultiplier),
				Players:           players,
			},
		},
	})
	g.publishProtoToAllPlayers(&proto.Event{
		Type: proto.EventType_TYPE_ROUND_READY,
		Event: &proto.Event_RoundReady{
			RoundReady: &proto.RoundReady{
				GameId:   uint32(g.gameInfo.ID),
				RoundNum: g.currentRound.roundNumber,
			},
		},
	})
	g.publishProtoToAllPlayers(&proto.Event{
		Type: proto.EventType_TYPE_TURN_READY,
		Event: &proto.Event_TurnReady{
			TurnReady: &proto.TurnReady{
				GameId:                      uint32(g.gameInfo.ID),
				RoundNum:                    g.currentRound.roundNumber,
				TurnNum:                     1,
				CommitmentSubmissionTimeout: ga.CommitmentSubmissionTimeout,
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
	if tx.RoundNumber != g.currentRound.roundNumber {
		return nil
	}
	if tx.TurnNumber != g.currentRound.getCurrentTurnNumber() {
		return nil
	}

	currentTurn := g.currentRound.getCurrentTurn()
	currentTurn.TurnStartAt = blockTime

	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_COMMITMENTS)
	if err := g.saveRound(); err != nil {
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
				CommitmentSubmissionTimeout: g.gameInfo.GameArgs.CommitmentSubmissionTimeout,
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
		err = g.saveRound()
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
					CardSubmissionTimeout: g.gameInfo.GameArgs.CardSubmissionTimeout,
				},
			},
		})
	} else {
		err = g.saveRound()
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
	return g.saveRound()
}
