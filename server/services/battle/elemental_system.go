package battle

import "fmt"

// ElementalSystem 五行相生相克系统
type ElementalSystem struct{}

// NewElementalSystem 创建新的五行系统
func NewElementalSystem() *ElementalSystem {
	return &ElementalSystem{}
}

// DetermineBattleRelation 判断五行关系
func (es *ElementalSystem) DetermineBattleRelation(card1, card2 Card) (string, string) {
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

// GetMovementByRelation 根据五行关系获取对应的动作
func (es *ElementalSystem) GetMovementByRelation(resultType string) Movement {
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

// BattleCards 两张卡牌对战（五行相生相克）
func (es *ElementalSystem) BattleCards(card1, card2 Card, player1Addr, player2Addr string) CardBattleResult {
	result := CardBattleResult{
		Player1Card: card1,
		Player2Card: card2,
		EffectValue: []int{0, 0}, // 初始化效果值数组
	}

	// 判断五行关系
	resultType, baseReason := es.DetermineBattleRelation(card1, card2)
	result.ResultType = resultType

	// 根据关系类型获取对应的动作
	movement := es.GetMovementByRelation(resultType)

	// 执行动作
	effectValue, movementReason := movement.Execute(card1, card2)
	result.EffectValue = effectValue

	// 生成详细的原因描述
	result.Reason = baseReason + "，" + movementReason

	return result
}
