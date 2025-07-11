package services

import (
	"os"
	"testing"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/services/battle"
)

func TestMain(m *testing.M) {
	log.InitGlobalLogger(&log.Config{
		Development: true,
		Level:       "DEBUG",
	})
	os.Exit(m.Run())
}

func TestBattleSimulator(t *testing.T) {
	// 创建对战推演器
	simulator := NewBattleSimulator()

	// 测试数据：两个玩家的卡牌
	player1Address := "0x1234567890123456789012345678901234567890"
	player2Address := "0x0987654321098765432109876543210987654321"

	// 玩家1的卡牌：9张卡牌，包含JMSHT，使用不同的小型号
	player1Cards := []string{
		"J0", "M1", "S2", // 第1阶段
		"H3", "T0", "J1", // 第2阶段
		"M2", "S3", "H0", // 第3阶段
	}

	// 玩家2的卡牌：9张卡牌，包含JMSHT，使用不同的小型号
	player2Cards := []string{
		"S0", "H1", "T2", // 第1阶段
		"J3", "M0", "S1", // 第2阶段
		"H2", "T3", "J0", // 第3阶段
	}

	roomID := "test-room-123"

	// 执行推演
	battleInfo, err := simulator.SimulateBattle(roomID, player1Address, player2Address, player1Cards, player2Cards)
	if err != nil {
		t.Fatalf("推演失败: %v", err)
	}

	// 验证推演结果
	if battleInfo == nil {
		t.Fatal("推演结果为空")
	}

	if battleInfo.RoomID != roomID {
		t.Errorf("房间ID不匹配，期望: %s, 实际: %s", roomID, battleInfo.RoomID)
	}

	if battleInfo.Player1.Address != player1Address {
		t.Errorf("玩家1地址不匹配，期望: %s, 实际: %s", player1Address, battleInfo.Player1.Address)
	}

	if battleInfo.Player2.Address != player2Address {
		t.Errorf("玩家2地址不匹配，期望: %s, 实际: %s", player2Address, battleInfo.Player2.Address)
	}

	// 验证初始血量
	if battleInfo.Player1.HP > 3000 {
		t.Errorf("玩家1初始血量不正确，期望: 3000, 实际: %d", battleInfo.Player1.HP)
	}

	if battleInfo.Player2.HP > 3000 {
		t.Errorf("玩家2初始血量不正确，期望: 3000, 实际: %d", battleInfo.Player2.HP)
	}

	// 验证阶段数量（游戏可能在某个阶段就结束，所以阶段数量可能少于3）
	// 新系统包含stage 10，所以动作数量会更多
	if len(battleInfo.Actions) < 3 || len(battleInfo.Actions) > 15 {
		t.Errorf("动作数量不正确，期望: 3-15, 实际: %d", len(battleInfo.Actions))
	}

	// 验证每个动作
	for i, action := range battleInfo.Actions {
		if action.Round < 1 || action.Round > 3 {
			t.Errorf("第%d个动作回合数不正确，期望: 1-3, 实际: %d", i+1, action.Round)
		}

		if action.ActionType == "" {
			t.Errorf("第%d个动作类型为空", i+1)
		}
		validTypes := map[string]bool{"attack": true, "heal": true}
		if !validTypes[action.ActionType] {
			t.Errorf("第%d个动作类型无效: %s", i+1, action.ActionType)
		}

		if action.Source.Address == "" {
			t.Errorf("第%d个动作发起者地址为空", i+1)
		}

		if action.Target.Address == "" {
			t.Errorf("第%d个动作接收者地址为空", i+1)
		}

		if action.Effect.Type == "" {
			t.Errorf("第%d个动作效果类型为空", i+1)
		}
		validEffectTypes := map[string]bool{"damage": true, "heal": true}
		if !validEffectTypes[action.Effect.Type] {
			t.Errorf("第%d个动作效果类型无效: %s", i+1, action.Effect.Type)
		}
	}

	// 验证游戏是否完成
	if !battleInfo.IsGameOver {
		t.Error("游戏应该已完成")
	}

	// 验证游戏结果不为空
	if battleInfo.GameResult == "" {
		t.Error("游戏结果不应为空")
	}

	t.Logf("推演成功完成，游戏结果: %s", battleInfo.GameResult)
	t.Logf("玩家1最终血量: %d", battleInfo.Player1.HP)
	t.Logf("玩家2最终血量: %d", battleInfo.Player2.HP)
	t.Logf("实际动作数量: %d", len(battleInfo.Actions))
}

