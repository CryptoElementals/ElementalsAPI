package game

import (
	"fmt"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
)

// PlayerStatus represents a player's status
type playerStatus int32

const (
	playerStatusOnline playerStatus = iota
	playerStatusOffline
	playerStatusSurrendered
)

type gamePlayer struct {
	player          *dao.GamePlayerInfo
	currentTurnInfo *dao.PlayerTurnInfo
	currentHP       int64
	status          playerStatus // Runtime player status during battle
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

// getCurrentPlayerTurnInfo returns PlayerTurnInfo for the active turn.
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

// round is runtime state for the current logical round / turn (backed by dao.Game.Turns).
type round struct {
	game           *dao.Game
	roundNumber    uint32
	turnNumber     uint32
	gamePlayers    map[string]*gamePlayer
	completeReason proto.RoundCompleteReason
	isLastRound    bool
}

func (r *round) maxConfiguredRounds() uint32 {
	if r.game == nil {
		return 0
	}
	return dao.EffectiveMaxRounds(r.game)
}

func (r *round) turnsPerRound() uint32 {
	if r == nil || r.game == nil {
		return 0
	}
	return dao.TurnsPerRoundForGame(r.game, r.roundNumber)
}

// getCurrentTurnNumber returns the current turn index (1..turnsPerRound) within this round.
func (r *round) getCurrentTurnNumber() uint32 {
	return r.turnNumber
}

// getCurrentTurn gets the persisted Turn row for the current round and in-round turn.
func (r *round) getCurrentTurn() *dao.Turn {
	if r.game == nil {
		return nil
	}
	for _, t := range r.game.Turns {
		if t != nil && t.RoundNumber == r.roundNumber && t.TurnNumber == r.turnNumber {
			return t
		}
	}
	return nil
}

// getTurnStatus returns the current turn's status from the underlying Turn record.
func (r *round) getTurnStatus() proto.TurnStatus {
	if r == nil {
		return 0
	}
	currentTurn := r.getCurrentTurn()
	if currentTurn == nil {
		return 0
	}
	return proto.TurnStatus(currentTurn.TurnStatus)
}

// setTurnStatus updates the current turn's status on the underlying Turn record.
func (r *round) setTurnStatus(status proto.TurnStatus) {
	if r == nil {
		return
	}
	currentTurn := r.getCurrentTurn()
	if currentTurn == nil {
		return
	}
	currentTurn.TurnStatus = uint32(status)
}

// createNewTurn creates a new Turn row on the game and wires PlayerTurnInfos + runtime pointers.
func (r *round) createNewTurn() *dao.Turn {
	if r.game == nil {
		return nil
	}
	newTurn := &dao.Turn{
		GameID:          r.game.ID,
		RoundNumber:     r.roundNumber,
		TurnNumber:      r.turnNumber,
		TurnStatus:      uint32(proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION),
		PlayerTurnInfos: make([]*dao.PlayerTurnInfo, 0),
	}

	for _, player := range r.gamePlayers {
		newTurnInfo := &dao.PlayerTurnInfo{
			PlayerID:         player.player.PlayerId,
			TemporaryAddress: player.player.TemporaryAddress,
			PlayerStatus:     proto.PlayerTurnStatus_PLAYER_TURN_UNKNOWN,
			TurnSubmittedCard: &dao.TurnSubmittedCard{
				// Snapshot the player's current runtime stats at the start of this turn.
				HealthBefore: uint32(player.currentHP),
			},
		}
		player.currentTurnInfo = newTurnInfo
		newTurn.PlayerTurnInfos = append(newTurn.PlayerTurnInfos, newTurnInfo)
	}

	r.game.Turns = append(r.game.Turns, newTurn)
	return newTurn
}
