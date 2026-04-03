package game

import (
	"errors"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

// ---- On-chain / contract callbacks (room created, turn setup, commitments, cards observed on chain) ----

func (g *Game) handleRoomCreated(blockTime int64) error {
	currentTurn := g.currentRound.getCurrentTurn()
	currentTurn.TurnStartAt = blockTime
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_WAITTING_COMMITMENTS)
	if err := g.persistCurrentTurn(); err != nil {
		return err
	}

	players := make([]*proto.PlayerAddress, 0, len(g.currentRound.gamePlayers))
	for _, player := range g.currentRound.gamePlayers {
		addr := player.PlayerAddress()
		players = append(players, addr.ToProto())
	}
	ga := g.gameInfo.GameArgs
	gi := g.gameInfo

	return g.afterTx(func() error {
		g.publishProtoToAllPlayers(&proto.Event{
			Type: proto.EventType_TYPE_GAME_CREATED,
			Event: &proto.Event_GameReady{
				GameReady: &proto.GameReady{
					GameId:                  uint32(gi.ID),
					InitialHP:               uint32(ga.InitialHP),
					Players:                 players,
					MatchId:                 gi.QueueMatchID,
					RegulationRounds:        dao.RegulationRoundsForPub(gi),
					OvertimeRounds:          dao.ExtraRoundsForPub(gi),
					RegulationTurnsPerRound: dao.RegulationTurnsPerRoundForPub(gi),
					OvertimeTurnsPerRound:   dao.OvertimeTurnsPerRoundForPub(gi),
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
		g.sendTimerEventByCurrentRound()
		return nil
	})
}

func (g *Game) handleNewTurnSetupOnChain(gameID uint, blockTime int64, tx *proto.TxGameTurnSetupReady) error {
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
	if err := g.persistCurrentTurn(); err != nil {
		return err
	}

	roundNum := tx.RoundNumber
	turnNum := tx.TurnNumber
	commitTimeout := g.gameInfo.GameArgs.CommitmentSubmissionTimeout

	return g.afterTx(func() error {
		if turnNum == 1 {
			g.publishProtoToAllPlayers(&proto.Event{
				Type: proto.EventType_TYPE_ROUND_READY,
				Event: &proto.Event_RoundReady{
					RoundReady: &proto.RoundReady{
						GameId:   uint32(g.gameInfo.ID),
						RoundNum: roundNum,
					},
				},
			})
		}
		g.publishProtoToAllPlayers(&proto.Event{
			Type: proto.EventType_TYPE_TURN_READY,
			Event: &proto.Event_TurnReady{
				TurnReady: &proto.TurnReady{
					GameId:                      uint32(g.gameInfo.ID),
					RoundNum:                    roundNum,
					TurnNum:                     turnNum,
					CommitmentSubmissionTimeout: commitTimeout,
				},
			},
		})
		g.sendTimerEventByCurrentRound()
		return nil
	})
}

func (g *Game) handleGameStateWaittingCommitments(tx *proto.TxCommitmentOnChain) error {
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
		err = g.persistCommitmentStep(playerTurnInfo, true)
		if err != nil {
			return err
		}
		roundNum := tx.RoundNumber
		cardTimeout := g.gameInfo.GameArgs.CardSubmissionTimeout
		return g.afterTx(func() error {
			g.publishProtoToAllPlayers(&proto.Event{
				Type: proto.EventType_TYPE_COMMITMENTS_ON_CHAIN,
				Event: &proto.Event_CommitmentsOnChain{
					CommitmentsOnChain: &proto.CommitmentsOnChain{
						GameId:                uint32(g.gameInfo.ID),
						RoundNum:              roundNum,
						TurnNum:               turnNumber,
						CardSubmissionTimeout: cardTimeout,
					},
				},
			})
			g.sendTimerEventByCurrentRound()
			return nil
		})
	}
	err = g.persistCommitmentStep(playerTurnInfo, false)
	if err != nil {
		return err
	}
	return g.afterTx(func() error {
		g.sendTimerEventByCurrentRound()
		return nil
	})
}

func (g *Game) handleGameStateCardSubmitted(tx *proto.TxCardOnChain) error {
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
	return g.persistPlayerTurnInfo(playerTurnInfo)
}
