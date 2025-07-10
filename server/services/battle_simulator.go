package services

import (
	"fmt"

	"github.com/CryptoElementals/common/log"
)

// Card 卡牌结构
type Card struct {
	Symbol    string `json:"symbol"`     // 卡牌符号 (J, M, S, H, T)
	SubType   int    `json:"sub_type"`   // 小型号 (0, 1, 2, 3)
	Level     string `json:"level"`      // 卡牌等级 (legendary, epic, rare, normal)
	LifeForce int    `json:"life_force"` // 生命力
	Attack    int    `json:"attack"`     // 攻击力
	Defense   int    `json:"defense"`    // 防御力
}

// Player 玩家结构
type Player struct {
	Address    string  `json:"address"`    // 玩家地址
	Cards      []Card  `json:"cards"`      // 玩家的卡牌
	HP         int     `json:"hp"`         // 血量
	Multiplier float64 `json:"multiplier"` // 积分倍率
	IsMyself   bool    `json:"is_myself"`  // 是否是我自己
}

// StageResult 阶段结果
type StageResult struct {
	Stage             int                `json:"stage"`              // 阶段编号 (0, 1, 2)
	Player1Cards      []Card             `json:"player1_cards"`      // 玩家1的卡牌
	Player2Cards      []Card             `json:"player2_cards"`      // 玩家2的卡牌
	Player1HP         int                `json:"player1_hp"`         // 玩家1血量
	Player2HP         int                `json:"player2_hp"`         // 玩家2血量
	Player1Multiplier float64            `json:"player1_multiplier"` // 玩家1积分倍率
	Player2Multiplier float64            `json:"player2_multiplier"` // 玩家2积分倍率
	BattleResults     []CardBattleResult `json:"battle_results"`     // 卡牌对战结果
	IsGameOver        bool               `json:"is_game_over"`       // 游戏是否结束
	Winner            string             `json:"winner"`             // 获胜者地址
}

// CardBattleResult 单张卡牌对战结果
type CardBattleResult struct {
	Player1Card Card   `json:"player1_card"` // 玩家1的卡牌
	Player2Card Card   `json:"player2_card"` // 玩家2的卡牌
	ResultType  string `json:"result_type"`  // 对战结果类型 ("sheng", "beisheng", "ke", "beike", "ping")
	EffectValue []int  `json:"effect_value"` // 对战效果值数组 [玩家1受影响值, 玩家2受影响值]（正数表示回血，负数表示伤害）
	Reason      string `json:"reason"`       // 对战原因描述
}

// StageBattleInput 阶段对战输入
type StageBattleInput struct {
	Player1Address    string   `json:"player1_address"`    // 玩家1地址
	Player2Address    string   `json:"player2_address"`    // 玩家2地址
	Player1HP         int      `json:"player1_hp"`         // 玩家1初始血量
	Player2HP         int      `json:"player2_hp"`         // 玩家2初始血量
	Player1Multiplier float64  `json:"player1_multiplier"` // 玩家1积分倍率
	Player2Multiplier float64  `json:"player2_multiplier"` // 玩家2积分倍率
	Player1Cards      []string `json:"player1_cards"`      // 玩家1的3张卡牌
	Player2Cards      []string `json:"player2_cards"`      // 玩家2的3张卡牌
}

// StageBattleResult 阶段对战结果
type StageBattleResult struct {
	Stage             int                `json:"stage"`              // 阶段编号 (0, 1, 2)
	Player1Address    string             `json:"player1_address"`    // 玩家1地址
	Player2Address    string             `json:"player2_address"`    // 玩家2地址
	Player1HP         int                `json:"player1_hp"`         // 玩家1最终血量
	Player2HP         int                `json:"player2_hp"`         // 玩家2最终血量
	Player1Multiplier float64            `json:"player1_multiplier"` // 玩家1积分倍率
	Player2Multiplier float64            `json:"player2_multiplier"` // 玩家2积分倍率
	BattleResults     []CardBattleResult `json:"battle_results"`     // 3轮对战结果
	IsGameOver        bool               `json:"is_game_over"`       // 游戏是否结束
	Winner            string             `json:"winner"`             // 获胜者地址（如果游戏结束）
}

