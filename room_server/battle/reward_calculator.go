package battle

// RewardCalculator reward calculator
type RewardCalculator struct {
	BaseStake int
}

// NewRewardCalculator create a new reward calculator
func NewRewardCalculator() *RewardCalculator {
	return &RewardCalculator{
		BaseStake: 1000,
	}
}

// CalculateRewards calculate battle rewards
func (rc *RewardCalculator) CalculateRewards(result *RoundResult) *BattleReward {
	baseStake := rc.BaseStake
	systemFeeRate := 0.016 // 1.6% system fee

	systemFee := int(float64(baseStake) * float64(result.GameFinalMultiplier) * systemFeeRate)

	var playerRewards []PlayerReward

	switch result.GameResultType {
	case GAME_NORMAL, GAME_KO:
		if result.Winner != "" && result.Winner != "tie" {
			// 胜者获得奖励
			winnerReward := PlayerReward{
				PlayerAddress: result.Winner,
				TokenChange:   int(float64(baseStake) * float64(result.GameFinalMultiplier) * (1.0 - systemFeeRate)),
				PointChange:   int(float64(baseStake) * float64(result.GameFinalMultiplier) * 0.012), // 1.2%
			}
			playerRewards = append(playerRewards, winnerReward)

			// 败者扣除赌注
			for _, player := range result.Players {
				if player.WalletAddress != result.Winner {
					loserReward := PlayerReward{
						PlayerAddress: player.WalletAddress,
						TokenChange:   -int(float64(baseStake) * float64(result.GameFinalMultiplier)),
						PointChange:   int(float64(baseStake) * float64(result.GameFinalMultiplier) * 0.004), // 0.4%
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
				PlayerAddress: player.WalletAddress,
				TokenChange:   -int(float64(baseStake) * float64(result.GameFinalMultiplier) * 0.8),
				PointChange:   int(float64(baseStake) * float64(result.GameFinalMultiplier) * 0.008), // 0.8%
			}
			playerRewards = append(playerRewards, tieReward)
		}
	}

	// KO 特殊处理积分
	if result.GameResultType == GAME_KO && result.Winner != "" && result.Winner != "tie" {
		for i := range playerRewards {
			if playerRewards[i].PlayerAddress == result.Winner {
				playerRewards[i].PointChange = int(float64(baseStake) * float64(result.GameFinalMultiplier) * 0.016) // 1.6%
			} else {
				// 败者积分为0
				playerRewards[i].PointChange = 0
			}
		}
	}

	return &BattleReward{
		PlayerRewards: playerRewards,
		SystemFee:     systemFee,
	}
}
