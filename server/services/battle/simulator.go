package battle

import (
	"fmt"

	"github.com/CryptoElementals/common/log"
)

// BattleSimulator 对战推演器
type BattleSimulator struct {
	cardFactory     *CardFactory
	elementalSystem *ElementalSystem
	multiplierCalc  *MultiplierCalculator
	actionGenerator *ActionGenerator
	gameLogic       *GameLogic
	battleCache     *BattleCache
}

// NewBattleSimulator 创建新的对战推演器
func NewBattleSimulator() *BattleSimulator {
	return &BattleSimulator{
		cardFactory:     NewCardFactory(),
		elementalSystem: NewElementalSystem(),
		multiplierCalc:  NewMultiplierCalculator(),
		actionGenerator: NewActionGenerator(),
		gameLogic:       NewGameLogic(),
		battleCache:     NewBattleCache(),
	}
}

// SimulateStage 推演单个阶段对战
func (bs *BattleSimulator) SimulateStage(input *StageBattleInput, stage int) (*StageBattleResult, error) {
	// 验证输入
	if len(input.Player1Cards) != 3 || len(input.Player2Cards) != 3 {
		return nil, fmt.Errorf("每个玩家必须有3张卡牌")
	}

	// 解析卡牌
	player1Cards := bs.cardFactory.ParseCards(input.Player1Cards)
	player2Cards := bs.cardFactory.ParseCards(input.Player2Cards)

	// 创建对战结果
	result := &StageBattleResult{
		Stage:             stage,
		Player1Address:    input.Player1Address,
		Player2Address:    input.Player2Address,
		Player1HP:         input.Player1HP,
		Player2HP:         input.Player2HP,
		Player1Multiplier: input.Player1Multiplier,
		Player2Multiplier: input.Player2Multiplier,
		BattleResults:     make([]CardBattleResult, 0),
		IsGameOver:        false,
	}

	// 阶段总效果值
	stagePlayer1Effect := 0
	stagePlayer2Effect := 0

	// 统计每个玩家打出的生和克的数量
	player1ShengCount := 0
	player1KeCount := 0
	player2ShengCount := 0
	player2KeCount := 0

	// 进行3轮卡牌对战，收集所有效果值
	for i := 0; i < 3; i++ {
		battleResult := bs.elementalSystem.BattleCards(player1Cards[i], player2Cards[i], input.Player1Address, input.Player2Address)
		result.BattleResults = append(result.BattleResults, battleResult)

		// 统计玩家打出的生和克
		if battleResult.ResultType == "sheng" {
			player1ShengCount++
		} else if battleResult.ResultType == "ke" {
			player1KeCount++
		} else if battleResult.ResultType == "beisheng" {
			player2ShengCount++
		} else if battleResult.ResultType == "beike" {
			player2KeCount++
		}

		// 累加效果值
		stagePlayer1Effect += battleResult.EffectValue[0]
		stagePlayer2Effect += battleResult.EffectValue[1]
	}

	// 计算倍率更新
	player1MultiplierUpdate := bs.multiplierCalc.CalculateMultiplierUpdate(player1ShengCount, player1KeCount)
	player2MultiplierUpdate := bs.multiplierCalc.CalculateMultiplierUpdate(player2ShengCount, player2KeCount)

	// 直接设置倍率（而不是乘以）
	result.Player1Multiplier = player1MultiplierUpdate
	result.Player2Multiplier = player2MultiplierUpdate

	// 统一应用阶段效果值到玩家HP
	result.Player1HP += stagePlayer1Effect
	result.Player2HP += stagePlayer2Effect

	// 检查游戏是否结束
	if isGameOver, winner := bs.gameLogic.CheckGameOver(result.Player1HP, result.Player2HP, input.Player1Address, input.Player2Address, stage); isGameOver {
		result.IsGameOver = true
		result.Winner = winner
	}

	// 生成详细动作数据
	result.DetailedActions = bs.actionGenerator.GenerateDetailedActions(result, input.Player1Address, input.Player2Address)

	// 自动缓存详细动作数据
	bs.battleCache.CacheDetailedActions(input.Player1Address, stage, result.DetailedActions)

	return result, nil
}

// GetDetailedActionsFromCache 从缓存获取详细动作
func (bs *BattleSimulator) GetDetailedActionsFromCache(roomID string, stage int) ([]BattleAction, bool) {
	return bs.battleCache.GetDetailedActionsFromCache(roomID, stage)
}

// GetBattleCache 获取对战缓存实例（用于调试）
func (bs *BattleSimulator) GetBattleCache() *BattleCache {
	return bs.battleCache
}