// Movement 动作接口
type Movement interface {
	Execute(card1, card2 Card) ([]int, string)
	GetName() string
}

// AttackMovement 攻击动作
type AttackMovement struct {
	attackerIndex int // 0表示玩家1攻击，1表示玩家2攻击
	times         int // 攻击次数
}

func NewAttackMovement(attackerIndex int, times int) *AttackMovement {
	return &AttackMovement{
		attackerIndex: attackerIndex,
		times:         times,
	}
}

func (am *AttackMovement) Execute(card1, card2 Card) ([]int, string) {
	effectValue := []int{0, 0}
	var reason string

	if am.attackerIndex == 0 {
		// 玩家1攻击玩家2
		totalDamage := 0
		for i := 0; i < am.times; i++ {
			damage := card1.Attack - card2.Defense
			if damage < 0 {
				damage = 0
			}
			totalDamage += damage
		}
		effectValue[1] = -totalDamage // 玩家2受伤（负数）
		reason = fmt.Sprintf("%s攻击力%d攻击%s防御力%d%d次，总伤害=%d",
			card1.Symbol, card1.Attack, card2.Symbol, card2.Defense, am.times, totalDamage)
	} else {
		// 玩家2攻击玩家1
		totalDamage := 0
		for i := 0; i < am.times; i++ {
			damage := card2.Attack - card1.Defense
			if damage < 0 {
				damage = 0
			}
			totalDamage += damage
		}
		effectValue[0] = -totalDamage // 玩家1受伤（负数）
		reason = fmt.Sprintf("%s攻击力%d攻击%s防御力%d%d次，总伤害=%d",
			card2.Symbol, card2.Attack, card1.Symbol, card1.Defense, am.times, totalDamage)
	}

	return effectValue, reason
}

func (am *AttackMovement) GetName() string {
	return fmt.Sprintf("攻击%d次", am.times)
}

// HealMovement 治疗动作
type HealMovement struct {
	healerIndex int // 0表示玩家1治疗，1表示玩家2治疗
}

func NewHealMovement(healerIndex int) *HealMovement {
	return &HealMovement{
		healerIndex: healerIndex,
	}
}

func (hm *HealMovement) Execute(card1, card2 Card) ([]int, string) {
	effectValue := []int{0, 0}
	var reason string

	if hm.healerIndex == 0 {
		// 玩家1治疗自己（被生的情况）
		effectValue[0] = card2.LifeForce // 使用对方卡牌的生命力来治疗自己
		reason = fmt.Sprintf("%s被%s生命力%d生，回血=%d", card1.Symbol, card2.Symbol, card2.LifeForce, card2.LifeForce)
	} else {
		// 玩家2治疗自己（生的情况）
		effectValue[1] = card1.LifeForce // 使用自己卡牌的生命力来治疗对方
		reason = fmt.Sprintf("%s生命力%d生%s，回血=%d", card1.Symbol, card1.LifeForce, card2.Symbol, card1.LifeForce)
	}

	return effectValue, reason
}

func (hm *HealMovement) GetName() string {
	return "治疗"
}

// DualAttackMovement 双方攻击动作
type DualAttackMovement struct{}

func NewDualAttackMovement() *DualAttackMovement {
	return &DualAttackMovement{}
}

func (dam *DualAttackMovement) Execute(card1, card2 Card) ([]int, string) {
	effectValue := []int{0, 0}

	// 玩家1攻击玩家2
	damage1 := card1.Attack - card2.Defense
	if damage1 < 0 {
		damage1 = 0
	}
	effectValue[1] = -damage1

	// 玩家2攻击玩家1
	damage2 := card2.Attack - card1.Defense
	if damage2 < 0 {
		damage2 = 0
	}
	effectValue[0] = -damage2

	reason := fmt.Sprintf("%s攻击力%d攻击%s防御力%d，伤害=%d；%s攻击力%d攻击%s防御力%d，伤害=%d",
		card1.Symbol, card1.Attack, card2.Symbol, card2.Defense, damage1,
		card2.Symbol, card2.Attack, card1.Symbol, card1.Defense, damage2)

	return effectValue, reason
}

