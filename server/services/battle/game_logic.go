package battle

// GameLogic 游戏逻辑系统
type GameLogic struct{}

// NewGameLogic 创建新的游戏逻辑系统
func NewGameLogic() *GameLogic {
	return &GameLogic{}
}

// CheckGameOver 检查游戏是否结束
func (gl *GameLogic) CheckGameOver(player1HP, player2HP int, player1Addr, player2Addr string, stage int) (bool, string) {
	// 如果有一方血量为0或负数，游戏结束
	if player1HP <= 0 || player2HP <= 0 {
		if player1HP <= 0 {
			return true, player2Addr
		} else {
			return true, player1Addr
		}
	}

	// stage 3和stage 10的特殊判断：如果双方血量都大于0，比较血量决定胜负
	if stage == 3 || stage == 10 {
		if player1HP > player2HP {
			return true, player1Addr
		} else if player2HP > player1HP {
			return true, player2Addr
		} else {
			// 血量相同，平局
			return true, "tie"
		}
	}

	return false, ""
}

// SettleGameResult 清点胜负（stage 10）
func (gl *GameLogic) SettleGameResult(player1Address, player2Address string, player1HP, player2HP int, finalMultiplier float64) (*StageBattleResult, error) {
	// stage 10不需要实际的卡牌对战，只是应用最终倍率
	// 创建stage 10结果
	result := &StageBattleResult{
		Stage:             10,
		Player1Address:    player1Address,
		Player2Address:    player2Address,
		Player1HP:         player1HP,
		Player2HP:         player2HP,
		Player1Multiplier: finalMultiplier,             // 双方都使用赢家的最高倍率
		Player2Multiplier: finalMultiplier,             // 双方都使用赢家的最高倍率
		BattleResults:     make([]CardBattleResult, 0), // stage 10没有卡牌对战
		IsGameOver:        true,                        // stage 10肯定游戏结束
	}

	// stage 10直接比较血量决定胜负
	if player1HP > player2HP {
		result.Winner = player1Address
	} else if player2HP > player1HP {
		result.Winner = player2Address
	} else {
		result.Winner = "tie" // 平局
	}

	return result, nil
}

// abs 绝对值函数
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
