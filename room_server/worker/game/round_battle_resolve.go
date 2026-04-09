package game

import (
	"slices"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

// round_battle_resolve: game-over detection, rewards, and timeout tie result.

func (r *round) buildGameEndStates() []*gameEndState {
	spread := r.hpSpreadMultiplier()
	gameEndStates := make([]*gameEndState, 0, len(r.gamePlayers))
	for _, player := range r.gamePlayers {
		if player.status != playerStatusSurrendered {
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

// hpSpreadMultiplier is (max HP − min HP) / dao.HPDiffPerMultiplierUnit among players (0 if fewer than 2).
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
	return dao.MultiplierFromHPSpread(maxHP - minHP)
}

func (r *round) buildGameResult(grType gameResultType, winnerPlayerId int64, temporaryAddress string, finalMul uint32) *dao.GameResult {
	playerStatuses := make(map[string]playerStatus)
	for _, player := range r.gamePlayers {
		playerStatuses[player.player.TemporaryAddress] = player.status
	}

	battleReward := r.calculateBattleRewardFromGamePlayers(r.gamePlayers, grType, winnerPlayerId, temporaryAddress, playerStatuses)

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
	return r.checkGameOverWithResult(false)
}

func (r *round) checkGameOverFromGamePlayersPostExecution() (bool, *dao.GameResult) {
	return r.checkGameOverWithResult(true)
}

func (r *round) checkGameOverWithResult(checkRoundTurnLimit bool) (bool, *dao.GameResult) {
	states := r.buildGameEndStates()
	ok, grType, winnerID, winnerTemp, finalMul := r.checkGameOver(states, r.roundNumber, checkRoundTurnLimit)
	if !ok {
		return false, nil
	}
	return true, r.buildGameResult(grType, winnerID, winnerTemp, finalMul)
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
	return dao.MultiplierFromHPSpread(int64(d))
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

// buildBattleRewardMetadata persists per-player outcome flags for lobby settlement.
// TokenChange, PointChange, and SystemFee are computed in the lobby (battlereward.ComputeBattleRewardAmounts) and written in the same transaction as wallet settlement.
func (r *round) calculateBattleRewardFromGamePlayers(players map[string]*gamePlayer, grType gameResultType, winnerPlayerId int64, temporaryAddress string, playerStatuses map[string]playerStatus) *dao.BattleReward {
	var playerRewards []*dao.PlayerReward

	switch grType {
	case gameResultTie:
		playerRewards = make([]*dao.PlayerReward, 0, len(players))
		for _, player := range players {
			status := playerStatuses[player.player.TemporaryAddress]
			playerRewards = append(playerRewards, &dao.PlayerReward{
				PlayerId:               player.player.PlayerId,
				TemporaryAddress:       player.player.TemporaryAddress,
				IsOffline:              status == playerStatusOffline,
				Surrendered:            status == playerStatusSurrendered,
				PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE,
			})
		}

	case gameResultNormal, gameResultKO:
		if winnerPlayerId != 0 && temporaryAddress != "" {
			playerRewards = make([]*dao.PlayerReward, 0, len(players))
			for _, player := range players {
				status := playerStatuses[player.player.TemporaryAddress]
				isWinner := player.player.TemporaryAddress == temporaryAddress
				var gameResultStatus proto.PlayerGameResultStatus
				if isWinner {
					gameResultStatus = proto.PlayerGameResultStatus_PLAYER_WIN
				} else {
					gameResultStatus = proto.PlayerGameResultStatus_PLAYER_LOSE
				}
				playerRewards = append(playerRewards, &dao.PlayerReward{
					PlayerId:               player.player.PlayerId,
					TemporaryAddress:       player.player.TemporaryAddress,
					IsOffline:              status == playerStatusOffline,
					Surrendered:            status == playerStatusSurrendered,
					PlayerGameResultStatus: gameResultStatus,
				})
			}
		}
	}

	return &dao.BattleReward{
		PlayerRewards: playerRewards,
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