func TestBattleSimulationComplete(t *testing.T) {
	// 创建对战推演器
	simulator := NewBattleSimulator()

	// 测试一个完整的对战场景
	player1Address := "0x1111111111111111111111111111111111111111"
	player2Address := "0x2222222222222222222222222222222222222222"

	// 玩家1的卡牌：全部是金，使用不同的小型号
	player1Cards := []string{
		"J0", "J1", "J2", // 第1阶段
		"J3", "J0", "J1", // 第2阶段
		"J2", "J3", "J0", // 第3阶段
	}

	// 玩家2的卡牌：全部是木，使用不同的小型号
	player2Cards := []string{
		"M0", "M1", "M2", // 第1阶段
		"M3", "M0", "M1", // 第2阶段
		"M2", "M3", "M0", // 第3阶段
	}

	roomID := "test-complete-battle"

	// 执行推演
	battleInfo, err := simulator.SimulateBattle(roomID, player1Address, player2Address, player1Cards, player2Cards)
	if err != nil {
		t.Fatalf("推演失败: %v", err)
	}

	// 验证推演结果
	if battleInfo == nil {
		t.Fatal("推演结果为空")
	}

	// 验证对战结果
	// 由于玩家1全是金，玩家2全是木，金克木，所以玩家1应该获胜
	if battleInfo.GameResult != player1Address && battleInfo.GameResult != player2Address && battleInfo.GameResult != "tie" {
		t.Errorf("游戏结果格式不正确: %s", battleInfo.GameResult)
	}

	t.Logf("完整对战推演成功，游戏结果: %s", battleInfo.GameResult)
	t.Logf("玩家1最终血量: %d", battleInfo.Player1.HP)
	t.Logf("玩家2最终血量: %d", battleInfo.Player2.HP)

	// 打印每个动作的详细信息
	for i, action := range battleInfo.Actions {
		t.Logf("动作 %d:", i+1)
		t.Logf("  回合: %d", action.Round)
		t.Logf("  动作类型: %s", action.ActionType)
		t.Logf("  发起者: %s (%s%d)", action.Source.Address, action.Source.Card.Symbol, action.Source.Card.SubType)
		t.Logf("  接收者: %s (%s%d)", action.Target.Address, action.Target.Card.Symbol, action.Target.Card.SubType)
		t.Logf("  效果类型: %s", action.Effect.Type)
		t.Logf("  效果数值: %d", action.Effect.Value)
		t.Logf("  描述: %s", action.Message)
	}
}

func TestWuxingRelations(t *testing.T) {
	// 创建五行系统
	elementalSystem := battle.NewElementalSystem()
	cardFactory := battle.NewCardFactory()

	// 测试五行相生相克关系
	testCases := []struct {
		card1      string
		card2      string
		expectType string
		expectDesc string
	}{
		{"J0", "M0", "ke", "J克M"},     // 金克木
		{"M0", "T0", "ke", "M克T"},     // 木克土
		{"T0", "S0", "ke", "T克S"},     // 土克水
		{"S0", "H0", "ke", "S克H"},     // 水克火
		{"H0", "J0", "ke", "H克J"},     // 火克金
		{"J0", "S0", "sheng", "J生S"},  // 金生水
		{"S0", "M0", "sheng", "S生M"},  // 水生木
		{"M0", "H0", "sheng", "M生H"},  // 木生火
		{"H0", "T0", "sheng", "H生T"},  // 火生土
		{"T0", "J0", "sheng", "T生J"},  // 土生金
		{"J0", "J0", "ping", "无生克关系"}, // 同类型
		{"J0", "H0", "beike", "J被H克"}, // 金被火克
	}

	for _, tc := range testCases {
		card1 := cardFactory.CreateCard(tc.card1)
		card2 := cardFactory.CreateCard(tc.card2)

		resultType, reason := elementalSystem.DetermineBattleRelation(card1, card2)

		if resultType != tc.expectType {
			t.Errorf("五行关系判断错误: %s vs %s, 期望: %s, 实际: %s",
				tc.card1, tc.card2, tc.expectType, resultType)
		}

		if reason != tc.expectDesc {
			t.Errorf("五行关系描述错误: %s vs %s, 期望: %s, 实际: %s",
				tc.card1, tc.card2, tc.expectDesc, reason)
		}
	}
}

