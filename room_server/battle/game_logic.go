package battle

import "fmt"

// GameEndState 用于游戏结束判定的玩家状态
type GameEndState struct {
	HP               int
	Multiplier       uint32
	WalletAddress    string
	TemporaryAddress string
}

// GameLogic game logic
type GameLogic struct{}

// NewGameLogic create a new game logic
func NewGameLogic() *GameLogic {
	return &GameLogic{}
}

// CheckGameOver check if game is over
// 返回是否结束、游戏结果类型、赢家地址列表（用|分割）、赢家临时地址列表（用|分割）、最终倍率
func (gl *GameLogic) CheckGameOver(states []*GameEndState, round uint32) (bool, GameResultType, string, string, uint32) {
	// 从states中提取需要的数据
	hps := make([]int, len(states))
	addresses := make([]string, len(states))
	temps := make([]string, len(states))
	multipliers := make([]uint32, len(states))

	for i, state := range states {
		hps[i] = state.HP
		addresses[i] = state.WalletAddress
		temps[i] = state.TemporaryAddress
		multipliers[i] = state.Multiplier
	}

	// 首先检查所有人血量是否相同
	allSameHP := true
	firstHP := hps[0]
	for _, hp := range hps[1:] {
		if hp != firstHP {
			allSameHP = false
			break
		}
	}

	// 平局判定：
	// 1. 如果所有人血量都是0，不管哪一轮都是平局
	// 2. 如果所有人血量相同但不是0，只有第3轮才是平局
	if allSameHP {
		if firstHP == 0 {
			// 所有人血量都是0，直接平局
			return true, GAME_TIE, "", "", 1
		} else if round == 3 {
			// 所有人血量相同但不是0，第3轮才平局
			return true, GAME_TIE, "", "", 1
		}
		// 所有人血量相同但不是0且不是第3轮，游戏继续
		return false, GAME_NORMAL, "", "", 1
	}

	// 血量不相同的情况
	// 检查是否有人血量为0
	hasZeroHP := false
	for _, hp := range hps {
		if hp == 0 {
			hasZeroHP = true
			break
		}
	}

	var winners []string
	var winnerTemps []string
	var gameType GameResultType
	var finalMultiplier uint32 = 1

	if hasZeroHP {
		// 有人血量为0：血量为0的是输家，其余是赢家，游戏结束类型是KO
		gameType = GAME_KO
		maxLoserMul := uint32(1)
		for i, hp := range hps {
			if hp > 0 {
				winners = append(winners, addresses[i])
				winnerTemps = append(winnerTemps, temps[i])
			} else {
				// 这是输家，更新最大倍率
				if multipliers[i] > maxLoserMul {
					maxLoserMul = multipliers[i]
				}
			}
		}
		finalMultiplier = maxLoserMul
	} else {
		// 没人血量为0：只有在第3轮才比较剩余血量
		if round != 3 {
			// 未结束
			return false, GAME_NORMAL, "", "", 1
		}

		gameType = GAME_NORMAL
		// 找到最高血量
		maxHP := -1
		for _, hp := range hps {
			if hp > maxHP {
				maxHP = hp
			}
		}

		// 血量最多的是赢家（可能多个），其余是输家
		maxLoserMul := uint32(1)
		for i, hp := range hps {
			if hp == maxHP {
				winners = append(winners, addresses[i])
				winnerTemps = append(winnerTemps, temps[i])
			} else {
				// 这是输家，更新最大倍率
				if multipliers[i] > maxLoserMul {
					maxLoserMul = multipliers[i]
				}
			}
		}
		finalMultiplier = maxLoserMul
	}

	// 拼接赢家地址（用|分割）
	winnersStr := ""
	winnerTempsStr := ""
	if len(winners) > 0 {
		winnersStr = winners[0]
		winnerTempsStr = winnerTemps[0]
		for i := 1; i < len(winners); i++ {
			winnersStr += "|" + winners[i]
			winnerTempsStr += "|" + winnerTemps[i]
		}
	}

	return true, gameType, winnersStr, winnerTempsStr, finalMultiplier
}

// ValidateRoundInput validate battle input
func (gl *GameLogic) ValidateRoundInput(input *RoundInput) error {
	if len(input.Players) < 2 {
		return fmt.Errorf("at least 2 players required")
	}
	for idx, p := range input.Players {
		if p.WalletAddress == "" {
			return fmt.Errorf("player %d address cannot be empty", idx+1)
		}
		if p.TemporaryAddress == "" {
			return fmt.Errorf("player %d temporary address cannot be empty", idx+1)
		}
		if p.HP <= 0 {
			return fmt.Errorf("player %d HP must be greater than 0", idx+1)
		}
		if len(p.Cards) != 3 {
			return fmt.Errorf("player %d must have 3 cards", idx+1)
		}
		if err := gl.validateCardElements(p.Cards, fmt.Sprintf("Player %d", idx+1)); err != nil {
			return err
		}
	}
	return nil
}

// validateCardElements validate card element types
func (gl *GameLogic) validateCardElements(cardIDs []int, playerName string) error {
	cardFactory := NewCardFactory()
	elementTypes := make(map[string]bool)
	cardIDSet := make(map[int]bool)
	validElements := map[string]bool{
		"Metal": true,
		"Wood":  true,
		"Water": true,
		"Fire":  true,
		"Earth": true,
	}

	for i, cardID := range cardIDs {
		if cardIDSet[cardID] {
			return fmt.Errorf("%s card ID duplicated: %d", playerName, cardID)
		}
		cardIDSet[cardID] = true

		card, err := cardFactory.GetCard(cardID)
		if err != nil {
			return fmt.Errorf("%s %dth card failed to get: %v", playerName, i+1, err)
		}

		if !validElements[card.ElementType] {
			return fmt.Errorf("%s %dth card element type invalid: %s", playerName, i+1, card.ElementType)
		}

		if elementTypes[card.ElementType] {
			return fmt.Errorf("%s card element type duplicated: %s", playerName, card.ElementType)
		}

		elementTypes[card.ElementType] = true
	}

	if len(elementTypes) != 3 {
		return fmt.Errorf("%s cards must contain 3 different element types", playerName)
	}

	return nil
}
