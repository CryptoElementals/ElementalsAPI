package battle

import "fmt"

// GameLogic game logic
type GameLogic struct{}

// NewGameLogic create a new game logic
func NewGameLogic() *GameLogic {
	return &GameLogic{}
}

// CheckGameOver check if game is over
func (gl *GameLogic) CheckGameOver(player1HP, player2HP int, player1Address, player2Address string) (bool, string) {
	if player1HP <= 0 {
		return true, player2Address
	}
	if player2HP <= 0 {
		return true, player1Address
	}
	return false, ""
}

// ValidateBattleInput validate battle input
func (gl *GameLogic) ValidateBattleInput(input *BattleInput) error {
	if input.Player1Address == "" || input.Player2Address == "" {
		return fmt.Errorf("player address cannot be empty")
	}

	if input.Player1HP <= 0 || input.Player2HP <= 0 {
		return fmt.Errorf("player HP must be greater than 0")
	}

	if len(input.Player1Cards) != 3 || len(input.Player2Cards) != 3 {
		return fmt.Errorf("each player must have 3 cards")
	}

	if err := gl.validateCardElements(input.Player1Cards, "Player 1"); err != nil {
		return err
	}
	if err := gl.validateCardElements(input.Player2Cards, "Player 2"); err != nil {
		return err
	}

	return nil
}

// validateCardElements validate card element types
func (gl *GameLogic) validateCardElements(cardIDs []int, playerName string) error {
	cardFactory := NewCardFactory()
	elementTypes := make(map[string]bool)
	cardIDSet := make(map[int]bool)
	validElements := map[string]bool{
		"Metal": true,
		"Wood":  true,
		"Water": true,
		"Fire":  true,
		"Earth": true,
	}

	for i, cardID := range cardIDs {
		if cardIDSet[cardID] {
			return fmt.Errorf("%s card ID duplicated: %d", playerName, cardID)
		}
		cardIDSet[cardID] = true

		card, err := cardFactory.GetCard(cardID)
		if err != nil {
			return fmt.Errorf("%s %dth card failed to get: %v", playerName, i+1, err)
		}

		if !validElements[card.ElementType] {
			return fmt.Errorf("%s %dth card element type invalid: %s", playerName, i+1, card.ElementType)
		}

		if elementTypes[card.ElementType] {
			return fmt.Errorf("%s card element type duplicated: %s", playerName, card.ElementType)
		}

		elementTypes[card.ElementType] = true
	}

	if len(elementTypes) != 3 {
		return fmt.Errorf("%s cards must contain 3 different element types", playerName)
	}

	return nil
}

// GetWinner determine winner based on health
func (gl *GameLogic) GetWinner(player1HP, player2HP int, player1Address, player2Address string) string {
	if player1HP > player2HP {
		return player1Address
	} else if player2HP > player1HP {
		return player2Address
	} else {
		return "tie"
	}
}