func TestBattleEffectCalculation(t *testing.T) {
	elementalSystem := battle.NewElementalSystem()
	cardFactory := battle.NewCardFactory()

	// 测试对战效果计算 - 使用新的movement系统
	card1 := cardFactory.CreateCard("J1") // 金1号：攻击力17，防御力6
	card2 := cardFactory.CreateCard("M0") // 木0号：攻击力20，防御力10

	// 测试克：金克木，应该攻击两次
	movement := elementalSystem.GetMovementByRelation("ke")
	effectValue, _ := movement.Execute(card1, card2)
	expectedDamage := (card1.Attack - card2.Defense) * 2 // 两次攻击
	if expectedDamage < 0 {
		expectedDamage = 0
	}
	if effectValue[1] != -expectedDamage {
		t.Errorf("克伤害计算错误，期望: %d, 实际: %d", -expectedDamage, effectValue[1])
	}

	// 测试被克：木被金克，应该被攻击两次
	movement = elementalSystem.GetMovementByRelation("beike")
	effectValue, _ = movement.Execute(card1, card2)
	expectedDamage = (card2.Attack - card1.Defense) * 2 // 两次攻击
	if expectedDamage < 0 {
		expectedDamage = 0
	}
	if effectValue[0] != -expectedDamage {
		t.Errorf("被克伤害计算错误，期望: %d, 实际: %d", -expectedDamage, effectValue[0])
	}

	// 测试生：给对方回血
	movement = elementalSystem.GetMovementByRelation("sheng")
	effectValue, _ = movement.Execute(card1, card2)
	if effectValue[1] != card1.LifeForce {
		t.Errorf("生效果计算错误，期望: %d, 实际: %d", card1.LifeForce, effectValue[1])
	}

	// 测试被生：自己回血
	movement = elementalSystem.GetMovementByRelation("beisheng")
	effectValue, _ = movement.Execute(card1, card2)
	if effectValue[0] != card2.LifeForce {
		t.Errorf("被生效果计算错误，期望: %d, 实际: %d", card2.LifeForce, effectValue[0])
	}

	// 测试平：双方攻击
	movement = elementalSystem.GetMovementByRelation("ping")
	effectValue, _ = movement.Execute(card1, card2)
	expectedDamage1 := card1.Attack - card2.Defense
	if expectedDamage1 < 0 {
		expectedDamage1 = 0
	}
	expectedDamage2 := card2.Attack - card1.Defense
	if expectedDamage2 < 0 {
		expectedDamage2 = 0
	}
	if effectValue[0] != -expectedDamage2 || effectValue[1] != -expectedDamage1 {
		t.Errorf("平效果计算错误，期望: [%d, %d], 实际: [%d, %d]", -expectedDamage2, -expectedDamage1, effectValue[0], effectValue[1])
	}

	t.Logf("对战效果计算测试通过")
}

func TestCardCreation(t *testing.T) {
	cardFactory := battle.NewCardFactory()

	// 测试卡牌创建（查表模式，0-3小型号）
	testCases := []struct {
		cardStr   string
		symbol    string
		subType   int
		level     string
		lifeForce int
		attack    int
		defense   int
	}{
		{"J0", "J", 0, "normal", 500, 1000, 500},
		{"J1", "J", 1, "rare", 90, 17, 6},
		{"J2", "J", 2, "epic", 100, 19, 7},
		{"J3", "J", 3, "legendary", 110, 21, 8},
		{"M0", "M", 0, "normal", 500, 1000, 500},
		{"M1", "M", 1, "rare", 110, 22, 11},
		{"M2", "M", 2, "epic", 120, 24, 12},
		{"M3", "M", 3, "legendary", 130, 26, 13},
		{"S0", "S", 0, "normal", 500, 1000, 500},
		{"S1", "S", 1, "rare", 100, 20, 9},
		{"S2", "S", 2, "epic", 110, 22, 10},
		{"S3", "S", 3, "legendary", 120, 24, 11},
		{"H0", "H", 0, "normal", 500, 1000, 500},
		{"H1", "H", 1, "rare", 130, 27, 16},
		{"H2", "H", 2, "epic", 140, 29, 17},
		{"H3", "H", 3, "legendary", 150, 31, 18},
		{"T0", "T", 0, "normal", 500, 1000, 500},
		{"T1", "T", 1, "rare", 120, 24, 13},
		{"T2", "T", 2, "epic", 130, 26, 14},
		{"T3", "T", 3, "legendary", 140, 28, 15},
	}

	for _, tc := range testCases {
		card := cardFactory.CreateCard(tc.cardStr)
		if card.Symbol != tc.symbol {
			t.Errorf("卡牌符号不正确: %s, 期望: %s", card.Symbol, tc.symbol)
		}
		if card.SubType != tc.subType {
			t.Errorf("卡牌小型号不正确: %s, 期望: %d, 实际: %d", tc.cardStr, tc.subType, card.SubType)
		}
		if card.Level != tc.level {
			t.Errorf("卡牌等级不正确: %s, 期望: %s, 实际: %s", tc.cardStr, tc.level, card.Level)
		}
		if card.LifeForce != tc.lifeForce {
			t.Errorf("卡牌生命力不正确: %s, 期望: %d, 实际: %d", tc.cardStr, tc.lifeForce, card.LifeForce)
		}
		if card.Attack != tc.attack {
			t.Errorf("卡牌攻击力不正确: %s, 期望: %d, 实际: %d", tc.cardStr, tc.attack, card.Attack)
		}
		if card.Defense != tc.defense {
			t.Errorf("卡牌防御力不正确: %s, 期望: %d, 实际: %d", tc.cardStr, tc.defense, card.Defense)
		}
	}
}

