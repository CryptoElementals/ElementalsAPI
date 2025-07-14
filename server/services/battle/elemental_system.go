package battle

import "fmt"

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
			Description: fmt.Sprintf("%s overpowers %s", card1.ElementType, card2.ElementType),
		}
	}
	if keRelations[card2.ElementType] == card1.ElementType {
		return &ElementalRelation{
			Type:        "overpowered",
			Description: fmt.Sprintf("%s is overpowered by %s", card1.ElementType, card2.ElementType),
		}
	}

	if shengRelations[card1.ElementType] == card2.ElementType {
		return &ElementalRelation{
			Type:        "nurture",
			Description: fmt.Sprintf("%s nurtures %s", card1.ElementType, card2.ElementType),
		}
	}
	if shengRelations[card2.ElementType] == card1.ElementType {
		return &ElementalRelation{
			Type:        "nurtured",
			Description: fmt.Sprintf("%s is nurtured by %s", card1.ElementType, card2.ElementType),
		}
	}

	return &ElementalRelation{
		Type:        "even",
		Description: "Same element",
	}
}

// BuildActions build action list based on elemental relations
func (es *ElementalSystem) BuildActions(card1, card2 *Card, relation *ElementalRelation) []BattleAction {
	var actions []BattleAction

	switch relation.Type {
	case "overpower":
		actions = append(actions, BattleAction{
			Type:        "attack",
			Target:      "player2",
			Value:       card1.Attack - card2.Defense,
			Description: fmt.Sprintf("%s(Player1) attacks %s(Player2)", card1.Name, card2.Name),
		})
		actions = append(actions, BattleAction{
			Type:        "attack",
			Target:      "player2",
			Value:       card1.Attack - card2.Defense,
			Description: fmt.Sprintf("%s(Player1) attacks %s(Player2) again", card1.Name, card2.Name),
		})
	case "overpowered":
		actions = append(actions, BattleAction{
			Type:        "attack",
			Target:      "player1",
			Value:       card2.Attack - card1.Defense,
			Description: fmt.Sprintf("%s(Player2) attacks %s(Player1)", card2.Name, card1.Name),
		})
		actions = append(actions, BattleAction{
			Type:        "attack",
			Target:      "player1",
			Value:       card2.Attack - card1.Defense,
			Description: fmt.Sprintf("%s(Player2) attacks %s(Player1) again", card2.Name, card1.Name),
		})
	case "nurture":
		actions = append(actions, BattleAction{
			Type:        "heal",
			Target:      "player2",
			Value:       card1.LifeForce,
			Description: fmt.Sprintf("%s(Player1) heals %s(Player2)", card1.Name, card2.Name),
		})
	case "nurtured":
		actions = append(actions, BattleAction{
			Type:        "heal",
			Target:      "player1",
			Value:       card2.LifeForce,
			Description: fmt.Sprintf("%s(Player2) heals %s(Player1)", card2.Name, card1.Name),
		})
	case "even":
		actions = append(actions, BattleAction{
			Type:        "attack",
			Target:      "player2",
			Value:       card1.Attack - card2.Defense,
			Description: fmt.Sprintf("%s(Player1) attacks %s(Player2)", card1.Name, card2.Name),
		})
		actions = append(actions, BattleAction{
			Type:        "attack",
			Target:      "player1",
			Value:       card2.Attack - card1.Defense,
			Description: fmt.Sprintf("%s(Player2) attacks %s(Player1)", card2.Name, card1.Name),
		})
	}

	return actions
}

// ExecuteActions execute action list and calculate damage
func (es *ElementalSystem) ExecuteActions(actions []BattleAction) (int, int) {
	player1Damage := 0
	player2Damage := 0

	for _, action := range actions {
		value := action.Value
		if value < 0 {
			value = 0
		}

		switch action.Type {
		case "attack":
			if action.Target == "player1" {
				player1Damage -= value
			} else if action.Target == "player2" {
				player2Damage -= value
			}
		case "heal":
			if action.Target == "player1" {
				player1Damage += value
			} else if action.Target == "player2" {
				player2Damage += value
			}
		}
	}

	return player1Damage, player2Damage
}