func (dam *DualAttackMovement) GetName() string {
	return "双方攻击"
}

// BattleSimulator 对战推演器
type BattleSimulator struct{}

// NewBattleSimulator 创建新的对战推演器
func NewBattleSimulator() *BattleSimulator {
	return &BattleSimulator{}
}

// calculateMultiplierUpdate 根据生和克的数量计算倍率更新
func (bs *BattleSimulator) calculateMultiplierUpdate(shengCount, keCount int) float64 {
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

// SimulateStage10 模拟stage 10对战（最终阶段）
func (bs *BattleSimulator) SimulateStage10(player1Address, player2Address string, player1HP, player2HP int, finalMultiplier float64) (*StageBattleResult, error) {
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

// SimulateStage 推演单个阶段对战
func (bs *BattleSimulator) SimulateStage(input *StageBattleInput, stage int) (*StageBattleResult, error) {
	// 验证输入
	if len(input.Player1Cards) != 3 || len(input.Player2Cards) != 3 {
		return nil, fmt.Errorf("每个玩家必须有3张卡牌")
	}

	// 解析卡牌
	player1Cards := bs.parseCards(input.Player1Cards)
	player2Cards := bs.parseCards(input.Player2Cards)

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
		battleResult := bs.battleCards(player1Cards[i], player2Cards[i], input.Player1Address, input.Player2Address)
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
	player1MultiplierUpdate := bs.calculateMultiplierUpdate(player1ShengCount, player1KeCount)
	player2MultiplierUpdate := bs.calculateMultiplierUpdate(player2ShengCount, player2KeCount)

	// 直接设置倍率（而不是乘以）
	result.Player1Multiplier = player1MultiplierUpdate
	result.Player2Multiplier = player2MultiplierUpdate

	// 统一应用阶段效果值到玩家HP
	result.Player1HP += stagePlayer1Effect
	result.Player2HP += stagePlayer2Effect

	// 检查游戏是否结束
	if isGameOver, winner := bs.checkGameOver(result.Player1HP, result.Player2HP, input.Player1Address, input.Player2Address, stage); isGameOver {
		result.IsGameOver = true
		result.Winner = winner
	}

	return result, nil
}

// parseCards 解析卡牌字符串为Card对象
func (bs *BattleSimulator) parseCards(cardStrings []string) []Card {
	cards := make([]Card, len(cardStrings))
	for i, cardStr := range cardStrings {
		cards[i] = bs.CreateCard(cardStr)
	}
	return cards
}

// CreateCard 根据卡牌字符串创建完整的卡牌对象
// 格式: "J0", "M1", "S2", "H3", "T4" 等
func (bs *BattleSimulator) CreateCard(cardStr string) Card {
	if len(cardStr) == 0 {
		// 如果卡牌字符串为空，抛出错误
		panic("卡牌字符串不能为空")
	}

	symbol := string(cardStr[0])
	var subType int

	if len(cardStr) == 1 {
		// 如果只有Symbol没有SubType，补充为0
		subType = 0
	} else {
		// 有SubType的情况
		subType = int(cardStr[1] - '0') // 将字符转换为数字
		// 确保小型号在有效范围内
		if subType < 0 || subType > 3 {
			subType = 0
		}
	}

	cardData := bs.getCardData(symbol, subType)
	return Card{
		Symbol:    symbol,
		SubType:   subType,
		Level:     cardData.level,
		LifeForce: cardData.lifeForce,
		Attack:    cardData.attack,
		Defense:   cardData.defense,
	}
}

// cardData 卡牌数据
type cardData struct {
	level     string
	lifeForce int
	attack    int
	defense   int
}

// cardDataTable 卡牌属性查表
var cardDataTable = map[string]map[int]cardData{
	"J": {
		0: {level: "normal", lifeForce: 500, attack: 1000, defense: 500},
		1: {level: "rare", lifeForce: 90, attack: 17, defense: 6},
		2: {level: "epic", lifeForce: 100, attack: 19, defense: 7},
		3: {level: "legendary", lifeForce: 110, attack: 21, defense: 8},
	},
	"M": {
		0: {level: "normal", lifeForce: 500, attack: 1000, defense: 500},
		1: {level: "rare", lifeForce: 110, attack: 22, defense: 11},
		2: {level: "epic", lifeForce: 120, attack: 24, defense: 12},
		3: {level: "legendary", lifeForce: 130, attack: 26, defense: 13},
	},
	"S": {
		0: {level: "normal", lifeForce: 500, attack: 1000, defense: 500},
		1: {level: "rare", lifeForce: 100, attack: 20, defense: 9},
		2: {level: "epic", lifeForce: 110, attack: 22, defense: 10},
		3: {level: "legendary", lifeForce: 120, attack: 24, defense: 11},
	},
	"H": {
		0: {level: "normal", lifeForce: 500, attack: 1000, defense: 500},
		1: {level: "rare", lifeForce: 130, attack: 27, defense: 16},
		2: {level: "epic", lifeForce: 140, attack: 29, defense: 17},
		3: {level: "legendary", lifeForce: 150, attack: 31, defense: 18},
	},
	"T": {
		0: {level: "normal", lifeForce: 500, attack: 1000, defense: 500},
		1: {level: "rare", lifeForce: 120, attack: 24, defense: 13},
		2: {level: "epic", lifeForce: 130, attack: 26, defense: 14},
		3: {level: "legendary", lifeForce: 140, attack: 28, defense: 15},
	},
}

// getCardData 获取卡牌完整数据（查表实现）
func (bs *BattleSimulator) getCardData(symbol string, subType int) cardData {
	typeTable, ok := cardDataTable[symbol]
	if !ok {
		// 类型无效，返回默认
		typeTable = cardDataTable["J"]
	}
	data, ok := typeTable[subType]
	if !ok {
		// 小型号无效，返回0号
		data = typeTable[0]
	}
	return data
}

// battleCards 两张卡牌对战（五行相生相克）
func (bs *BattleSimulator) battleCards(card1, card2 Card, player1Addr, player2Addr string) CardBattleResult {
	result := CardBattleResult{
		Player1Card: card1,
		Player2Card: card2,
		EffectValue: []int{0, 0}, // 初始化效果值数组
	}

	// 判断五行关系
	resultType, baseReason := bs.DetermineBattleRelation(card1, card2)
	result.ResultType = resultType

	// 根据关系类型获取对应的动作
	movement := bs.getMovementByRelation(resultType)

	// 执行动作
	effectValue, movementReason := movement.Execute(card1, card2)
	result.EffectValue = effectValue

	// 生成详细的原因描述
	result.Reason = baseReason + "，" + movementReason

	return result
}

// getMovementByRelation 根据五行关系获取对应的动作
func (bs *BattleSimulator) getMovementByRelation(resultType string) Movement {
	switch resultType {
	case "ke":
		// 克：攻击对方两次
		return NewAttackMovement(0, 2)
	case "beike":
		// 被克：被攻击两次
		return NewAttackMovement(1, 2)
	case "sheng":
		// 生：给对方回血
		return NewHealMovement(1)
	case "beisheng":
		// 被生：自己回血
		return NewHealMovement(0)
	case "ping":
		// 平：双方各攻击对方一次
		return NewDualAttackMovement()
	default:
		// 默认无动作
		return NewAttackMovement(0, 0)
	}
}

// DetermineBattleRelation 判断五行关系
func (bs *BattleSimulator) DetermineBattleRelation(card1, card2 Card) (string, string) {
	// 五行相克关系：金克木，木克土，土克水，水克火，火克金
	keRelations := map[string]string{
		"J": "M", // 金克木
		"M": "T", // 木克土
		"T": "S", // 土克水
		"S": "H", // 水克火
		"H": "J", // 火克金
	}

	// 五行相生关系：金生水，水生木，木生火，火生土，土生金
	shengRelations := map[string]string{
		"J": "S", // 金生水
		"S": "M", // 水生木
		"M": "H", // 木生火
		"H": "T", // 火生土
		"T": "J", // 土生金
	}

	// 检查相克关系
	if keRelations[card1.Symbol] == card2.Symbol {
		return "ke", fmt.Sprintf("%s克%s", card1.Symbol, card2.Symbol)
	}
	if keRelations[card2.Symbol] == card1.Symbol {
		return "beike", fmt.Sprintf("%s被%s克", card1.Symbol, card2.Symbol)
	}

	// 检查相生关系
	if shengRelations[card1.Symbol] == card2.Symbol {
		return "sheng", fmt.Sprintf("%s生%s", card1.Symbol, card2.Symbol)
	}
	if shengRelations[card2.Symbol] == card1.Symbol {
		return "beisheng", fmt.Sprintf("%s被%s生", card1.Symbol, card2.Symbol)
	}

	// 无生克关系，双方互相攻击
	return "ping", "无生克关系"
}

// checkGameOver 检查游戏是否结束
func (bs *BattleSimulator) checkGameOver(player1HP, player2HP int, player1Addr, player2Addr string, stage int) (bool, string) {
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

// abs 绝对值函数
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// BattleSimulation 完整对战模拟结果
type BattleSimulation struct {
	RoomID           string         `json:"room_id"`           // 房间ID
	Stage            int            `json:"stage"`             // 当前阶段
	Player1          Player         `json:"player1"`           // 玩家1信息
	Player2          Player         `json:"player2"`           // 玩家2信息
	Actions          []BattleAction `json:"actions"`           // 当前阶段的动作信息
	IsGameOver       bool           `json:"is_game_over"`      // 游戏是否结束
	GameResult       string         `json:"game_result"`       // 游戏结果：win/lose/tie
	WinnerMultiplier float64        `json:"winner_multiplier"` // 赢家的最终倍率
}

// BattleAction 单次对战动作
type BattleAction struct {
	Round      int          `json:"round"`       // 回合数
	ActionType string       `json:"action_type"` // 动作类型：attack, heal
	Source     ActionActor  `json:"source"`      // 动作发起者
	Target     ActionActor  `json:"target"`      // 动作接收者
	Effect     ActionEffect `json:"effect"`      // 动作效果
	Message    string       `json:"message"`     // 描述信息
}

// ActionActor 动作参与者
type ActionActor struct {
	Address string `json:"address"` // 玩家地址
	Card    Card   `json:"card"`    // 卡牌信息
}

// ActionEffect 动作效果
type ActionEffect struct {
	Type  string `json:"type"`  // 效果类型：damage, heal
	Value int    `json:"value"` // 效果数值
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
		stageActions := bs.convertStageResultToActions(stageResult, player1Address, player2Address)
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
	log.Info("winnerMaxMultiplier", "player1StageMultipliers", player1StageMultipliers)
	log.Info("winnerMaxMultiplier", "player2StageMultipliers", player2StageMultipliers)
	var winnerMaxMultiplier float64 = 1.0
	if battleInfo.GameResult != "tie" {
		winnerAddress := battleInfo.GameResult
		var winnerMultipliers []float64

		if winnerAddress == player1Address {
			winnerMultipliers = player1StageMultipliers
		} else {
			winnerMultipliers = player2StageMultipliers
		}

		log.Info("winnerMultipliers", "winnerMultipliers", winnerMultipliers)

		// 找到最高倍率
		for _, multiplier := range winnerMultipliers {
			if multiplier > winnerMaxMultiplier {
				winnerMaxMultiplier = multiplier
			}
		}
	} else {
		winnerMaxMultiplier = 0.0
	}

	// 执行stage 10（最终阶段），双方都使用赢家的最高倍率
	stage10Result, err := bs.SimulateStage10(player1Address, player2Address, battleInfo.Player1.HP, battleInfo.Player2.HP, winnerMaxMultiplier)
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
	stage10Actions := bs.convertStageResultToActions(stage10Result, player1Address, player2Address)
	battleInfo.Actions = append(battleInfo.Actions, stage10Actions...)

	// stage 10肯定游戏结束，直接设置结果
	battleInfo.IsGameOver = true
	battleInfo.GameResult = stage10Result.Winner

	// 设置赢家的最终倍率
	battleInfo.WinnerMultiplier = winnerMaxMultiplier

	return battleInfo, nil
}

// convertStageResultToActions 将阶段结果转换为动作列表
func (bs *BattleSimulator) convertStageResultToActions(stageResult *StageBattleResult, player1Addr, player2Addr string) []BattleAction {
	var actions []BattleAction

	// 遍历每轮对战结果
	for round, battleResult := range stageResult.BattleResults {
		// 根据对战结果类型生成详细动作
		roundActions := bs.generateActionsFromBattleResult(round+1, battleResult, player1Addr, player2Addr)
		actions = append(actions, roundActions...)
	}

	return actions
}

// generateActionsFromBattleResult 从对战结果生成详细动作
func (bs *BattleSimulator) generateActionsFromBattleResult(round int, battleResult CardBattleResult, player1Addr, player2Addr string) []BattleAction {
	var actions []BattleAction

	switch battleResult.ResultType {
	case "ke":
		// 克：攻击对方两次
		damage := battleResult.Player1Card.Attack - battleResult.Player2Card.Defense
		if damage < 0 {
			damage = 0
		}

		// 第一次攻击
		action1 := BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Target: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Effect: ActionEffect{
				Type:  "damage",
				Value: damage,
			},
			Message: fmt.Sprintf("%s克%s，第一次攻击", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action1)

		// 第二次攻击
		action2 := BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Target: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Effect: ActionEffect{
				Type:  "damage",
				Value: damage,
			},
			Message: fmt.Sprintf("%s克%s，第二次攻击", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action2)

	case "beike":
		// 被克：被攻击两次
		damage := battleResult.Player2Card.Attack - battleResult.Player1Card.Defense
		if damage < 0 {
			damage = 0
		}

		// 第一次被攻击
		action1 := BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Target: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Effect: ActionEffect{
				Type:  "damage",
				Value: damage,
			},
			Message: fmt.Sprintf("%s被%s克，第一次被攻击", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action1)

		// 第二次被攻击
		action2 := BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Target: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Effect: ActionEffect{
				Type:  "damage",
				Value: damage,
			},
			Message: fmt.Sprintf("%s被%s克，第二次被攻击", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action2)

	case "sheng":
		// 生：给对方回血
		healAmount := battleResult.Player1Card.LifeForce
		action := BattleAction{
			Round:      round,
			ActionType: "heal",
			Source: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Target: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Effect: ActionEffect{
				Type:  "heal",
				Value: healAmount,
			},
			Message: fmt.Sprintf("%s生%s，治疗%d点血量", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol, healAmount),
		}
		actions = append(actions, action)

	case "beisheng":
		// 被生：自己回血
		healAmount := battleResult.Player2Card.LifeForce
		action := BattleAction{
			Round:      round,
			ActionType: "heal",
			Source: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Target: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Effect: ActionEffect{
				Type:  "heal",
				Value: healAmount,
			},
			Message: fmt.Sprintf("%s被%s生，治疗%d点血量", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol, healAmount),
		}
		actions = append(actions, action)

	case "ping":
		// 平：双方各攻击对方一次（拆解为两个独立动作）
		damage1 := battleResult.Player1Card.Attack - battleResult.Player2Card.Defense
		if damage1 < 0 {
			damage1 = 0
		}
		damage2 := battleResult.Player2Card.Attack - battleResult.Player1Card.Defense
		if damage2 < 0 {
			damage2 = 0
		}

		// 玩家1攻击玩家2
		action1 := BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Target: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Effect: ActionEffect{
				Type:  "damage",
				Value: damage1,
			},
			Message: fmt.Sprintf("无生克关系，%s攻击%s", battleResult.Player1Card.Symbol, battleResult.Player2Card.Symbol),
		}
		actions = append(actions, action1)

		// 玩家2攻击玩家1
		action2 := BattleAction{
			Round:      round,
			ActionType: "attack",
			Source: ActionActor{
				Address: player2Addr,
				Card:    battleResult.Player2Card,
			},
			Target: ActionActor{
				Address: player1Addr,
				Card:    battleResult.Player1Card,
			},
			Effect: ActionEffect{
				Type:  "damage",
				Value: damage2,
			},
			Message: fmt.Sprintf("无生克关系，%s攻击%s", battleResult.Player2Card.Symbol, battleResult.Player1Card.Symbol),
		}
		actions = append(actions, action2)
	}

	return actions
}
