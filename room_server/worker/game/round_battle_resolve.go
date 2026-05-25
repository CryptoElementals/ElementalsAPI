package game

import (
	"slices"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

// loserPlayerGameResultStatus maps in-round player status to persisted proto status for non-winners.
func loserPlayerGameResultStatus(status playerStatus) proto.PlayerGameResultStatus {
	switch status {
	case playerStatusOffline:
		return proto.PlayerGameResultStatus_PLAYER_OFFLINE
	case playerStatusSurrendered:
		return proto.PlayerGameResultStatus_PLAYER_SURRENDER
	default:
		return proto.PlayerGameResultStatus_PLAYER_LOSE
	}
}

// round_battle_resolve: game-over detection and timeout tie result. Economy rows are created at lobby settlement.

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
	return r.buildGameResultWithStatuses(grType, winnerPlayerId, temporaryAddress, finalMul, playerStatuses)
}

func (r *round) buildGameResultWithStatuses(grType gameResultType, winnerPlayerId int64, temporaryAddress string, finalMul uint32, playerStatuses map[string]playerStatus) *dao.GameResult {
	infos := r.buildPlayerResultInfos(grType, winnerPlayerId, temporaryAddress, playerStatuses)
	return &dao.GameResult{
		GameID:            r.game.ID,
		GameType:          proto.GameType(r.game.Type),
		Multiplier:        int32(finalMul),
		GameResultType:    proto.GameResultType(grType),
		PlayerResultInfos: infos,
	}
}

func (r *round) buildPlayerResultInfos(grType gameResultType, winnerPlayerId int64, temporaryAddress string, playerStatuses map[string]playerStatus) []*dao.PlayerResultInfo {
	var infos []*dao.PlayerResultInfo

	switch grType {
	case gameResultTie:
		infos = make([]*dao.PlayerResultInfo, 0, len(r.gamePlayers))
		for _, player := range r.gamePlayers {
			infos = append(infos, &dao.PlayerResultInfo{
				PlayerId:               player.player.PlayerId,
				TemporaryAddress:       player.player.TemporaryAddress,
				IsWinner:               false,
				PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE,
			})
		}

	case gameResultNormal, gameResultKO:
		if winnerPlayerId == 0 || temporaryAddress == "" {
			return nil
		}
		infos = make([]*dao.PlayerResultInfo, 0, len(r.gamePlayers))
		for _, player := range r.gamePlayers {
			status := playerStatuses[player.player.TemporaryAddress]
			isWinner := player.player.TemporaryAddress == temporaryAddress
			var st proto.PlayerGameResultStatus
			if isWinner {
				st = proto.PlayerGameResultStatus_PLAYER_WIN
			} else {
				st = loserPlayerGameResultStatus(status)
			}
			infos = append(infos, &dao.PlayerResultInfo{
				PlayerId:               player.player.PlayerId,
				TemporaryAddress:       player.player.TemporaryAddress,
				IsWinner:               isWinner,
				PlayerGameResultStatus: st,
			})
		}
	}

	return infos
}

func (r *round) isGameEndsByRoundAndTurn() bool {
	return r.roundNumber == r.maxConfiguredRounds() && r.getCurrentTurnNumber() == r.turnsPerRound()
}

// isGameEndsByRegulationRoundAndTurn is true on the last turn of the round (turnNumber == turnsPerRound)
// when roundNumber is at least the final regulation round (regulation finale or any OT round).
// maxConfiguredRounds is regulation+OT, so regulation end is not the same as last overall round.
func (r *round) isGameEndsByRegulationRoundAndTurn() bool {
	if r == nil || r.game == nil {
		return false
	}
	reg := dao.RegulationRoundsForPub(r.game)
	if reg < 1 {
		return false
	}
	if r.getCurrentTurnNumber() != r.turnsPerRound() {
		return false
	}
	return r.roundNumber >= reg
}

