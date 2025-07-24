package battle

import "github.com/CryptoElementals/common/config"

// RewardCalculator reward calculator
type RewardCalculator struct {
	BaseStake int
}

// NewRewardCalculator create a new reward calculator
func NewRewardCalculator() *RewardCalculator {
	return &RewardCalculator{BaseStake: config.GameParams.BaseStake}
}

// CalculateRewards calculate battle rewards
func (rc *RewardCalculator) CalculateRewards(result *RoundResult) BattleReward {
	// 如果游戏尚未结束或 GameResult 为空，直接返回 nil
	if result == nil || result.GameResult == nil {
		return BattleReward{}
	}

	gr := result.GameResult

	baseStake := rc.BaseStake
	systemFeeRate := config.GameParams.SystemFeeRate

	systemFee := int(float64(baseStake) * float64(gr.Multiplier) * systemFeeRate)

	var playerRewards []PlayerReward

	switch gr.GameResultType {
	case GAME_NORMAL, GAME_KO:
		if gr.WinnerWalletAddress != "" {
			// 胜者获得奖励
			winnerReward := PlayerReward{
				WalletAddress:    gr.WinnerWalletAddress,
				TemporaryAddress: gr.WinnerTemporaryAddress,
				TokenChange:      int(float64(baseStake) * float64(gr.Multiplier) * (1.0 - systemFeeRate)),
				PointChange:      int(float64(baseStake) * float64(gr.Multiplier) * config.GameParams.WinnerPointRate),
			}
			playerRewards = append(playerRewards, winnerReward)

			// 败者扣除赌注
			for _, player := range result.Players {
				if player.WalletAddress != gr.WinnerWalletAddress {
					loserReward := PlayerReward{
						WalletAddress:    player.WalletAddress,
						TemporaryAddress: player.TemporaryAddress,
						TokenChange:      -int(float64(baseStake) * float64(gr.Multiplier)),
						PointChange:      int(float64(baseStake) * float64(gr.Multiplier) * config.GameParams.LoserPointRate),
					}
					playerRewards = append(playerRewards, loserReward)
					break
				}
			}
		}
	case GAME_TIE:
		// 平局时所有玩家都扣除部分赌注
		for _, player := range result.Players {
			tieReward := PlayerReward{
				WalletAddress:    player.WalletAddress,
				TemporaryAddress: player.TemporaryAddress,
				TokenChange:      -int(float64(baseStake) * float64(gr.Multiplier) * config.GameParams.TieTokenRate),
				PointChange:      int(float64(baseStake) * float64(gr.Multiplier) * config.GameParams.TiePointRate),
			}
			playerRewards = append(playerRewards, tieReward)
		}
	}

	// KO 特殊处理积分
	if gr.GameResultType == GAME_KO && gr.WinnerWalletAddress != "" {
		for i := range playerRewards {
			if playerRewards[i].WalletAddress == gr.WinnerWalletAddress {
				playerRewards[i].PointChange = int(float64(baseStake) * float64(gr.Multiplier) * config.GameParams.WinnerPointRate)
			} else {
				// 败者积分为0
				playerRewards[i].PointChange = 0
			}
		}
	}

	return BattleReward{
		PlayerRewards: playerRewards,
		SystemFee:     systemFee,
	}
}
