package game

import (
	"strings"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
)

// buildRuntimeState constructs a round runtime view (currentRound) from a fully-loaded dao.Game.
// It does NOT modify or save the DB; callers are responsible for persisting changes.
func buildRuntimeState(gameInfo *dao.Game) *round {
	// Initialize gamePlayers map from gameInfo.Players
	gamePlayers := make(map[string]*gamePlayer)
	for _, playerInfo := range gameInfo.Players {
		if playerInfo == nil {
			continue
		}
		gamePlayers[strings.ToLower(playerInfo.TemporaryAddress)] = &gamePlayer{
			player:     playerInfo,
			currentHP:  gameInfo.InitialHP,
			multiplier: uint32(gameInfo.InitialMultiplier),
		}
	}

	runtimeRound := &round{
		round:       nil,
		gamePlayers: gamePlayers,
		turnNumber:  1,
	}

	// No rounds yet – start at round 1, turn 1, waiting for confirmation
	if len(gameInfo.Rounds) == 0 {
		return runtimeRound
	}

	// Pick current round by RoundNumber
	var currentRound *dao.Round
	var maxRoundNum uint32
	for _, r := range gameInfo.Rounds {
		if r != nil && r.RoundNumber > maxRoundNum {
			maxRoundNum = r.RoundNumber
			currentRound = r
		}
	}
	if currentRound == nil {
		return runtimeRound
	}
	runtimeRound.round = currentRound

	// Determine current turn: highest TurnNumber in current round (if any)
	var currentTurn *dao.Turn
	var maxTurnNum uint32
	for _, t := range currentRound.Turns {
		if t != nil && t.TurnNumber > maxTurnNum {
			maxTurnNum = t.TurnNumber
			currentTurn = t
		}
	}
	if maxTurnNum > 0 {
		runtimeRound.turnNumber = maxTurnNum
	} else {
		runtimeRound.turnNumber = 1
	}

	// Restore each player's currentTurnInfo from the current turn (if any)
	if currentTurn != nil {
		for _, pti := range currentTurn.PlayerTurnInfos {
			if pti == nil {
				continue
			}
			key := strings.ToLower(pti.TemporaryAddress)
			player, ok := runtimeRound.gamePlayers[key]
			if !ok {
				player, ok = runtimeRound.gamePlayers[pti.TemporaryAddress]
			}
			if !ok || player == nil {
				log.Errorf("buildRuntimeState: player %s not found in gamePlayers for game %d", pti.TemporaryAddress, gameInfo.ID)
				continue
			}
			player.currentTurnInfo = pti
			// Restore current stats from the current turn snapshot (start-of-turn).
			// If not available yet, keep initial defaults from GameArgs.
			if pti.TurnSubmittedCard != nil && pti.TurnSubmittedCard.HealthBefore > 0 {
				player.currentHP = int64(pti.TurnSubmittedCard.HealthBefore)
				if pti.TurnSubmittedCard.MultiplierBefore > 0 {
					player.multiplier = pti.TurnSubmittedCard.MultiplierBefore
				}
			}
		}
	}

	return runtimeRound
}