func (r *round) checkGameOver(checkRoundTurnLimit bool) (bool, *dao.GameResult) {
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
	}

	spread := r.hpSpreadMultiplier()
	if spread < 1 {
		spread = 1
	}

	surrenderedCount := 0
	for _, player := range r.gamePlayers {
		if player.status == playerStatusSurrendered {
			surrenderedCount++
		}
	}

	if surrenderedCount > 0 {
		if surrenderedCount == len(r.gamePlayers) {
			return true, r.buildGameResult(gameResultTie, 0, "", 1)
		}
		for _, player := range r.gamePlayers {
			if player.status != playerStatusSurrendered {
				return true, r.buildGameResult(gameResultKO, player.player.PlayerId, player.player.TemporaryAddress, spread)
			}
		}
	}

	offlineCount := 0
	for _, player := range r.gamePlayers {
		if player.status == playerStatusOffline {
			offlineCount++
		}
	}

	if offlineCount > 0 {
		if offlineCount == len(r.gamePlayers) {
			return true, r.buildGameResult(gameResultTie, 0, "", 1)
		}
		for _, player := range r.gamePlayers {
			if player.status == playerStatusOnline {
				return true, r.buildGameResult(gameResultNormal, player.player.PlayerId, player.player.TemporaryAddress, spread)
			}
		}
	}

	ok, grType, winnerID, winnerTemp, finalMul := r.checkGameOverByHP(false, checkRoundTurnLimit)
	if !ok {
		return false, nil
	}
	return true, r.buildGameResult(grType, winnerID, winnerTemp, finalMul)
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

func (r *round) checkGameOverByHP(hasOffline bool, checkRoundTurnLimit bool) (bool, gameResultType, int64, string, uint32) {
	players := make([]*gamePlayer, 0, len(r.gamePlayers))
	for _, player := range r.gamePlayers {
		players = append(players, player)
	}
	hps := make([]int, len(players))
	for i, player := range players {
		hps[i] = int(player.currentHP)
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
		for i, player := range players {
			if hps[i] > 0 {
				winnerPlayerId = player.player.PlayerId
				winnerTemp = player.player.TemporaryAddress
			}
		}
		finalMultiplier = rewardSpreadMultiplier(hps)
	} else {
		endedBySchedule := checkRoundTurnLimit && r.isGameEndsByRoundAndTurn()
		endedRegulationOrOTRound := checkRoundTurnLimit && r.isGameEndsByRegulationRoundAndTurn()
		if !hasOffline && !endedBySchedule && !endedRegulationOrOTRound {
			return false, gameResultNormal, 0, "", 1
		}

		gType = gameResultNormal
		maxHP := -1
		for _, hp := range hps {
			if hp > maxHP {
				maxHP = hp
			}
		}

		for i, player := range players {
			if hps[i] == maxHP {
				winnerPlayerId = player.player.PlayerId
				winnerTemp = player.player.TemporaryAddress
			}
		}
		finalMultiplier = rewardSpreadMultiplier(hps)
	}

	return true, gType, winnerPlayerId, winnerTemp, finalMultiplier
}

func (r *round) handleServerTimeout() (bool, *dao.GameResult, error) {
	infos := make([]*dao.PlayerResultInfo, 0, len(r.gamePlayers))
	for _, gamePlayer := range r.gamePlayers {
		infos = append(infos, &dao.PlayerResultInfo{
			PlayerId:               gamePlayer.player.PlayerId,
			TemporaryAddress:       gamePlayer.player.TemporaryAddress,
			IsWinner:               false,
			PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE,
		})
	}

	gameResult := &dao.GameResult{
		GameID:            r.game.ID,
		GameType:          proto.GameType(r.game.Type),
		Multiplier:        0,
		GameResultType:    proto.GameResultType_GAME_TIE,
		PlayerResultInfos: infos,
	}

	return true, gameResult, nil
}
