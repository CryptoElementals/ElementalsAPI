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
func (rc *RewardCalculator) CalculateRewards(result *BattleResult) *BattleReward {
	baseStake := rc.BaseStake
	systemFeeRate := 0.016 // 1.6% system fee

	systemFee := int(float64(baseStake) * result.GameFinalMultiplier * systemFeeRate)

	var player1TokenChange, player2TokenChange int
	var player1PointChange, player2PointChange int

	switch result.GameResultType {
	case "normal", "ko":
		if result.Winner == result.Player1Address {
			player1TokenChange = int(float64(baseStake) * result.GameFinalMultiplier * (1.0 - systemFeeRate))
			player2TokenChange = -int(float64(baseStake) * result.GameFinalMultiplier)
		} else {
			player2TokenChange = int(float64(baseStake) * result.GameFinalMultiplier * (1.0 - systemFeeRate))
			player1TokenChange = -int(float64(baseStake) * result.GameFinalMultiplier)
		}
	case "tie":
		player1TokenChange = -int(float64(baseStake) * result.GameFinalMultiplier * 0.8)
		player2TokenChange = -int(float64(baseStake) * result.GameFinalMultiplier * 0.8)
	}

	switch result.GameResultType {
	case "normal":
		if result.Winner == result.Player1Address {
			player1PointChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.012) // 1.2%
			player2PointChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.004) // 0.4%
		} else {
			player1PointChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.004) // 0.4%
			player2PointChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.012) // 1.2%
		}
	case "ko":
		if result.Winner == result.Player1Address {
			player1PointChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.016) // 1.6%
			player2PointChange = 0
		} else {
			player1PointChange = 0
			player2PointChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.016) // 1.6%
		}
	case "tie":
		player1PointChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.008) // 0.8%
		player2PointChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.008) // 0.8%
	}

	return &BattleReward{
		Player1TokenChange: player1TokenChange,
		Player2TokenChange: player2TokenChange,
		SystemFee:          systemFee,
		Player1PointChange: player1PointChange,
		Player2PointChange: player2PointChange,
	}
}
