package battle

import (
	"fmt"
)

// GameEndState 用于游戏结束判定的玩家状态
type GameEndState struct {
	HP               int
	Multiplier       uint32
	WalletAddress    string
	TemporaryAddress string
	Status           PlayerStatus // 添加状态字段
}

// GameLogic game logic
type GameLogic struct{}

// NewGameLogic create a new game logic
func NewGameLogic() *GameLogic {
	return &GameLogic{}
}

// check if game is over
// 返回是否结束、游戏结果类型、赢家地址列表（用|分割）、赢家临时地址列表（用|分割）、最终倍率
func (gl *GameLogic) CheckGameOver(states []*GameEndState, round uint32) (bool, GameResultType, string, string, uint32) {
	// 首先检查投降玩家情况
	surrenderedCount := 0
	for _, state := range states {
		if state.Status == PLAYER_SURRENDERED {
			surrenderedCount++
		}
	}

	// 如果有投降玩家，需要特殊处理
	if surrenderedCount > 0 {
		if surrenderedCount == len(states) {
			// 全员投降，直接平局
			return true, GAME_TIE, "", "", 1
		}
		if surrenderedCount < len(states) {
			// 有投降玩家：未投降的是赢家，投降的是输家，应该是KO
			var winners []string
			var winnerTemps []string
			var maxLoserMul uint32 = 1

			for _, state := range states {
				if state.Status == PLAYER_SURRENDERED {
					// 投降玩家是输家，更新最大倍率
					if state.Multiplier > maxLoserMul {
						maxLoserMul = state.Multiplier
					}
				} else {
					winners = append(winners, state.WalletAddress)
					winnerTemps = append(winnerTemps, state.TemporaryAddress)
				}
			}

			// 拼接赢家地址
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

			return true, GAME_KO, winnersStr, winnerTempsStr, maxLoserMul
		}
	}

	// 然后检查离线玩家情况
	onlineCount := 0
	offlineCount := 0

	for _, state := range states {
		if state.Status == PLAYER_ONLINE {
			onlineCount++
		} else if state.Status == PLAYER_OFFLINE {
			offlineCount++
		}
	}
	// 如果有离线玩家，需要特殊处理
	if offlineCount > 0 {
		if offlineCount == len(states) {
			// 全员未提交 Commitment 或未提交卡牌，直接平局
			return true, GAME_TIE, "", "", 1
		}
		if onlineCount > 0 {
			// 有在线玩家：在线的是赢家，离线的是输家，应该是normal（血量都大于0）
			var winners []string
			var winnerTemps []string
			var maxLoserMul uint32 = 1

			for _, state := range states {
				if state.Status == PLAYER_ONLINE {
					winners = append(winners, state.WalletAddress)
					winnerTemps = append(winnerTemps, state.TemporaryAddress)
				} else if state.Status == PLAYER_OFFLINE {
					// 离线玩家是输家，更新最大倍率
					if state.Multiplier > maxLoserMul {
						maxLoserMul = state.Multiplier
					}
				}
			}

			// 拼接赢家地址
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

			return true, GAME_NORMAL, winnersStr, winnerTempsStr, maxLoserMul
		}
	}

	// 没有离线玩家，使用原有逻辑
	return gl.checkGameOverByHP(states, round, false)
}

// checkGameOverByHP 按血量判断游戏结束的逻辑
// hasOffline: 如果为true，表示有离线玩家，立即结算；否则按原有轮数规则
func (gl *GameLogic) checkGameOverByHP(states []*GameEndState, round uint32, hasOffline bool) (bool, GameResultType, string, string, uint32) {
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
	if allSameHP {
		if firstHP == 0 {
			// 所有人血量都是0，直接平局
			return true, GAME_TIE, "", "", 1
		} else if hasOffline || round == 3 {
			// 有离线玩家时立即判断，或者第3轮血量相同时为平局
			return true, GAME_TIE, "", "", 1
		}
		// 血量相同但不是第3轮且没有离线玩家，游戏继续
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
		// 没人血量为0：有离线玩家时立即判断，否则只有在第3轮才比较剩余血量
		if !hasOffline && round != 3 {
			// 没有离线玩家且不是第3轮，未结束
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

	// 首先处理所有玩家的基本验证和默认状态
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
		if len(p.Cards) > 3 {
			return fmt.Errorf("player %d must have at most 3 cards", idx+1)
		}

		// Status字段默认为PLAYER_ONLINE(0)，无需显式设置
	}

	// 然后处理未提交的情况（强制设为离线）
	for idx, p := range input.Players {
		// 如果玩家投降，设置为投降状态（优先级最高）
		if p.Surrendered {
			input.Players[idx].Status = PLAYER_SURRENDERED
			continue
		}

		// 如果未提交 Commitment，则视为离线
		if len(p.Commitment) == 0 {
			input.Players[idx].Status = PLAYER_OFFLINE
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

// handleServerTimeoutRound 处理服务器超时导致的回合结束
func (gl *GameLogic) handleServerTimeoutRound(input *RoundInput, playerCount int) (*RoundResult, error) {
	// 构建玩家回合数据（空的卡牌统计）
	playerStats := make([]PlayerRoundStat, playerCount)
	for i, p := range input.Players {
		playerStats[i] = PlayerRoundStat{
			WalletAddress:    p.WalletAddress,
			TemporaryAddress: p.TemporaryAddress,
			LostHP:           p.LostHP,
			CardStats:        []PlayerCardStat{}, // 空的卡牌统计
		}
	}

	// 创建游戏结果 - 平局，token为0，point为0
	gameRes := &GameResult{
		Multiplier:             0,        // 倍率为0
		WinnerWalletAddress:    "",       // 平局，无胜者
		WinnerTemporaryAddress: "",       // 平局，无胜者
		GameResultType:         GAME_TIE, // 平局类型
		Reward: BattleReward{
			PlayerRewards: make([]PlayerReward, playerCount),
			SystemFee:     0, // 系统手续费为0
		},
	}

	// 为每个玩家设置奖励（token为0，point为0）
	for i, p := range input.Players {
		// 根据玩家的状态设置 IsOffline 和 IsSurrendered
		isOffline := false
		isSurrendered := false

		// 如果玩家投降，设置为投降状态
		if p.Surrendered {
			isSurrendered = true
		} else if len(p.Commitment) == 0 || len(p.Cards) < 3 {
			// 如果未提交 Commitment 或卡牌数量不足，则视为离线
			isOffline = true
		}

		gameRes.Reward.PlayerRewards[i] = PlayerReward{
			WalletAddress:    p.WalletAddress,
			TemporaryAddress: p.TemporaryAddress,
			TokenChange:      0, // token变化为0
			PointChange:      0, // point变化为0
			IsOffline:        isOffline,
			IsSurrendered:    isSurrendered,
		}
	}

	// 构建回合结果
	roundRes := &RoundResult{
		Players:     playerStats,
		RoundNumber: input.RoundNumber,
		IsGameOver:  true, // 游戏结束
		GameResult:  gameRes,
	}

	return roundRes, nil
}
