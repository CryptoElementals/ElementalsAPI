package battle

import (
	"fmt"
	"strings"
)

// ElementalSystem elemental system
type ElementalSystem struct{}

// NewElementalSystem create a new elemental system
func NewElementalSystem() *ElementalSystem {
	return &ElementalSystem{}
}

// GetElementalRelation get elemental relation
func (es *ElementalSystem) GetElementalRelation(card1, card2 *Card) *ElementalRelation {
	keRelations := map[string]string{
		"Metal": "Wood",  // Metal overpowers Wood
		"Wood":  "Earth", // Wood overpowers Earth
		"Earth": "Water", // Earth overpowers Water
		"Water": "Fire",  // Water overpowers Fire
		"Fire":  "Metal", // Fire overpowers Metal
	}

	shengRelations := map[string]string{
		"Metal": "Water", // Metal nurtures Water
		"Water": "Wood",  // Water nurtures Wood
		"Wood":  "Fire",  // Wood nurtures Fire
		"Fire":  "Earth", // Fire nurtures Earth
		"Earth": "Metal", // Earth nurtures Metal
	}

	if keRelations[card1.ElementType] == card2.ElementType {
		return &ElementalRelation{
			Type:        "overpower",
			Description: "{self} overpowers {opponent}",
		}
	}
	if keRelations[card2.ElementType] == card1.ElementType {
		return &ElementalRelation{
			Type:        "overpowered",
			Description: "{self} is overpowered by {opponent}",
		}
	}

	if shengRelations[card1.ElementType] == card2.ElementType {
		return &ElementalRelation{
			Type:        "nurture",
			Description: "{self} is nurtured by {opponent}",
		}
	}
	if shengRelations[card2.ElementType] == card1.ElementType {
		return &ElementalRelation{
			Type:        "nurtured",
			Description: "{self} nurtures {opponent}",
		}
	}

	return &ElementalRelation{
		Type:        "even",
		Description: "{self} and {opponent} are even",
	}
}

// BuildEffects build effect list based on elemental relations
func (es *ElementalSystem) BuildEffects(card1, card2 *Card, relation *ElementalRelation, wallet1, wallet2, temp1, temp2 string) []BattleEffect {
	var effects []BattleEffect

	// 辅助函数：生成描述（用卡牌名字）
	desc := func(selfName, oppName, action string) string {
		if strings.Contains(action, "again") {
			// 如果action包含again，将其移到句子末尾
			baseAction := strings.Replace(action, " again", "", 1)
			return fmt.Sprintf("%s(self) %s %s(opponent) again", selfName, baseAction, oppName)
		}
		return fmt.Sprintf("%s(self) %s %s(opponent)", selfName, action, oppName)
	}

	switch relation.Type {
	case "overpower":
		effects = append(effects, BattleEffect{
			Type:                   ATTACK,
			Value:                  card1.Attack - card2.Defense,
			Description:            desc(card2.Name, card1.Name, "is attacked by"),
			TargetWalletAddress:    wallet2,
			TargetTemporaryAddress: temp2,
		})
		effects = append(effects, BattleEffect{
			Type:                   ATTACK,
			Value:                  card1.Attack - card2.Defense,
			Description:            desc(card2.Name, card1.Name, "is attacked by again"),
			TargetWalletAddress:    wallet2,
			TargetTemporaryAddress: temp2,
		})
	case "overpowered":
		effects = append(effects, BattleEffect{
			Type:                   ATTACK,
			Value:                  card2.Attack - card1.Defense,
			Description:            desc(card1.Name, card2.Name, "is attacked by"),
			TargetWalletAddress:    wallet1,
			TargetTemporaryAddress: temp1,
		})
		effects = append(effects, BattleEffect{
			Type:                   ATTACK,
			Value:                  card2.Attack - card1.Defense,
			Description:            desc(card1.Name, card2.Name, "is attacked by again"),
			TargetWalletAddress:    wallet1,
			TargetTemporaryAddress: temp1,
		})
	case "nurture":
		effects = append(effects, BattleEffect{
			Type:                   HEAL,
			Value:                  card1.LifeForce,
			Description:            desc(card1.Name, card2.Name, "is healed by"),
			TargetWalletAddress:    wallet1,
			TargetTemporaryAddress: temp1,
		})
	case "nurtured":
		effects = append(effects, BattleEffect{
			Type:                   HEAL,
			Value:                  card2.LifeForce,
			Description:            desc(card1.Name, card2.Name, "is healed by"),
			TargetWalletAddress:    wallet1,
			TargetTemporaryAddress: temp1,
		})
	case "even":
		effects = append(effects, BattleEffect{
			Type:                   ATTACK,
			Value:                  card1.Attack - card2.Defense,
			Description:            desc(card2.Name, card1.Name, "is attacked by"),
			TargetWalletAddress:    wallet2,
			TargetTemporaryAddress: temp2,
		})
		effects = append(effects, BattleEffect{
			Type:                   ATTACK,
			Value:                  card2.Attack - card1.Defense,
			Description:            desc(card1.Name, card2.Name, "is attacked by"),
			TargetWalletAddress:    wallet1,
			TargetTemporaryAddress: temp1,
		})
	}

	return effects
}

// ExecuteEffects execute effect list and calculate HP delta for self
func (es *ElementalSystem) ExecuteEffects(effects []BattleEffect) int {
	hpDelta := 0
	for _, effect := range effects {
		value := effect.Value
		if value < 0 {
			value = 0
		}
		switch effect.Type {
		case ATTACK:
			hpDelta -= value
		case HEAL:
			hpDelta += value
		}
	}
	return hpDelta
}
