package game

import (
	"fmt"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

type gamePlayer struct {
	player          *dao.GamePlayerInfo
	currentTurnInfo *dao.PlayerTurnInfo
	totalLostHP     int64
	currentHP       int64
	// Battle state fields (used during battle execution)
	multiplier uint32       // Calculated from totalLostHP
	status     playerStatus // Runtime player status during battle
}

func (p *gamePlayer) PlayerAddress() types.PlayerAddress {
	addr := types.PlayerAddress{}
	addr.FromDao(*p.player)
	return addr
}

func (p *gamePlayer) String() string {
	return fmt.Sprintf("%d_%s", p.player.PlayerId, p.player.TemporaryAddress)
}

func (g *gamePlayer) getLastSubmittedCard() *dao.TurnSubmittedCard {
	return g.currentTurnInfo.TurnSubmittedCard
}

// getCurrentPlayerTurnInfo returns PlayerTurnInfo for a specific turn number
// Note: This assumes playerTurnInfos are ordered by turn number, which may not always be true
// A better approach would be to match by TurnID, but we need access to the Turn records
func (p *gamePlayer) getCurrentPlayerTurnInfo() *dao.PlayerTurnInfo {
	return p.currentTurnInfo
}

// isPlayerReady checks if player is ready for current turn
func (p *gamePlayer) isPlayerReady() bool {
	// Check the latest turn info's status
	if p.currentTurnInfo == nil {
		return false
	}
	return p.currentTurnInfo.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_READY ||
		p.currentTurnInfo.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_COMMITMENT_SUBMITTED ||
		p.currentTurnInfo.PlayerStatus == proto.PlayerTurnStatus_PLAYER_TURN_CARD_SUBMITTED
}

// isSurrendered checks if player has surrendered
func (p *gamePlayer) isSurrendered() bool {
	// Check if any turn info indicates surrender
	return p.status == playerStatusSurrendered
}

// round represents a game round with its players
type round struct {
	round       *dao.Round
	gamePlayers map[string]*gamePlayer // Also used for battle state during battle execution
	turnNumber  uint32                 // Current turn number within this round (1-3), runtime state only
	turnStatus  proto.TurnStatus       // Runtime turn status (not persisted in Round model)
}

// getCurrentTurnNumber returns the current turn number (1-3) for this round
func (r *round) getCurrentTurnNumber() uint32 {
	return r.turnNumber
}

// getCurrentTurn gets the current Turn record
func (r *round) getCurrentTurn() *dao.Turn {
	turnNumber := r.getCurrentTurnNumber()
	idx := int(turnNumber) - 1
	return r.round.Turns[idx]
}

// createNewTurn creates a new Turn record for the current turn number
// Uses index-based access: turnNumber 1 -> index 0, turnNumber 2 -> index 1, turnNumber 3 -> index 2
// Also initializes playerTurnInfos for all players to ensure the slice is large enough for the new turn
func (r *round) createNewTurn() *dao.Turn {
	turnNumber := r.getCurrentTurnNumber()
	r.turnStatus = proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION
	// Create new turn
	newTurn := &dao.Turn{
		TurnNumber:      turnNumber,
		PlayerTurnInfos: make([]*dao.PlayerTurnInfo, 0),
	}

	for _, player := range r.gamePlayers {
		newTurnInfo := &dao.PlayerTurnInfo{
			PlayerID:          player.player.PlayerId,
			TemporaryAddress:  player.player.TemporaryAddress,
			PlayerStatus:      proto.PlayerTurnStatus_PLAYER_TURN_UNKNOWN,
			TurnSubmittedCard: &dao.TurnSubmittedCard{},
		}
		player.currentTurnInfo = newTurnInfo
		newTurn.PlayerTurnInfos = append(newTurn.PlayerTurnInfos, newTurnInfo)
	}

	r.round.Turns = append(r.round.Turns, newTurn)
	return newTurn
}
