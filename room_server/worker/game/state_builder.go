package game

import (
	"strings"

	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
)

// buildRuntimeState constructs a round runtime view (currentRound) from a fully-loaded dao.Game.
// It does NOT modify or save the DB; callers are responsible for persisting changes.
func buildRuntimeState(gameInfo *dao.Game) *round {
	gamePlayers := make(map[string]*gamePlayer)
	ga := gameInfo.GameArgs
	initialHP := ga.InitialHP
	for _, playerInfo := range gameInfo.Players {
		if playerInfo == nil {
			continue
		}
		gamePlayers[strings.ToLower(playerInfo.TemporaryAddress)] = &gamePlayer{
			player:    playerInfo,
			currentHP: initialHP,
		}
	}

	runtimeRound := &round{
		game:        gameInfo,
		roundNumber: 1,
		turnNumber:  1,
		gamePlayers: gamePlayers,
	}

	if len(gameInfo.Turns) == 0 {
		return runtimeRound
	}

	var bestR, bestT uint32
	var currentTurn *dao.Turn
	for _, t := range gameInfo.Turns {
		if t == nil {
			continue
		}
		if t.RoundNumber > bestR || (t.RoundNumber == bestR && t.TurnNumber > bestT) {
			bestR, bestT = t.RoundNumber, t.TurnNumber
			currentTurn = t
		}
	}
	if currentTurn == nil {
		return runtimeRound
	}

	runtimeRound.roundNumber = bestR
	runtimeRound.turnNumber = bestT

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
			if pti.TurnSubmittedCard != nil && pti.TurnSubmittedCard.HealthBefore > 0 {
				player.currentHP = int64(pti.TurnSubmittedCard.HealthBefore)
			}
		}
	}

	return runtimeRound
}
