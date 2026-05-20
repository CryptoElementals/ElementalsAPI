package battlereward

import (
	"cmp"
	"slices"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

// ComputeBattleRewardAmounts fills TokenChange, PointChange on each PlayerReward and SystemFee on BattleRewardPVP
// using GameResult.PlayerResultInfos for winner and forfeit (offline/surrender). Room persists zero amounts until settlement.
func ComputeBattleRewardAmounts(gr *dao.GameResult, br *dao.BattleRewardPVP, ga *dao.GameArgs) {
	if gr == nil || br == nil || ga == nil || len(br.PlayerRewards) == 0 {
		return
	}
	prs := br.PlayerRewards

	slices.SortFunc(prs, func(a, b *dao.PlayerReward) int {
		return cmp.Compare(a.PlayerId, b.PlayerId)
	})

	// Server timeout tie: Multiplier 0, no token/point movement (matches former room_server handleServerTimeout).
	if gr.GameResultType == proto.GameResultType_GAME_TIE && gr.Multiplier == 0 {
		br.SystemFee = 0
		for _, pr := range prs {
			pr.TokenChange = 0
			pr.PointChange = 0
		}
		return
	}

	switch gr.GameResultType {
	case proto.GameResultType_GAME_TIE:
		tokenDeduction := applyBps(ga.BaseStake, ga.TieTokenRateBps)
		pointGain := applyBps(ga.BaseStake, ga.TiePointRateBps)
		br.SystemFee = int32(tokenDeduction * len(prs))
		for _, pr := range prs {
			pr.TokenChange = int32(-tokenDeduction)
			pr.PointChange = int32(pointGain)
		}

	case proto.GameResultType_GAME_NORMAL, proto.GameResultType_GAME_KO:
		winnerPID, _, wok := winnerFromPlayerResultInfos(gr.PlayerResultInfos)
		if !wok || winnerPID == 0 {
			return
		}
		loserCount := len(prs) - 1
		if loserCount < 1 {
			return
		}
		poolMul := uint32(gr.Multiplier)
		if poolMul < 1 {
			poolMul = 1
		}
		totalPool := int(ga.BaseStake * int64(poolMul))

		systemFee := applyBps(int64(totalPool), ga.SystemFeeRateBps)
		winnerTokenPerPlayer := totalPool - systemFee
		loserTokenPerPlayer := totalPool / loserCount

		var winnerPointPerPlayer, loserPointPerPlayer int
		if gr.GameResultType == proto.GameResultType_GAME_NORMAL {
			winnerPointPerPlayer = applyBps(int64(totalPool), ga.NormalWinnerPointRateBps)
			loserPointPerPlayer = applyBps(int64(totalPool), ga.NormalLoserPointRateBps) / loserCount
		} else {
			winnerPointPerPlayer = applyBps(int64(totalPool), ga.KOWinnerPointRateBps)
			loserPointPerPlayer = 0
		}

		bonusPointsForWinner := 0
		for _, pr := range prs {
			if pr.PlayerId == winnerPID {
				continue
			}
			if playerForfeited(gr, pr.PlayerId) {
				bonusPointsForWinner += loserPointPerPlayer
			}
		}

		br.SystemFee = int32(systemFee)

		for _, pr := range prs {
			if pr.PlayerId == winnerPID {
				pr.TokenChange = int32(winnerTokenPerPlayer)
				pr.PointChange = int32(winnerPointPerPlayer + bonusPointsForWinner)
			} else {
				pr.TokenChange = int32(-loserTokenPerPlayer)
				if playerForfeited(gr, pr.PlayerId) {
					pr.PointChange = 0
				} else {
					pr.PointChange = int32(loserPointPerPlayer)
				}
			}
		}
	}
}

func applyBps(amount int64, bps int64) int {
	if amount <= 0 || bps <= 0 {
		return 0
	}
	return int((amount * bps) / 10000)
}

func winnerFromPlayerResultInfos(infos []*dao.PlayerResultInfo) (playerId int64, temp string, ok bool) {
	for _, p := range infos {
		if p == nil {
			continue
		}
		if p.IsWinner || p.PlayerGameResultStatus == proto.PlayerGameResultStatus_PLAYER_WIN {
			return p.PlayerId, p.TemporaryAddress, true
		}
	}
	return 0, "", false
}

func playerForfeited(gr *dao.GameResult, playerID int64) bool {
	for _, p := range gr.PlayerResultInfos {
		if p == nil || p.PlayerId != playerID {
			continue
		}
		switch p.PlayerGameResultStatus {
		case proto.PlayerGameResultStatus_PLAYER_OFFLINE, proto.PlayerGameResultStatus_PLAYER_SURRENDER:
			return true
		default:
			return false
		}
	}
	return false
}
