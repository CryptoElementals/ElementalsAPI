package battle

import (
	"fmt"
	"strings"
)

type BattleEngine struct {
	cardFactory      *CardFactory
	elementalSystem  *ElementalSystem
	multiplierCalc   *MultiplierCalculator
	gameLogic        *GameLogic
	rewardCalculator *RewardCalculator
}

func NewBattleEngine() *BattleEngine {
	return &BattleEngine{
		cardFactory:      NewCardFactory(),
		elementalSystem:  NewElementalSystem(),
		multiplierCalc:   NewMultiplierCalculator(),
		gameLogic:        NewGameLogic(),
		rewardCalculator: NewRewardCalculator(),
	}
}

// ExecuteRound execute round
func (be *BattleEngine) ExecuteRound(input *RoundInput) (*RoundResult, error) {
	if err := be.gameLogic.ValidateRoundInput(input); err != nil {
		return nil, err
	}

	if input.RoundNumber < 1 || input.RoundNumber > 3 {
		return nil, fmt.Errorf("round parameter must be between 1 and 3")
	}

	playerCount := len(input.Players)
	if playerCount < 2 {
		return nil, fmt.Errorf("at least 2 players required")
	}

	// 获取所有玩家的卡牌
	playerCards := make([][]*Card, playerCount)
	for i, p := range input.Players {
		cards, err := be.cardFactory.GetCards(p.Cards)
		if err != nil {
			return nil, err
		}
		playerCards[i] = cards
	}

	// 初始化每个玩家的状态
	type playerState struct {
		HP               int
		Multiplier       uint32
		LostHP           int
		Stats            []PlayerCardStat
		WalletAddress    string
		TemporaryAddress string
	}
	states := make([]*playerState, playerCount)
	for i, p := range input.Players {
		states[i] = &playerState{
			HP:               p.HP,
			Multiplier:       p.Multiplier,
			LostHP:           p.LostHP,
			Stats:            make([]PlayerCardStat, 0, 3),
			WalletAddress:    p.WalletAddress,
			TemporaryAddress: p.TemporaryAddress,
		}
	}

	// 只保留属于自己的effect
	filterEffects := func(effects []BattleEffect, target string) []BattleEffect {
		var filtered []BattleEffect
		for _, e := range effects {
			if e.Target == target {
				filtered = append(filtered, e)
			}
		}
		return filtered
	}

	for cardIdx := 0; cardIdx < 3; cardIdx++ {
		for i := 0; i < playerCount; i++ {
			for j := i + 1; j < playerCount; j++ {
				p1 := states[i]
				p2 := states[j]
				p1Card := playerCards[i][cardIdx]
				p2Card := playerCards[j][cardIdx]
				relation := be.elementalSystem.GetElementalRelation(p1Card, p2Card)
				effects := be.elementalSystem.BuildEffects(p1Card, p2Card, relation, p1.WalletAddress, p2.WalletAddress)
				effects1 := filterEffects(effects, p1.WalletAddress)
				effects2 := filterEffects(effects, p2.WalletAddress)
				p1BeforeHP := p1.HP
				p2BeforeHP := p2.HP
				p1BeforeMul := p1.Multiplier
				p2BeforeMul := p2.Multiplier
				p1.HP += be.elementalSystem.ExecuteEffects(effects1)
				p2.HP += be.elementalSystem.ExecuteEffects(effects2)
				if p1.HP < 0 {
					p1.HP = 0
				}
				if p2.HP < 0 {
					p2.HP = 0
				}
				p1.LostHP = input.Players[i].HP - p1.HP
				p2.LostHP = input.Players[j].HP - p2.HP
				p1.Multiplier = be.multiplierCalc.CalculateMultiplierByLostHP(p1.LostHP)
				p2.Multiplier = be.multiplierCalc.CalculateMultiplierByLostHP(p2.LostHP)
				// 生成p1和p2视角的描述（元素类型+视角），并根据relation.Type选择正确模板
				descP1 := relation.Description
				descP2 := relation.Description
				switch relation.Type {
				case "overpower":
					descP2 = "{self} is overpowered by {opponent}"
				case "overpowered":
					descP2 = "{self} overpowers {opponent}"
				case "nurture":
					descP2 = "{self} nurtures {opponent}"
				case "nurtured":
					descP2 = "{self} is nurtured by {opponent}"
				case "even":
					descP2 = "{self} and {opponent} are even"
				}

				descP1 = strings.ReplaceAll(descP1, "{self}", fmt.Sprintf("%s(self)", p1Card.ElementType))
				descP1 = strings.ReplaceAll(descP1, "{opponent}", fmt.Sprintf("%s(opponent)", p2Card.ElementType))

				descP2 = strings.ReplaceAll(descP2, "{self}", fmt.Sprintf("%s(self)", p2Card.ElementType))
				descP2 = strings.ReplaceAll(descP2, "{opponent}", fmt.Sprintf("%s(opponent)", p1Card.ElementType))

				p1.Stats = append(p1.Stats, PlayerCardStat{
					CardNumber:       cardIdx + 1,
					CardID:           p1Card.ID,
					HPBefore:         p1BeforeHP,
					HPAfter:          p1.HP,
					MultiplierBefore: p1BeforeMul,
					MultiplierAfter:  p1.Multiplier,
					Effects:          effects1,
					Description:      descP1,
					ElementRelation:  mapElementRelationStringToEnum(relation.Type),
				})
				p2.Stats = append(p2.Stats, PlayerCardStat{
					CardNumber:       cardIdx + 1,
					CardID:           p2Card.ID,
					HPBefore:         p2BeforeHP,
					HPAfter:          p2.HP,
					MultiplierBefore: p2BeforeMul,
					MultiplierAfter:  p2.Multiplier,
					Effects:          effects2,
					Description:      descP2,
					ElementRelation:  mapElementRelationStringToEnum(reverseRelationType(relation.Type)),
				})

				// 适配新的CheckGameOver - 每张牌对战后检查，此时卡牌未全部打完
				hps := make([]int, playerCount)
				addrs := make([]string, playerCount)
				for idx, st := range states {
					hps[idx] = st.HP
					addrs[idx] = st.WalletAddress
				}
				if isGameOver, _ := be.gameLogic.CheckGameOver(hps, addrs, uint(input.RoundNumber), false); isGameOver {
					goto END
				}
			}
		}
	}
END:
	// 构建玩家回合数据
	playerStats := make([]PlayerRoundStat, playerCount)
	for i, p := range states {
		playerStats[i] = PlayerRoundStat{
			WalletAddress:    p.WalletAddress,
			TemporaryAddress: p.TemporaryAddress,
			LostHP:           p.LostHP,
			CardStats:        p.Stats,
		}
	}

	// 游戏结束判定 - 所有卡牌都打完后的最终判定
	hps := make([]int, playerCount)
	addrs := make([]string, playerCount)
	for idx, st := range states {
		hps[idx] = st.HP
		addrs[idx] = st.WalletAddress
	}
	isGameOver, winner := be.gameLogic.CheckGameOver(hps, addrs, uint(input.RoundNumber), true)

	// 确定游戏结果类型和最终倍率
	var gameResultType GameResultType
	var gameFinalMultiplier uint32

	if isGameOver {
		gameResultType = be.determineGameResultType(hps, addrs)
		// 计算最终倍率（取败者倍率，平局为1）
		if winner != "tie" {
			for _, st := range states {
				if st.WalletAddress != winner {
					gameFinalMultiplier = st.Multiplier
					break
				}
			}
		} else {
			gameFinalMultiplier = 1.0
		}
	} else {
		// 游戏未结束时，GameResultType为空，GameFinalMultiplier为0
		gameResultType = GAME_NORMAL
		gameFinalMultiplier = 0.0
	}

	result := &RoundResult{
		Players:             playerStats,
		RoundNumber:         input.RoundNumber,
		GameFinalMultiplier: gameFinalMultiplier,
		Winner:              winner,
		IsGameOver:          isGameOver,
		GameResultType:      gameResultType,
		Reward:              nil, // 先设为nil
	}

	// 计算奖励
	if isGameOver {
		result.Reward = be.rewardCalculator.CalculateRewards(result)
	}

	return result, nil
}

// determineGameResultType determine game result type
func (be *BattleEngine) determineGameResultType(hps []int, addresses []string) GameResultType {
	alive := 0
	for _, hp := range hps {
		if hp > 0 {
			alive++
		}
	}
	if alive == 0 {
		return GAME_TIE
	} else if alive == 1 {
		return GAME_KO
	} else {
		return GAME_NORMAL
	}
}

// reverseRelationType 用于反转元素关系类型
func reverseRelationType(t string) string {
	switch t {
	case "overpower":
		return "overpowered"
	case "overpowered":
		return "overpower"
	case "nurture":
		return "nurtured"
	case "nurtured":
		return "nurture"
	default:
		return t
	}
}

func mapElementRelationStringToEnum(s string) ElementRelation {
	switch s {
	case "overpower":
		return OVER_POWER
	case "overpowered":
		return OVER_POWERED
	case "nurture":
		return NURTURE
	case "nurtured":
		return NURTURED
	default:
		return TIE
	}
}
