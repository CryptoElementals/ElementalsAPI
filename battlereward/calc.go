package battlereward

import (
	"cmp"
	"slices"
	"strings"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

// ComputeBattleRewardAmounts fills TokenChange, PointChange on each PlayerReward and SystemFee on BattleReward
// from persisted game outcome metadata. The room server persists per-player flags (offline, surrendered, status)
// and leaves token/point amounts and system fee at zero; lobby settlement runs this before writing amounts and applying wallets.
// Call only when gr.BattleReward and PlayerRewards are non-empty (settlement validates this).
func ComputeBattleRewardAmounts(gr *dao.GameResult, baseStake int) {
	br := gr.BattleReward
	if br == nil || len(br.PlayerRewards) == 0 {
		return
	}
	prs := br.PlayerRewards

	slices.SortFunc(prs, func(a, b *dao.PlayerReward) int {
		if c := cmp.Compare(a.PlayerId, b.PlayerId); c != 0 {
			return c
		}
		return strings.Compare(strings.ToLower(a.TemporaryAddress), strings.ToLower(b.TemporaryAddress))
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
		tokenDeduction := int(float64(baseStake) * 0.008)
		pointGain := int(float64(baseStake) * 0.008)
		br.SystemFee = int32(tokenDeduction * len(prs))
		for _, pr := range prs {
			pr.TokenChange = int32(-tokenDeduction)
			pr.PointChange = int32(pointGain)
		}

	case proto.GameResultType_GAME_NORMAL, proto.GameResultType_GAME_KO:
		winnerTemp := gr.WinnerTemporaryAddress
		if winnerTemp == "" || gr.WinnerPlayerId == 0 {
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
		totalPool := int(float64(baseStake) * float64(poolMul))

		winnerTokenPerPlayer := int(float64(totalPool) * (1.0 - 0.016))
		loserTokenPerPlayer := totalPool / loserCount

		var winnerPointPerPlayer, loserPointPerPlayer int
		if gr.GameResultType == proto.GameResultType_GAME_NORMAL {
			winnerPointPerPlayer = int(float64(totalPool) * 0.012)
			loserPointPerPlayer = int(float64(totalPool)*0.004) / loserCount
		} else {
			winnerPointPerPlayer = int(float64(totalPool) * 0.016)
			loserPointPerPlayer = 0
		}

		bonusPointsForWinner := 0
		for _, pr := range prs {
			if strings.EqualFold(pr.TemporaryAddress, winnerTemp) {
				continue
			}
			if pr.Surrendered || pr.IsOffline {
				bonusPointsForWinner += loserPointPerPlayer
			}
		}

		br.SystemFee = int32(int(float64(totalPool) * 0.016))

		for _, pr := range prs {
			if strings.EqualFold(pr.TemporaryAddress, winnerTemp) {
				pr.TokenChange = int32(winnerTokenPerPlayer)
				pr.PointChange = int32(winnerPointPerPlayer + bonusPointsForWinner)
			} else {
				pr.TokenChange = int32(-loserTokenPerPlayer)
				if pr.Surrendered || pr.IsOffline {
					pr.PointChange = 0
				} else {
					pr.PointChange = int32(loserPointPerPlayer)
				}
			}
		}
	}
}
