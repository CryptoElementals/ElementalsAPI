package battle

import (
	"fmt"

	"github.com/CryptoElementals/common/config"
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
			Multiplier:       be.multiplierCalc.CalculateMultiplierByLostHP(p.LostHP),
			LostHP:           p.LostHP,
			Stats:            make([]PlayerCardStat, 0, 3),
			WalletAddress:    p.WalletAddress,
			TemporaryAddress: p.TemporaryAddress,
		}
	}

	// 只保留属于自己的effect
	//目前只有2人对战，这里实际上只会取到i=0,j=1，处理一次card对战
	//如果3人，应该在每个round的每个card的3次card对战结束后再统一结算effect，包括effect的记录和计算
	for cardIdx := 0; cardIdx < 3; cardIdx++ {
		for i := 0; i < playerCount; i++ {
			for j := i + 1; j < playerCount; j++ {
				p1 := states[i]
				p2 := states[j]
				p1Card := playerCards[i][cardIdx]
				p2Card := playerCards[j][cardIdx]
				relation := be.elementalSystem.GetElementalRelation(p1Card, p2Card, p1.WalletAddress, p2.WalletAddress)
				effects1 := be.elementalSystem.BuildPlayerEffects(relation.P1Type, p1Card, p2Card, p1.WalletAddress, p1.TemporaryAddress, p2.WalletAddress)
				effects2 := be.elementalSystem.BuildPlayerEffects(relation.P2Type, p2Card, p1Card, p2.WalletAddress, p2.TemporaryAddress, p1.WalletAddress)
				p1BeforeHP := p1.HP
				p2BeforeHP := p2.HP
				p1BeforeMul := p1.Multiplier
				p2BeforeMul := p2.Multiplier
				p1.HP += be.elementalSystem.ExecuteEffects(effects1)
				p2.HP += be.elementalSystem.ExecuteEffects(effects2)
				// 下限 0，上限 MaxHP
				if p1.HP < 0 {
					p1.HP = 0
				} else if p1.HP > config.GameParams.MaxHP {
					p1.HP = config.GameParams.MaxHP
				}
				if p2.HP < 0 {
					p2.HP = 0
				} else if p2.HP > config.GameParams.MaxHP {
					p2.HP = config.GameParams.MaxHP
				}
				// 计算此次卡牌造成的伤害，并累加到总 LostHP
				damage1 := p1BeforeHP - p1.HP
				if damage1 < 0 {
					damage1 = 0
				}
				p1.LostHP += damage1

				damage2 := p2BeforeHP - p2.HP
				if damage2 < 0 {
					damage2 = 0
				}
				p2.LostHP += damage2
				p1.Multiplier = be.multiplierCalc.CalculateMultiplierByLostHP(p1.LostHP)
				p2.Multiplier = be.multiplierCalc.CalculateMultiplierByLostHP(p2.LostHP)

				p1.Stats = append(p1.Stats, PlayerCardStat{
					CardNumber:       cardIdx + 1,
					CardID:           p1Card.ID,
					HPBefore:         p1BeforeHP,
					HPAfter:          p1.HP,
					MultiplierBefore: p1BeforeMul,
					MultiplierAfter:  p1.Multiplier,
					Effects:          effects1,
					Description:      relation.P1Description,
					ElementRelation:  mapElementRelationStringToEnum(relation.P1Type),
				})
				p2.Stats = append(p2.Stats, PlayerCardStat{
					CardNumber:       cardIdx + 1,
					CardID:           p2Card.ID,
					HPBefore:         p2BeforeHP,
					HPAfter:          p2.HP,
					MultiplierBefore: p2BeforeMul,
					MultiplierAfter:  p2.Multiplier,
					Effects:          effects2,
					Description:      relation.P2Description,
					ElementRelation:  mapElementRelationStringToEnum(relation.P2Type),
				})

			}
		}
		// 每个round的每张牌对战后都检查一次游戏是否结束，而不是每个人3张卡牌全部打完再结算
		hps := make([]int, playerCount)
		addrs := make([]string, playerCount)
		temps := make([]string, playerCount)
		for idx, st := range states {
			hps[idx] = st.HP
			addrs[idx] = st.WalletAddress
		}
		if isGameOver, _, _, _ := be.gameLogic.CheckGameOver(hps, addrs, temps, input.RoundNumber, false); isGameOver {
			break
		}
	}

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
	temps := make([]string, playerCount)
	for idx, st := range states {
		hps[idx] = st.HP
		addrs[idx] = st.WalletAddress
		temps[idx] = st.TemporaryAddress
	}

	isGameOver, grType, winner, temporaryAddress := be.gameLogic.CheckGameOver(hps, addrs, temps, input.RoundNumber, true)

	// 先构建回合结果（暂不含 GameResult）
	roundRes := &RoundResult{
		Players:     playerStats,
		RoundNumber: input.RoundNumber,
		IsGameOver:  isGameOver,
		GameResult:  nil,
	}

	// 如果游戏结束，再构建 GameResult 并计算奖励
	if isGameOver {
		var finalMul uint32
		if winner != "" {
			for _, st := range states {
				if st.WalletAddress != winner {
					finalMul = st.Multiplier
					break
				}
			}
		} else {
			finalMul = 1
		}

		gameRes := &GameResult{
			Multiplier:             finalMul,
			WinnerWalletAddress:    winner,
			WinnerTemporaryAddress: temporaryAddress,
			GameResultType:         grType,
		}

		// 先将 GameResult 赋给 roundRes，以便奖励计算器使用
		roundRes.GameResult = gameRes
		// 计算奖励并填充
		gameRes.Reward = be.rewardCalculator.CalculateRewards(roundRes)
	}

	return roundRes, nil
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
