package game

import (
	"fmt"

	dao "github.com/CryptoElementals/common/models"
)

// Element type constants
const (
	ElementFire  = "Fire"
	ElementWater = "Water"
	ElementEarth = "Earth"
	ElementMetal = "Metal"
	ElementWood  = "Wood"
)

// Element relation type constants
const (
	RelationOverpower   = "overpower"
	RelationOverpowered = "overpowered"
	RelationNurture     = "nurture"
	RelationNurtured    = "nurtured"
	RelationEven        = "even"
)

// ElementalRelation represents the relationship between two cards
type elementalRelation struct {
	P1Type        string
	P2Type        string
	P1Description string
	P2Description string
}

// Ke (Overpower) relations - Metal > Wood > Earth > Water > Fire > Metal
var keRelations = map[string]string{
	ElementMetal: ElementWood,
	ElementWood:  ElementEarth,
	ElementEarth: ElementWater,
	ElementWater: ElementFire,
	ElementFire:  ElementMetal,
}

// Sheng (Nurture) relations - Metal nurtures Water, Water nurtures Wood, etc.
var shengRelations = map[string]string{
	ElementMetal: ElementWater,
	ElementWater: ElementWood,
	ElementWood:  ElementFire,
	ElementFire:  ElementEarth,
	ElementEarth: ElementMetal,
}

// getElementalRelation determines the elemental relation between two cards
func getElementalRelation(card1, card2 *dao.Card, playerId1, playerId2 int64) *elementalRelation {
	// Check Ke (overpower) relations
	if target, hasRelation := keRelations[card1.ElementType]; hasRelation && target == card2.ElementType {
		return buildRelation(RelationOverpower, RelationOverpowered, card1.ElementType, playerId1, card2.ElementType, playerId2)
	}
	if target, hasRelation := keRelations[card2.ElementType]; hasRelation && target == card1.ElementType {
		return buildRelation(RelationOverpowered, RelationOverpower, card1.ElementType, playerId1, card2.ElementType, playerId2)
	}

	// Check Sheng (nurture) relations
	if target, hasRelation := shengRelations[card1.ElementType]; hasRelation && target == card2.ElementType {
		return buildRelation(RelationNurture, RelationNurtured, card1.ElementType, playerId1, card2.ElementType, playerId2)
	}
	if target, hasRelation := shengRelations[card2.ElementType]; hasRelation && target == card1.ElementType {
		return buildRelation(RelationNurtured, RelationNurture, card1.ElementType, playerId1, card2.ElementType, playerId2)
	}

	// Even (tie)
	return &elementalRelation{
		P1Type:        RelationEven,
		P2Type:        RelationEven,
		P1Description: fmt.Sprintf("%s(%d) and %s(%d) are even", card1.ElementType, playerId1, card2.ElementType, playerId2),
		P2Description: fmt.Sprintf("%s(%d) and %s(%d) are even", card2.ElementType, playerId2, card1.ElementType, playerId1),
	}
}

// buildRelation builds an elemental relation structure
func buildRelation(p1Type, p2Type string, elem1 string, playerId1 int64, elem2 string, playerId2 int64) *elementalRelation {
	var p1Desc, p2Desc string
	switch p1Type {
	case RelationOverpower:
		p1Desc = fmt.Sprintf("%s(%d) overpowers %s(%d)", elem1, playerId1, elem2, playerId2)
		p2Desc = fmt.Sprintf("%s(%d) is overpowered by %s(%d)", elem2, playerId2, elem1, playerId1)
	case RelationOverpowered:
		p1Desc = fmt.Sprintf("%s(%d) is overpowered by %s(%d)", elem1, playerId1, elem2, playerId2)
		p2Desc = fmt.Sprintf("%s(%d) overpowers %s(%d)", elem2, playerId2, elem1, playerId1)
	case RelationNurture:
		p1Desc = fmt.Sprintf("%s(%d) nurtures %s(%d)", elem1, playerId1, elem2, playerId2)
		p2Desc = fmt.Sprintf("%s(%d) is nurtured by %s(%d)", elem2, playerId2, elem1, playerId1)
	case RelationNurtured:
		p1Desc = fmt.Sprintf("%s(%d) is nurtured by %s(%d)", elem1, playerId1, elem2, playerId2)
		p2Desc = fmt.Sprintf("%s(%d) nurtures %s(%d)", elem2, playerId2, elem1, playerId1)
	}
	return &elementalRelation{
		P1Type:        p1Type,
		P2Type:        p2Type,
		P1Description: p1Desc,
		P2Description: p2Desc,
	}
}
