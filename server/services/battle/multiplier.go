package battle

// MultiplierCalculator 倍率计算器
type MultiplierCalculator struct{}

// NewMultiplierCalculator 创建新的倍率计算器
func NewMultiplierCalculator() *MultiplierCalculator {
	return &MultiplierCalculator{}
}

// CalculateMultiplierUpdate 根据生和克的数量计算倍率更新
func (mc *MultiplierCalculator) CalculateMultiplierUpdate(shengCount, keCount int) float64 {
	// 如果生或克的数量超过2个，触发倍率
	if shengCount == 2 {
		return 2.0
	} else if shengCount == 3 {
		return 4.0
	} else if keCount == 2 {
		return 4.0
	} else if keCount == 3 {
		return 8.0
	}

	// 没有触发倍率条件，返回1.0
	return 1.0
}

// CalculateWinnerMaxMultiplier 计算赢家在stage 1-3中的最高倍率
func (mc *MultiplierCalculator) CalculateWinnerMaxMultiplier(gameResult string, player1Addr, player2Addr string, player1StageMultipliers, player2StageMultipliers []float64) float64 {
	var winnerMaxMultiplier float64 = 1.0

	if gameResult != "tie" {
		winnerAddress := gameResult
		var winnerMultipliers []float64

		if winnerAddress == player1Addr {
			winnerMultipliers = player1StageMultipliers
		} else {
			winnerMultipliers = player2StageMultipliers
		}

		// 找到最高倍率
		for _, multiplier := range winnerMultipliers {
			if multiplier > winnerMaxMultiplier {
				winnerMaxMultiplier = multiplier
			}
		}
	} else {
		winnerMaxMultiplier = 0.0
	}

	return winnerMaxMultiplier
}