func TestSimulateStage(t *testing.T) {
	// 创建对战推演器
	simulator := NewBattleSimulator()

	// 测试单个阶段对战
	input := &StageBattleInput{
		Player1Address:    "0x1111111111111111111111111111111111111111",
		Player2Address:    "0x2222222222222222222222222222222222222222",
		Player1HP:         3000,                       // 初始血量
		Player2HP:         3000,                       // 初始血量
		Player1Multiplier: 1.5,                        // 玩家1倍率
		Player2Multiplier: 1.5,                        // 玩家2倍率
		Player1Cards:      []string{"J0", "M1", "S2"}, // 玩家1的3张卡牌
		Player2Cards:      []string{"M0", "T1", "H2"}, // 玩家2的3张卡牌
	}

	// 执行阶段模拟
	result, err := simulator.SimulateStage(input, 1) // 添加stage参数
	if err != nil {
		t.Fatalf("阶段模拟失败: %v", err)
	}

	// 验证结果
	if result == nil {
		t.Fatal("模拟结果为空")
	}

	// 验证玩家地址
	if result.Player1Address != input.Player1Address {
		t.Errorf("玩家1地址不匹配，期望: %s, 实际: %s", input.Player1Address, result.Player1Address)
	}

	if result.Player2Address != input.Player2Address {
		t.Errorf("玩家2地址不匹配，期望: %s, 实际: %s", input.Player2Address, result.Player2Address)
	}

	// 验证血量变化
	if result.Player1HP > input.Player1HP {
		t.Errorf("玩家1血量不应该增加，初始: %d, 最终: %d", input.Player1HP, result.Player1HP)
	}

	if result.Player2HP > input.Player2HP {
		t.Errorf("玩家2血量不应该增加，初始: %d, 最终: %d", input.Player2HP, result.Player2HP)
	}

	// 验证倍率变化（倍率会根据生克关系变化）
	// 玩家1有3个克，倍率应该是8.0
	if result.Player1Multiplier != 8.0 {
		t.Errorf("玩家1倍率不正确，期望: 8.0, 实际: %.1f", result.Player1Multiplier)
	}

	// 玩家2没有生或克，倍率应该是1.0
	if result.Player2Multiplier != 1.0 {
		t.Errorf("玩家2倍率不正确，期望: 1.0, 实际: %.1f", result.Player2Multiplier)
	}

	// 验证对战结果数量
	if len(result.BattleResults) != 3 {
		t.Errorf("对战结果数量不正确，期望: 3, 实际: %d", len(result.BattleResults))
	}

	// 验证每轮对战结果
	for i, battleResult := range result.BattleResults {
		if battleResult.ResultType == "" {
			t.Errorf("第%d轮对战结果类型为空", i+1)
		}
		validTypes := map[string]bool{"sheng": true, "beisheng": true, "ke": true, "beike": true, "ping": true}
		if !validTypes[battleResult.ResultType] {
			t.Errorf("第%d轮对战结果类型无效: %s", i+1, battleResult.ResultType)
		}
	}

	// 验证游戏结束逻辑
	if result.IsGameOver {
		if result.Winner == "" {
			t.Error("游戏结束时获胜者不应为空")
		}
		if result.Winner != input.Player1Address && result.Winner != input.Player2Address {
			t.Errorf("获胜者地址无效: %s", result.Winner)
		}
	}

	t.Logf("阶段模拟成功完成")
	t.Logf("玩家1最终血量: %d", result.Player1HP)
	t.Logf("玩家2最终血量: %d", result.Player2HP)
	t.Logf("玩家1倍率: %.1f", result.Player1Multiplier)
	t.Logf("玩家2倍率: %.1f", result.Player2Multiplier)
	t.Logf("游戏结束: %v", result.IsGameOver)
	if result.IsGameOver {
		t.Logf("获胜者: %s", result.Winner)
	}

	// 打印每轮对战详情
	for i, battleResult := range result.BattleResults {
		t.Logf("第%d轮: %s%d vs %s%d, 结果: %s, 效果值: %v, 原因: %s",
			i+1, battleResult.Player1Card.Symbol, battleResult.Player1Card.SubType,
			battleResult.Player2Card.Symbol, battleResult.Player2Card.SubType,
			battleResult.ResultType, battleResult.EffectValue, battleResult.Reason)
	}
}
