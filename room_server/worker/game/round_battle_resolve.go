package game

import (
	"slices"

	"github.com/CryptoElementals/common/config"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

// round_battle_resolve: game-over detection, rewards, and timeout tie result.

func (r *round) buildGameEndStates() []*gameEndState {
	spread := r.hpSpreadMultiplier()
	playerCount := len(r.gamePlayers)
	gameEndStates := make([]*gameEndState, 0, playerCount)
	for _, player := range r.gamePlayers {
		if player.status == playerStatusSurrendered {
			gameEndStates = append(gameEndStates, &gameEndState{
				HP:               int(player.currentHP),
				Multiplier:       spread,
				PlayerId:         player.player.PlayerId,
				TemporaryAddress: player.player.TemporaryAddress,
				Status:           player.status,
			})
			continue
		}

		submittedCard := player.getLastSubmittedCard()
		hasSubmittedCommitment := submittedCard != nil && len(submittedCard.CommitmentHash) > 0
		hasSubmittedCard := submittedCard != nil && submittedCard.CardID > 0

		switch r.getTurnStatus() {
		case proto.TurnStatus_TURN_WAITTING_BATTLE_CONFIRMATION:
			if !player.isPlayerReady() {
				player.status = playerStatusOffline
			}
		case proto.TurnStatus_TURN_WAITTING_COMMITMENTS:
			if !hasSubmittedCommitment {
				player.status = playerStatusOffline
			}
		case proto.TurnStatus_TURN_WAITTING_CARDS:
			if !hasSubmittedCard {
				player.status = playerStatusOffline
			}
		}

		gameEndStates = append(gameEndStates, &gameEndState{
			HP:               int(player.currentHP),
			Multiplier:       spread,
			PlayerId:         player.player.PlayerId,
			TemporaryAddress: player.player.TemporaryAddress,
			Status:           player.status,
		})
	}
	return gameEndStates
}

// hpSpreadMultiplier is max(currentHP) − min(currentHP) over all players in this round (0 if fewer than 2 players).
func (r *round) hpSpreadMultiplier() uint32 {
	if r == nil || len(r.gamePlayers) < 2 {
		return 0
	}
	var minHP, maxHP int64
	first := true
	for _, p := range r.gamePlayers {
		h := p.currentHP
		if first {
			minHP, maxHP = h, h
			first = false
			continue
		}
		if h < minHP {
			minHP = h
		}
		if h > maxHP {
			maxHP = h
		}
	}
	return uint32(maxHP - minHP)
}

func (r *round) buildGameResult(grType gameResultType, winnerPlayerId int64, temporaryAddress string, finalMul uint32) *dao.GameResult {
	playerStatuses := make(map[string]playerStatus)
	for _, player := range r.gamePlayers {
		playerStatuses[player.player.TemporaryAddress] = player.status
	}

	battleReward := r.calculateBattleRewardFromGamePlayers(r.gamePlayers, grType, winnerPlayerId, temporaryAddress, finalMul, playerStatuses)

	return &dao.GameResult{
		GameID:                 r.game.ID,
		Multiplier:             int32(finalMul),
		WinnerPlayerId:         winnerPlayerId,
		WinnerTemporaryAddress: temporaryAddress,
		GameResultType:         proto.GameResultType(grType),
		BattleReward:           battleReward,
	}
}

func (r *round) checkGameOverFromGamePlayersPreExecution() (bool, *dao.GameResult) {
	gameEndStates := r.buildGameEndStates()
	isGameOver, grType, winnerPlayerId, temporaryAddress, finalMul := r.checkGameOver(gameEndStates, r.roundNumber, false)
	var gameResult *dao.GameResult
	if isGameOver {
		gameResult = r.buildGameResult(grType, winnerPlayerId, temporaryAddress, finalMul)
	}
	return isGameOver, gameResult
}

func (r *round) checkGameOverFromGamePlayersPostExecution() (bool, *dao.GameResult) {
	gameEndStates := r.buildGameEndStates()
	isGameOver, grType, winnerPlayerId, temporaryAddress, finalMul := r.checkGameOver(gameEndStates, r.roundNumber, true)
	var gameResult *dao.GameResult
	if isGameOver {
		gameResult = r.buildGameResult(grType, winnerPlayerId, temporaryAddress, finalMul)
	}
	return isGameOver, gameResult
}

func (r *round) isGameEndsByRoundAndTurn() bool {
	return r.roundNumber == r.maxConfiguredRounds() && r.getCurrentTurnNumber() == r.turnsPerRound()
}

func (r *round) checkGameOver(states []*gameEndState, round uint32, checkRoundTurnLimit bool) (bool, gameResultType, int64, string, uint32) {
	surrenderedCount := 0
	maxLoserMul := uint32(1)

	for _, state := range states {
		if state.Status == playerStatusSurrendered {
			surrenderedCount++
			if state.Multiplier > maxLoserMul {
				maxLoserMul = state.Multiplier
			}
		}
	}

	if surrenderedCount > 0 {
		if surrenderedCount == len(states) {
			return true, gameResultTie, 0, "", 1
		}
		for _, state := range states {
			if state.Status != playerStatusSurrendered {
				return true, gameResultKO, state.PlayerId, state.TemporaryAddress, maxLoserMul
			}
		}
	}

	offlineCount := 0
	var offlineMaxMul uint32 = 1
	for _, state := range states {
		if state.Status == playerStatusOffline {
			offlineCount++
			if state.Multiplier > offlineMaxMul {
				offlineMaxMul = state.Multiplier
			}
		}
	}

	if offlineCount > 0 {
		if offlineCount == len(states) {
			return true, gameResultTie, 0, "", 1
		}
		for _, state := range states {
			if state.Status == playerStatusOnline {
				return true, gameResultNormal, state.PlayerId, state.TemporaryAddress, offlineMaxMul
			}
		}
	}

	return r.checkGameOverByHP(states, round, false, checkRoundTurnLimit)
}

func rewardSpreadMultiplier(hps []int) uint32 {
	if len(hps) == 0 {
		return 1
	}
	minH, maxH := hps[0], hps[0]
	for _, h := range hps[1:] {
		if h < minH {
			minH = h
		}
		if h > maxH {
			maxH = h
		}
	}
	d := maxH - minH
	if d < 1 {
		return 1
	}
	return uint32(d)
}

func (r *round) checkGameOverByHP(states []*gameEndState, round uint32, hasOffline bool, checkRoundTurnLimit bool) (bool, gameResultType, int64, string, uint32) {
	hps := make([]int, len(states))
	for i, state := range states {
		hps[i] = state.HP
	}

	allSameHP := true
	firstHP := hps[0]
	for _, hp := range hps[1:] {
		if hp != firstHP {
			allSameHP = false
			break
		}
	}

	if allSameHP {
		if firstHP == 0 {
			return true, gameResultTie, 0, "", 1
		}
		if hasOffline || (checkRoundTurnLimit && r.isGameEndsByRoundAndTurn()) {
			return true, gameResultTie, 0, "", 1
		}
		return false, gameResultNormal, 0, "", 1
	}

	hasZeroHP := slices.Contains(hps, 0)

	var winnerPlayerId int64
	var winnerTemp string
	var gType gameResultType
	var finalMultiplier uint32 = 1

	if hasZeroHP {
		gType = gameResultKO
		for i, state := range states {
			if hps[i] > 0 {
				winnerPlayerId = state.PlayerId
				winnerTemp = state.TemporaryAddress
			}
		}
		finalMultiplier = rewardSpreadMultiplier(hps)
	} else {
		if !hasOffline && !(checkRoundTurnLimit && r.isGameEndsByRoundAndTurn()) {
			return false, gameResultNormal, 0, "", 1
		}

		gType = gameResultNormal
		maxHP := -1
		for _, hp := range hps {
			if hp > maxHP {
				maxHP = hp
			}
		}

		for i, state := range states {
			if hps[i] == maxHP {
				winnerPlayerId = state.PlayerId
				winnerTemp = state.TemporaryAddress
			}
		}
		finalMultiplier = rewardSpreadMultiplier(hps)
	}

	return true, gType, winnerPlayerId, winnerTemp, finalMultiplier
}

// calculateBattleRewardFromGamePlayers calculates battle rewards for all players based on game result type.
//
// Reward Calculation Logic:
// 1. Tie Game (gameResultTie):
//   - All players lose 0.8% of base stake in tokens (0.008 * baseStake)
//   - All players gain 0.8% of base stake in points (0.008 * baseStake)
//   - System fee = token deduction * number of players
//   - All players marked with PLAYER_TIE status
//
// 2. Win/Loss Game (gameResultNormal or gameResultKO):
//   - Total pool = baseStake * finalMultiplier
//   - Winner receives: (totalPool * 0.984) tokens + points (0.012 * totalPool for normal, 0.016 * totalPool for KO)
//   - Each loser loses: (totalPool / loserCount) tokens + points (0.004 * totalPool / loserCount for normal, 0 for KO)
//   - System fee = totalPool * 0.016 (1.6%)
//
// Surrender and Offline Handling:
// - For win/loss games: If a loser surrenders or goes offline, their points are transferred to the winner.
//   - Surrendered/offline losers receive 0 points (instead of their normal loser points)
//   - Winner receives bonus points equal to the sum of all surrendered/offline losers' points
//
// - For tie games: All players receive the same points regardless of status.
// - The IsOffline and Surrendered flags are set in PlayerReward for record-keeping.
// - Status is determined from playerStatuses map passed as parameter, which reflects the player's state at game end.
func (r *round) calculateBattleRewardFromGamePlayers(players map[string]*gamePlayer, grType gameResultType, winnerPlayerId int64, temporaryAddress string, finalMul uint32, playerStatuses map[string]playerStatus) *dao.BattleReward {
	baseStake := config.GameParams.BaseStake
	var playerRewards []*dao.PlayerReward
	var systemFee int

	switch grType {
	case gameResultTie:
		tokenDeduction := int(float64(baseStake) * 0.008)
		pointGain := int(float64(baseStake) * 0.008)
		playerRewards = make([]*dao.PlayerReward, 0, len(players))

		for _, player := range players {
			status := playerStatuses[player.player.TemporaryAddress]
			playerRewards = append(playerRewards, &dao.PlayerReward{
				PlayerId:               player.player.PlayerId,
				TemporaryAddress:       player.player.TemporaryAddress,
				TokenChange:            int32(-tokenDeduction),
				PointChange:            int32(pointGain),
				IsOffline:              status == playerStatusOffline,
				Surrendered:            status == playerStatusSurrendered,
				PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE,
			})
		}
		systemFee = tokenDeduction * len(players)

	case gameResultNormal, gameResultKO:
		if winnerPlayerId != 0 && temporaryAddress != "" {
			loserCount := len(players) - 1
			totalPool := int(float64(baseStake) * float64(finalMul))

			winnerTokenPerPlayer := int(float64(totalPool) * (1.0 - 0.016))
			loserTokenPerPlayer := totalPool / loserCount

			var winnerPointPerPlayer, loserPointPerPlayer int
			if grType == gameResultNormal {
				winnerPointPerPlayer = int(float64(totalPool) * 0.012)
				loserPointPerPlayer = int(float64(totalPool)*0.004) / loserCount
			} else {
				winnerPointPerPlayer = int(float64(totalPool) * 0.016)
				loserPointPerPlayer = 0
			}

			bonusPointsForWinner := 0
			for _, player := range players {
				if player.player.TemporaryAddress != temporaryAddress {
					status := playerStatuses[player.player.TemporaryAddress]
					if status == playerStatusSurrendered || status == playerStatusOffline {
						bonusPointsForWinner += loserPointPerPlayer
					}
				}
			}

			playerRewards = make([]*dao.PlayerReward, 0, len(players))
			for _, player := range players {
				status := playerStatuses[player.player.TemporaryAddress]
				isWinner := player.player.TemporaryAddress == temporaryAddress

				var tokenChange, pointChange int
				var gameResultStatus proto.PlayerGameResultStatus
				if isWinner {
					tokenChange = winnerTokenPerPlayer
					pointChange = winnerPointPerPlayer + bonusPointsForWinner
					gameResultStatus = proto.PlayerGameResultStatus_PLAYER_WIN
				} else {
					tokenChange = -loserTokenPerPlayer
					if status == playerStatusSurrendered || status == playerStatusOffline {
						pointChange = 0
					} else {
						pointChange = loserPointPerPlayer
					}
					gameResultStatus = proto.PlayerGameResultStatus_PLAYER_LOSE
				}

				playerRewards = append(playerRewards, &dao.PlayerReward{
					PlayerId:               player.player.PlayerId,
					TemporaryAddress:       player.player.TemporaryAddress,
					TokenChange:            int32(tokenChange),
					PointChange:            int32(pointChange),
					IsOffline:              status == playerStatusOffline,
					Surrendered:            status == playerStatusSurrendered,
					PlayerGameResultStatus: gameResultStatus,
				})
			}

			systemFee = int(float64(totalPool) * 0.016)
		}
	}

	return &dao.BattleReward{
		PlayerRewards: playerRewards,
		SystemFee:     int32(systemFee),
	}
}

func (r *round) handleServerTimeout() (bool, *dao.GameResult, error) {
	var playerRewards []*dao.PlayerReward

	for _, gamePlayer := range r.gamePlayers {
		submittedCard := gamePlayer.getLastSubmittedCard()
		hasCommitment := len(submittedCard.CommitmentHash) > 0
		playerRewards = append(playerRewards, &dao.PlayerReward{
			PlayerId:               gamePlayer.player.PlayerId,
			TemporaryAddress:       gamePlayer.player.TemporaryAddress,
			TokenChange:            0,
			PointChange:            0,
			IsOffline:              !hasCommitment,
			Surrendered:            gamePlayer.isSurrendered(),
			PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE,
		})
	}

	gameResult := &dao.GameResult{
		GameID:                 r.game.ID,
		Multiplier:             0,
		WinnerPlayerId:         0,
		WinnerTemporaryAddress: "",
		GameResultType:         proto.GameResultType_GAME_TIE,
		BattleReward: &dao.BattleReward{
			PlayerRewards: playerRewards,
			SystemFee:     0,
		},
	}

	return true, gameResult, nil
}
