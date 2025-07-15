package battle

import (
	"fmt"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
)

// CardFactory card factory
type CardFactory struct{}

// NewCardFactory create a new card factory
func NewCardFactory() *CardFactory {
	return &CardFactory{}
}

// GetCard get card information by card ID
func (cf *CardFactory) GetCard(cardID int) (*Card, error) {

	card, err := db.GetCardByID(cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get card information [ID:%d]: %v", cardID, err)
	}

	return cf.convertToBattleCard(card), nil
}

// GetCards get card information in batches by card ID list
func (cf *CardFactory) GetCards(cardIDs []int) ([]*Card, error) {
	cards := make([]*Card, 0, len(cardIDs))

	for i, cardID := range cardIDs {
		card, err := cf.GetCard(cardID)
		if err != nil {
			return nil, fmt.Errorf("failed to get card information [index:%d, ID:%d]: %v", i, cardID, err)
		}
		cards = append(cards, card)
	}

	return cards, nil
}

// convertToBattleCard convert database card to battle card
func (cf *CardFactory) convertToBattleCard(dbCard *dao.Card) *Card {
	return &Card{
		ID:          dbCard.CardID,
		ElementType: dbCard.ElementType,
		Level:       dbCard.Level,
		LifeForce:   dbCard.LifeForce,
		Attack:      dbCard.Attack,
		Defense:     dbCard.Defense,
		Name:        dbCard.Name,
		Description: dbCard.Description,
	}
}
