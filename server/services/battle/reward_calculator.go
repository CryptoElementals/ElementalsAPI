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
	tokenReward := rc.calculateTokenReward(result, result.GameResultType)
	scoreReward := rc.calculateScoreReward(result, result.GameResultType)

	return &BattleReward{
		TokenReward: *tokenReward,
		ScoreReward: *scoreReward,
	}
}

// calculateTokenReward calculate token rewards
func (rc *RewardCalculator) calculateTokenReward(result *BattleResult, gameResult string) *TokenReward {
	baseStake := rc.BaseStake
	systemFeeRate := 0.016 // 1.6% system fee

	systemFee := int(float64(baseStake) * result.GameFinalMultiplier * systemFeeRate)

	var player1TokenChange, player2TokenChange int

	switch gameResult {
	case "win", "ko":
		if result.Winner == result.Player1Address {
			player1Reward := int(float64(baseStake) * result.GameFinalMultiplier * (1.0 - systemFeeRate))
			player1TokenChange = player1Reward
			player2TokenChange = -int(float64(baseStake) * result.GameFinalMultiplier)
		} else {
			player2Reward := int(float64(baseStake) * result.GameFinalMultiplier * (1.0 - systemFeeRate))
			player2TokenChange = player2Reward
			player1TokenChange = -int(float64(baseStake) * result.GameFinalMultiplier)
		}

	case "tie":
		player1TokenChange = -int(float64(baseStake) * result.GameFinalMultiplier * 0.8)
		player2TokenChange = -int(float64(baseStake) * result.GameFinalMultiplier * 0.8)
	}

	return &TokenReward{
		Player1TokenChange: player1TokenChange,
		Player2TokenChange: player2TokenChange,
		SystemFee:          systemFee,
	}
}

// calculateScoreReward calculate score rewards
func (rc *RewardCalculator) calculateScoreReward(result *BattleResult, gameResult string) *ScoreReward {
	baseStake := rc.BaseStake

	var player1ScoreChange, player2ScoreChange int

	switch gameResult {
	case "win":
		if result.Winner == result.Player1Address {
			player1ScoreChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.012) // 1.2%
			player2ScoreChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.004) // 0.4%
		} else {
			player1ScoreChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.004) // 0.4%
			player2ScoreChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.012) // 1.2%
		}

	case "ko":
		if result.Winner == result.Player1Address {
			player1ScoreChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.016) // 1.6%
			player2ScoreChange = 0
		} else {
			player1ScoreChange = 0
			player2ScoreChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.016) // 1.6%
		}

	case "tie":
		player1ScoreChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.008) // 0.8%
		player2ScoreChange = int(float64(baseStake) * result.GameFinalMultiplier * 0.008) // 0.8%
	}

	return &ScoreReward{
		Player1ScoreChange: player1ScoreChange,
		Player2ScoreChange: player2ScoreChange,
	}
}
