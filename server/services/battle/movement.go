package battle

import "fmt"

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