// SimulateBattle 模拟完整对战（3个阶段）
func (bs *BattleSimulator) SimulateBattle(roomID, player1Address, player2Address string, player1Cards, player2Cards []string) (*BattleSimulation, error) {
	// 验证卡牌数量
	if len(player1Cards) != 9 || len(player2Cards) != 9 {
		return nil, fmt.Errorf("每个玩家必须有9张卡牌（3个阶段，每阶段3张）")
	}

	// 创建对战结果
	battleInfo := &BattleSimulation{
		RoomID: roomID,
		Player1: Player{
			Address:    player1Address,
			HP:         3000,  // 初始血量
			Multiplier: 1.0,   // 初始倍率
			IsMyself:   false, // 这个字段需要在API层设置
		},
		Player2: Player{
			Address:    player2Address,
			HP:         3000,  // 初始血量
			Multiplier: 1.0,   // 初始倍率
			IsMyself:   false, // 这个字段需要在API层设置
		},
		Actions:    make([]BattleAction, 0),
		IsGameOver: false,
	}

	// 跟踪每个阶段的倍率
	stageMultipliers := make([]float64, 3)
	player1StageMultipliers := make([]float64, 3)
	player2StageMultipliers := make([]float64, 3)

	// 逐阶段进行对战（stage 1-3）
	for stage := 1; stage <= 3; stage++ {
		// 计算当前阶段的卡牌索引
		startIndex := (stage - 1) * 3
		endIndex := startIndex + 3

		// 获取当前阶段的卡牌
		stagePlayer1Cards := player1Cards[startIndex:endIndex]
		stagePlayer2Cards := player2Cards[startIndex:endIndex]

		// 计算当前阶段的倍率
		multiplier := 1.0 + float64(stage-1)*0.5

		// 创建阶段对战输入
		input := &StageBattleInput{
			Player1Address:    player1Address,
			Player2Address:    player2Address,
			Player1HP:         battleInfo.Player1.HP,
			Player2HP:         battleInfo.Player2.HP,
			Player1Multiplier: multiplier,
			Player2Multiplier: multiplier,
			Player1Cards:      stagePlayer1Cards,
			Player2Cards:      stagePlayer2Cards,
		}

		// 执行阶段对战
		stageResult, err := bs.SimulateStage(input, stage)
		if err != nil {
			return nil, fmt.Errorf("阶段%d对战失败: %v", stage, err)
		}

		// 记录阶段倍率
		stageMultipliers[stage-1] = multiplier
		player1StageMultipliers[stage-1] = stageResult.Player1Multiplier
		player2StageMultipliers[stage-1] = stageResult.Player2Multiplier

		// 更新玩家血量
		battleInfo.Player1.HP = stageResult.Player1HP
		battleInfo.Player2.HP = stageResult.Player2HP

		// 将阶段结果转换为动作
		stageActions := bs.actionGenerator.ConvertStageResultToActions(stageResult, player1Address, player2Address)
		battleInfo.Actions = append(battleInfo.Actions, stageActions...)

		// 如果游戏结束，设置获胜者并退出
		if stageResult.IsGameOver {
			battleInfo.IsGameOver = true
			battleInfo.GameResult = stageResult.Winner
			break
		}
	}

	// 如果3个阶段都完成了但游戏没有结束，比较血量决定胜负
	if !battleInfo.IsGameOver {
		if battleInfo.Player1.HP > battleInfo.Player2.HP {
			battleInfo.GameResult = player1Address
		} else if battleInfo.Player2.HP > battleInfo.Player1.HP {
			battleInfo.GameResult = player2Address
		} else {
			battleInfo.GameResult = "tie" // 平局
		}
	}

	// 计算赢家在stage 1-3中的最高倍率
	log.Infof("winnerMaxMultiplier player1StageMultipliers: %v", player1StageMultipliers)
	log.Infof("winnerMaxMultiplier player2StageMultipliers: %v", player2StageMultipliers)

	winnerMaxMultiplier := bs.multiplierCalc.CalculateWinnerMaxMultiplier(
		battleInfo.GameResult,
		player1Address,
		player2Address,
		player1StageMultipliers,
		player2StageMultipliers,
	)

	// 执行stage 10（最终阶段），双方都使用赢家的最高倍率
	stage10Result, err := bs.gameLogic.SettleGameResult(player1Address, player2Address, battleInfo.Player1.HP, battleInfo.Player2.HP, winnerMaxMultiplier)
	if err != nil {
		return nil, fmt.Errorf("stage 10对战失败: %v", err)
	}

	// 更新最终血量
	battleInfo.Player1.HP = stage10Result.Player1HP
	battleInfo.Player2.HP = stage10Result.Player2HP

	// 更新玩家倍率（stage 10中双方倍率相同，都是赢家的最高倍率）
	battleInfo.Player1.Multiplier = winnerMaxMultiplier
	battleInfo.Player2.Multiplier = winnerMaxMultiplier

	// 将stage 10结果转换为动作
	stage10Actions := bs.actionGenerator.ConvertStageResultToActions(stage10Result, player1Address, player2Address)
	battleInfo.Actions = append(battleInfo.Actions, stage10Actions...)

	// stage 10肯定游戏结束，直接设置结果
	battleInfo.IsGameOver = true
	battleInfo.GameResult = stage10Result.Winner

	// 设置赢家的最终倍率
	battleInfo.WinnerMultiplier = winnerMaxMultiplier

	return battleInfo, nil
}
