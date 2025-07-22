package battle

import "fmt"

// GameLogic game logic
type GameLogic struct{}

// NewGameLogic create a new game logic
func NewGameLogic() *GameLogic {
	return &GameLogic{}
}

// CheckGameOver check if game is over
// 改为支持任意数量玩家，返回是否结束和胜者地址（或空/"tie"）
// 添加round参数，支持第3轮特殊规则
// allCardsPlayed: 是否所有卡牌都已打完
func (gl *GameLogic) CheckGameOver(hps []int, addresses []string, round uint, allCardsPlayed bool) (bool, string) {
	alive := 0
	winner := ""
	for i, hp := range hps {
		if hp > 0 {
			alive++
			winner = addresses[i]
		}
	}

	// 如果有玩家血量为0，直接判定胜负
	if alive == 1 {
		return true, winner
	}
	if alive == 0 {
		return true, "tie"
	}

	// 第3轮特殊规则：只有在所有卡牌都打完后，且所有玩家都还活着时，才比较血量
	if round == 3 && allCardsPlayed && alive > 1 {
		maxHP := -1
		winner = ""
		tie := false

		for i, hp := range hps {
			if hp > maxHP {
				maxHP = hp
				winner = addresses[i]
				tie = false
			} else if hp == maxHP {
				tie = true
			}
		}

		if tie {
			return true, "tie"
		} else {
			return true, winner
		}
	}

	return false, ""
}

// ValidateRoundInput validate battle input
// 改为校验Players列表
func (gl *GameLogic) ValidateRoundInput(input *RoundInput) error {
	if len(input.Players) < 2 {
		return fmt.Errorf("at least 2 players required")
	}
	for idx, p := range input.Players {
		if p.WalletAddress == "" {
			return fmt.Errorf("player %d address cannot be empty", idx+1)
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

// GetWinner determine winner based on health
func (gl *GameLogic) GetWinner(player1HP, player2HP int, player1Address, player2Address string) string {
	if player1HP > player2HP {
		return player1Address
	} else if player2HP > player1HP {
		return player2Address
	} else {
		return "tie"
	}
}
