package game

import (
	"fmt"

	"github.com/CryptoElementals/common/conversion"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

// abortedGameResult creates a game result for an aborted game (economy rows are not created in room).
func (g *Game) abortedGameResult() *dao.GameResult {
	gameRes := &dao.GameResult{
		GameType:          proto.GameType(g.gameInfo.Type),
		GameResultType:    proto.GameResultType_GAME_ABORTED,
		PlayerResultInfos: []*dao.PlayerResultInfo{},
	}
	for _, player := range g.currentRound.gamePlayers {
		gameRes.PlayerResultInfos = append(gameRes.PlayerResultInfos, &dao.PlayerResultInfo{
			PlayerId:               player.player.PlayerId,
			TemporaryAddress:       player.player.TemporaryAddress,
			PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_ABORTED,
		})
	}
	return gameRes
}

// runAfterAbortPersisted runs PubSub + settlement after game rows are committed.
func (g *Game) runAfterAbortPersisted() error {
	return g.afterTx(func() error {
		g.sendTurnCompletedEventForAbort()
		completeEvt := &types.GameCompletedEvent{GameID: g.gameInfo.ID, GameType: g.gameInfo.Type}
		if err := g.completeGameAndNotify(completeEvt); err != nil {
			log.Errorw("handle game complete event failed", "err", err, "game id", g.gameInfo.ID)
			return err
		}
		g.stopGame()
		return nil
	})
}

// sendTurnCompletedEventForAbort sends a turn completed event to all players when the game is aborted
func (g *Game) sendTurnCompletedEventForAbort() {
	var roundNumber uint32 = 1
	var turnNumber uint32 = 1

	playerTurnInfos := make([]*proto.PlayerTurnInfo, 0)
	if g.currentRound != nil {
		roundNumber = g.currentRound.roundNumber
		turnNumber = g.currentRound.getCurrentTurnNumber()
		if len(g.currentRound.gamePlayers) > 0 {
			for _, p := range g.currentRound.gamePlayers {
				var submittedCard *dao.TurnSubmittedCard
				if turnInfo := p.getCurrentPlayerTurnInfo(); turnInfo != nil && turnInfo.TurnSubmittedCard != nil {
					submittedCard = turnInfo.TurnSubmittedCard
				}
				addr := p.PlayerAddress()
				pti := &proto.PlayerTurnInfo{
					PlayerAddress: addr.ToProto(),
				}
				if submittedCard != nil {
					pti.SubmittedCard = conversion.TurnSubmittedCardToProtoRoundSubmittedCard(submittedCard, turnNumber)
				}
				if st := conversion.PlayerGameResultStatusPtrFromGameResult(g.gameInfo.GameResult, p.player.PlayerId); st != nil {
					pti.PlayerGameResultStatus = st
				}
				playerTurnInfos = append(playerTurnInfos, pti)
			}
		}
	}

	turnCompleted := &proto.TurnCompleted{
		GameId:          g.gameInfo.ID,
		RoundNum:        roundNumber,
		TurnNum:         turnNumber,
		IsRoundComplete: true,
		IsGameComplete:  true,
		PlayerTurnInfos: playerTurnInfos,
	}
	if g.gameInfo.GameResult != nil {
		turnCompleted.GameResult = conversion.DbGameResultToProtoGameResult(g.gameInfo.GameResult)
	}

	g.publishProtoToAllPlayers(&proto.Event{
		Type: proto.EventType_TYPE_TURN_COMPLETE,
		Event: &proto.Event_TurnCompleted{
			TurnCompleted: turnCompleted,
		},
	})
}

// handleGameAbortInit handles game abortion during initialization
// can go into game end from any other status
func (g *Game) handleGameAbortInit() error {
	log.Infow("game aborted", "game id", g.gameInfo.ID)
	if g.gameInfo.Status != proto.GameStatus_GAME_INIT {
		return fmt.Errorf("invalid game status: %d", g.gameInfo.Status)
	}
	g.currentRound.isLastRound = true
	g.gameInfo.Status = proto.GameStatus_GAME_END
	g.gameInfo.GameResult = g.abortedGameResult()
	if err := g.persistAbortInit(); err != nil {
		return err
	}
	return g.runAfterAbortPersisted()
}

// handleGameAbortInternalError handles game abortion due to internal errors
// can go into game end from any other status
func (g *Game) handleGameAbortInternalError() error {
	log.Infow("game aborted with internal error", "game id", g.gameInfo.ID)
	g.currentRound.isLastRound = true
	g.currentRound.setTurnStatus(proto.TurnStatus_TURN_ROUND_COMPLETED)

	g.gameInfo.Status = proto.GameStatus_GAME_END
	g.gameInfo.GameResult = g.abortedGameResult()
	if err := g.persistAbortInternal(); err != nil {
		return err
	}
	return g.runAfterAbortPersisted()
}
