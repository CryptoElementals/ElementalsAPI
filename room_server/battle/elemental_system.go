package battle

import (
	"fmt"
)

// ElementalSystem elemental system
type ElementalSystem struct{}

// NewElementalSystem create a new elemental system
func NewElementalSystem() *ElementalSystem {
	return &ElementalSystem{}
}

// GetElementalRelation get elemental relation
func (es *ElementalSystem) GetElementalRelation(card1, card2 *Card, wallet1, wallet2 string) *ElementalRelation {
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

	// 截取钱包地址用于显示
	addr1 := truncateAddress(wallet1)
	addr2 := truncateAddress(wallet2)

	if keRelations[card1.ElementType] == card2.ElementType {
		return &ElementalRelation{
			P1Type:        "overpower",
			P2Type:        "overpowered",
			P1Description: fmt.Sprintf("%s(%s) overpowers %s(%s)", card1.ElementType, addr1, card2.ElementType, addr2),
			P2Description: fmt.Sprintf("%s(%s) is overpowered by %s(%s)", card2.ElementType, addr2, card1.ElementType, addr1),
		}
	}
	if keRelations[card2.ElementType] == card1.ElementType {
		return &ElementalRelation{
			P1Type:        "overpowered",
			P2Type:        "overpower",
			P1Description: fmt.Sprintf("%s(%s) is overpowered by %s(%s)", card1.ElementType, addr1, card2.ElementType, addr2),
			P2Description: fmt.Sprintf("%s(%s) overpowers %s(%s)", card2.ElementType, addr2, card1.ElementType, addr1),
		}
	}

	if shengRelations[card1.ElementType] == card2.ElementType {
		return &ElementalRelation{
			P1Type:        "nurture",
			P2Type:        "nurtured",
			P1Description: fmt.Sprintf("%s(%s) nurtures %s(%s)", card1.ElementType, addr1, card2.ElementType, addr2),
			P2Description: fmt.Sprintf("%s(%s) is nurtured by %s(%s)", card2.ElementType, addr2, card1.ElementType, addr1),
		}
	}
	if shengRelations[card2.ElementType] == card1.ElementType {
		return &ElementalRelation{
			P1Type:        "nurtured",
			P2Type:        "nurture",
			P1Description: fmt.Sprintf("%s(%s) is nurtured by %s(%s)", card1.ElementType, addr1, card2.ElementType, addr2),
			P2Description: fmt.Sprintf("%s(%s) nurtures %s(%s)", card2.ElementType, addr2, card1.ElementType, addr1),
		}
	}

	return &ElementalRelation{
		P1Type:        "even",
		P2Type:        "even",
		P1Description: fmt.Sprintf("%s(%s) and %s(%s) are even", card1.ElementType, addr1, card2.ElementType, addr2),
		P2Description: fmt.Sprintf("%s(%s) and %s(%s) are even", card2.ElementType, addr2, card1.ElementType, addr1),
	}
}

// BuildPlayerEffects build effects for a specific player based on their relation type
func (es *ElementalSystem) BuildPlayerEffects(playerType string, selfCard, opponentCard *Card, selfWallet, selfTemp, opponentWallet string) []BattleEffect {
	var effects []BattleEffect

	// 截取钱包地址用于显示
	selfAddr := truncateAddress(selfWallet)
	oppAddr := truncateAddress(opponentWallet)

	// 辅助函数：生成描述
	desc := func(action string) string {
		return fmt.Sprintf("%s(%s) %s %s(%s)", selfCard.ElementType, selfAddr, action, opponentCard.ElementType, oppAddr)
	}

	switch playerType {
	case "overpower":
		// overpower 不对自己造成影响，对手会被攻击
		// 这里不产生任何效果，因为效果作用在对手身上
	case "overpowered":
		// 被对手压制，自己受到两次攻击
		effects = append(effects, BattleEffect{
			Type:                   ATTACK,
			Value:                  opponentCard.Attack - selfCard.Defense,
			Description:            desc("is attacked by"),
			TargetWalletAddress:    selfWallet,
			TargetTemporaryAddress: selfTemp,
		})
		effects = append(effects, BattleEffect{
			Type:                   ATTACK,
			Value:                  opponentCard.Attack - selfCard.Defense,
			Description:            desc("is double attacked by"),
			TargetWalletAddress:    selfWallet,
			TargetTemporaryAddress: selfTemp,
		})
	case "nurture":
		// nurture 不对自己造成影响，对手会被治疗
		// 这里不产生任何效果，因为效果作用在对手身上
	case "nurtured":
		// 被对手滋养，自己获得治疗
		effects = append(effects, BattleEffect{
			Type:                   HEAL,
			Value:                  selfCard.LifeForce,
			Description:            desc("is healed by"),
			TargetWalletAddress:    selfWallet,
			TargetTemporaryAddress: selfTemp,
		})
	case "even":
		// 平手，自己受到一次攻击
		effects = append(effects, BattleEffect{
			Type:                   ATTACK,
			Value:                  opponentCard.Attack - selfCard.Defense,
			Description:            desc("is attacked by"),
			TargetWalletAddress:    selfWallet,
			TargetTemporaryAddress: selfTemp,
		})
	}

	return effects
}

// truncateAddress 截取钱包地址用于显示（前6位+...+后4位）
func truncateAddress(address string) string {
	if len(address) <= 10 {
		return address
	}
	return address[:6] + "..." + address[len(address)-4:]
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
