package battle

import (
	"fmt"

	"github.com/CryptoElementals/common/config"
	pb "github.com/CryptoElementals/common/rpc/proto"
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

	// 检查是否是服务器超时导致的回合结束
	if input.Reason == pb.RoundCompleteReason_ROUND_COMPLETE_SERVER_CHAIN_TIMEOUT ||
		input.Reason == pb.RoundCompleteReason_ROUND_COMPLETE_SERVER_INTERNAL_TIMEOUT {
		// 直接返回游戏结果，双方算作平局，token为0，point为0
		return be.gameLogic.handleServerTimeoutRound(input, playerCount)
	}

	// 检查是否有玩家投降
	hasSurrenderedPlayer := false
	for _, p := range input.Players {
		if p.Status == PLAYER_SURRENDERED {
			hasSurrenderedPlayer = true
			break
		}
	}

	// 检查是否都提交了commitment，没提交的视为离线
	hasOfflinePlayer := false
	for _, p := range input.Players {
		if p.Status == PLAYER_OFFLINE {
			hasOfflinePlayer = true
			break
		}
	}

	//检查是否都提交了卡牌
	if !hasOfflinePlayer {
		for i := range input.Players {
			p := &input.Players[i]
			if len(p.Cards) != 3 {
				p.Status = PLAYER_OFFLINE
				hasOfflinePlayer = true
			} else {
				// 数量为3的同时校验卡牌内容是否合规，不合规也视为离线
				if err := be.gameLogic.validateCardElements(p.Cards, fmt.Sprintf("Player %d", i+1)); err != nil {
					p.Status = PLAYER_OFFLINE
					hasOfflinePlayer = true
				}
			}
		}
	}

	// 获取所有在线玩家的卡牌（只有在没有离线玩家时才需要获取）
	playerCards := make([][]*Card, playerCount)
	if !hasOfflinePlayer {
		for i, p := range input.Players {
			if p.Status == PLAYER_ONLINE {
				cards, err := be.cardFactory.GetCards(p.Cards)
				if err != nil {
					input.Players[i].Status = PLAYER_OFFLINE
					hasOfflinePlayer = true
					continue
				}
				playerCards[i] = cards
			}
		}
	}

	// 初始化每个玩家的状态
	type playerState struct {
		HP               int
		Multiplier       uint32
		LostHP           int
		Stats            []PlayerCardStat
		WalletAddress    string
		TemporaryAddress string
		Status           PlayerStatus
	}
	states := make([]*playerState, playerCount)
	for i, p := range input.Players {
		states[i] = &playerState{
			HP:               p.HP,
			Multiplier:       be.multiplierCalc.CalculateMultiplierByLostHP(p.LostHP),
			LostHP:           p.LostHP,
			Stats:            make([]PlayerCardStat, 0, 3), //如果有人离线，跳过对战，这个字段为空
			WalletAddress:    p.WalletAddress,
			TemporaryAddress: p.TemporaryAddress,
			Status:           p.Status,
		}
	}

	// 如果有离线玩家或投降玩家，直接跳过卡牌对战进入结算
	if !hasOfflinePlayer && !hasSurrenderedPlayer {
		//目前只有2人对战，这里实际上只会取到i=0,j=1，处理一次card对战
		//如果3人，应该在每个round的每个card的3次card对战结束后再统一结算effect，包括effect的记录和计算
		//3人的情况下可能effect里要加一个source，表示是哪个玩家发起的动作，目前只在description里有，由于现在不需要前端实现，所以暂时不加
		//多人情况下怎么解决合谋作弊问题，即两个人串通让第三个人输？未解决这个问题，所以现在游戏里没有3人对战的模式
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
					// 下限 0，不设置上限
					if p1.HP < 0 {
						p1.HP = 0
					}
					if p2.HP < 0 {
						p2.HP = 0
					}
					// 计算此次卡牌造成的伤害，并累加到总 LostHP, 如果超过初始血量，则设置为初始血量
					damage1 := p1BeforeHP - p1.HP
					if damage1 < 0 {
						damage1 = 0
					}
					p1.LostHP += damage1
					if p1.LostHP > int(config.GameParams.InitialHP) {
						p1.LostHP = int(config.GameParams.InitialHP)
					}

					damage2 := p2BeforeHP - p2.HP
					if damage2 < 0 {
						damage2 = 0
					}
					p2.LostHP += damage2
					if p2.LostHP > int(config.GameParams.InitialHP) {
						p2.LostHP = int(config.GameParams.InitialHP)
					}
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

			// 检查是否有玩家血量为0，如果有就结束游戏
			hasZeroHP := false
			for _, st := range states {
				if st.HP == 0 {
					hasZeroHP = true
					break
				}
			}
			if hasZeroHP {
				break
			}
		}
	}

	// 构建玩家回合数据
	playerStats := make([]PlayerRoundStat, playerCount)
	for i, p := range states {
		playerStats[i] = PlayerRoundStat{
			WalletAddress:    p.WalletAddress,
			TemporaryAddress: p.TemporaryAddress,
			LostHP:           p.LostHP,
			CardStats:        p.Stats, // 如果有人离线，跳过对战，这个字段为空
		}
	}

	// 游戏结束判定 - 所有卡牌都打完后的最终判定
	gameEndState := make([]*GameEndState, playerCount)
	for i, st := range states {
		gameEndState[i] = &GameEndState{
			HP:               st.HP,
			Multiplier:       st.Multiplier,
			WalletAddress:    st.WalletAddress,
			TemporaryAddress: st.TemporaryAddress,
			Status:           st.Status, // 添加状态字段
		}
	}

	isGameOver, grType, winner, temporaryAddress, finalMul := be.gameLogic.CheckGameOver(gameEndState, input.RoundNumber)

	// 先构建回合结果（暂不含 GameResult）
	roundRes := &RoundResult{
		Players:     playerStats,
		RoundNumber: input.RoundNumber,
		IsGameOver:  isGameOver,
		GameResult:  nil,
	}

	// 如果游戏结束，再构建 GameResult 并计算奖励
	if isGameOver {
		gameRes := &GameResult{
			Multiplier:             finalMul,
			WinnerWalletAddress:    winner,
			WinnerTemporaryAddress: temporaryAddress,
			GameResultType:         grType,
		}

		// 先将 GameResult 赋给 roundRes，以便奖励计算器使用
		roundRes.GameResult = gameRes

		// 构建玩家状态映射
		playerStatuses := make(map[string]PlayerStatus)
		for _, st := range states {
			playerStatuses[st.TemporaryAddress] = st.Status
		}

		// 计算奖励并填充
		gameRes.Reward = be.rewardCalculator.CalculateRewards(roundRes, playerStatuses)
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
