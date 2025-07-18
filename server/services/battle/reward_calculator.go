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

	systemFee := int(float64(baseStake) * result.GameFinalMultiplier * systemFeeRate)

	playerRewards := make(map[string]PlayerReward)

	switch result.GameResultType {
	case "normal", "ko":
		if result.Winner != "" && result.Winner != "tie" {
			// 胜者获得奖励
			winnerReward := PlayerReward{
				TokenChange: int(float64(baseStake) * result.GameFinalMultiplier * (1.0 - systemFeeRate)),
				PointChange: int(float64(baseStake) * result.GameFinalMultiplier * 0.012), // 1.2%
			}
			playerRewards[result.Winner] = winnerReward

			// 败者扣除赌注
			for _, player := range result.Players {
				if player.Player != result.Winner {
					loserReward := PlayerReward{
						TokenChange: -int(float64(baseStake) * result.GameFinalMultiplier),
						PointChange: int(float64(baseStake) * result.GameFinalMultiplier * 0.004), // 0.4%
					}
					playerRewards[player.Player] = loserReward
					break
				}
			}
		}
	case "tie":
		// 平局时所有玩家都扣除部分赌注
		for _, player := range result.Players {
			playerRewards[player.Player] = PlayerReward{
				TokenChange: -int(float64(baseStake) * result.GameFinalMultiplier * 0.8),
				PointChange: int(float64(baseStake) * result.GameFinalMultiplier * 0.008), // 0.8%
			}
		}
	}

	// KO 特殊处理积分
	if result.GameResultType == "ko" && result.Winner != "" && result.Winner != "tie" {
		if winnerReward, exists := playerRewards[result.Winner]; exists {
			winnerReward.PointChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.016) // 1.6%
			playerRewards[result.Winner] = winnerReward
		}
		// 败者积分为0
		for _, player := range result.Players {
			if player.Player != result.Winner {
				if loserReward, exists := playerRewards[player.Player]; exists {
					loserReward.PointChange = 0
					playerRewards[player.Player] = loserReward
				}
				break
			}
		}
	}

	return &BattleReward{
		PlayerRewards: playerRewards,
		SystemFee:     systemFee,
	}
}
