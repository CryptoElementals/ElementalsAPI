package game

import (
	"fmt"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

// abortedGameResult creates a game result for an aborted game
func (g *Game) abortedGameResult() *dao.GameResult {
	gameRes := &dao.GameResult{
		GameResultType: proto.GameResultType_GAME_ABORTED,
		BattleReward: &dao.BattleReward{
			PlayerRewards: []*dao.PlayerReward{},
		},
	}
	for _, player := range g.currentRound.gamePlayers {
		playerReward := &dao.PlayerReward{
			PlayerId:         player.player.PlayerId,
			TemporaryAddress: player.player.TemporaryAddress,
		}
		gameRes.BattleReward.PlayerRewards = append(gameRes.BattleReward.PlayerRewards, playerReward)
	}
	return gameRes
}

// sendGameCompletedEventAndStop sends game completed event and stops the game
// Used by abort handlers that need to trigger game result settlement
func (g *Game) sendGameCompletedEventAndStop() {
	completeEvt := &types.GameCompletedEvent{
		GameID:   g.gameInfo.ID,
		GameInfo: g.gameInfo,
	}
	// Still need to call HandleGameCompletedEvent for game result settlement
	if err := g.gameContextHandler.HandleGameCompletedEvent(completeEvt); err != nil {
		log.Errorw("handle game complete event failed", "err", err, "game id", g.gameInfo.ID)
	}
	g.stopGame()
}

// sendTurnCompletedEventForAbort sends a turn completed event to all players when the game is aborted
func (g *Game) sendTurnCompletedEventForAbort() {
	// Get round and turn numbers - use defaults if currentRound doesn't exist
	var roundNumber uint32 = 1
	var turnNumber uint32 = 1

	playerTurnInfos := make([]*types.PlayerTurnInfo, 0)
	if g.currentRound != nil {
		if g.currentRound.round != nil {
			roundNumber = uint32(g.currentRound.round.RoundNumber)
			turnNumber = g.currentRound.getCurrentTurnNumber()
		}
		// Build PlayerTurnInfo for all players
		if len(g.currentRound.gamePlayers) > 0 {
			for _, p := range g.currentRound.gamePlayers {
				// Get PlayerTurnInfo for current turn if available
				var submittedCard *dao.TurnSubmittedCard
				if turnInfo := p.getCurrentPlayerTurnInfo(); turnInfo != nil && turnInfo.TurnSubmittedCard != nil {
					submittedCard = turnInfo.TurnSubmittedCard
				}
				playerTurnInfos = append(playerTurnInfos, &types.PlayerTurnInfo{
					PlayerAddress: p.PlayerAddress(),
					SubmittedCard: submittedCard,
				})
			}
		}
	}

	// Create and send turn completed event
	turnCompletedEvt := &types.TurnCompletedEvent{
		GameID:          g.gameInfo.ID,
		RoundNumber:     roundNumber,
		TurnNumber:      turnNumber,
		IsRoundComplete: true, // Game is ending, so round is complete
		IsGameComplete:  true, // Game is being aborted, so game is complete
		PlayerTurnInfo:  playerTurnInfos,
		GameResult:      g.gameInfo.GameResult, // Include the aborted game result
	}
	g.sendEventsToAllPlayers(types.NewEvent(g.workerID(), turnCompletedEvt))
}

// handleGameAbortInit handles game abortion during initialization
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
	// Send turn completed event to all players
	g.sendTurnCompletedEventForAbort()
	g.sendGameCompletedEventAndStop()
	return nil
}

// handleGameAbortInternalError handles game abortion due to internal errors
// can go into game end from any other status
func (g *Game) handleGameAbortInternalError() error {
	log.Infow("game aborted with internal error", "game id", g.gameInfo.ID)
	if g.currentRound.round != nil {
		g.currentRound.round.IsLastRound = true
		g.currentRound.turnStatus = proto.TurnStatus_TURN_ROUND_COMPLETED
	}

	g.gameInfo.Status = proto.GameStatus_GAME_ABORTED
	g.gameInfo.GameResult = g.abortedGameResult()
	err := g.saveGame()
	if err != nil {
		return err
	}
	// Send turn completed event to all players
	g.sendTurnCompletedEventForAbort()
	g.sendGameCompletedEventAndStop()
	return nil
}
