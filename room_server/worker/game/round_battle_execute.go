package game

import (
	"fmt"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

// round_battle_execute: per-turn card resolution, HP updates, and elemental relations.

// executeCardIndex runs the battle for the current card slot between the two players.
// Returns (isGameOver, gameResult, error).
func (r *round) executeCardIndex() (bool, *dao.GameResult, error) {
	if err := r.validateRound(); err != nil {
		return false, nil, err
	}
	if r.isServerTimeout() {
		return r.handleServerTimeout()
	}
	isGameOver, gameResult := r.checkGameOver(false)
	if isGameOver {
		return isGameOver, gameResult, nil
	}
	var p1 *gamePlayer
	var p2 *gamePlayer
	var card1 *dao.Card
	var card2 *dao.Card
	for _, gamePlayer := range r.gamePlayers {
		submittedCard := gamePlayer.getLastSubmittedCard()
		cardID, err := r.getCard(int(submittedCard.CardID))
		if err != nil {
			return false, nil, fmt.Errorf("failed to get card: %v", err)
		}
		if card1 == nil {
			card1 = cardID
			p1 = gamePlayer
		} else if card2 == nil {
			card2 = cardID
			p2 = gamePlayer
		}
	}

	if p1.status == playerStatusOnline && p2.status == playerStatusOnline {
		r.processCardBattle(p1, p2, card1, card2)
	}

	isGameOver, gameResult = r.checkGameOver(true)
	return isGameOver, gameResult, nil
}

func (r *round) validateRound() error {
	if r.game == nil {
		return fmt.Errorf("game is nil")
	}

	playerCount := len(r.gamePlayers)
	if playerCount < 2 {
		return fmt.Errorf("at least 2 players required")
	}

	maxR := r.maxConfiguredRounds()
	if r.roundNumber < 1 || r.roundNumber > maxR {
		return fmt.Errorf("round parameter must be between 1 and %d", maxR)
	}

	return nil
}

func (r *round) isServerTimeout() bool {
	return r.completeReason == proto.RoundCompleteReason_ROUND_COMPLETE_SERVER_CHAIN_TIMEOUT ||
		r.completeReason == proto.RoundCompleteReason_ROUND_COMPLETE_SERVER_INTERNAL_TIMEOUT
}

func (r *round) processCardBattle(p1, p2 *gamePlayer, card1, card2 *dao.Card) {
	relation := getElementalRelation(card1, card2, p1.player.PlayerId, p2.player.PlayerId)
	d1, d2 := hpDeltasForElementalRelation(relation.P1Type, card1, card2)

	p1BeforeHP := p1.currentHP
	p2BeforeHP := p2.currentHP

	r.updatePlayerHPFromEffects(p1, d1)
	r.updatePlayerHPFromEffects(p2, d2)

	r.updateCardStats(p1.getLastSubmittedCard(), int(p1BeforeHP), int(p1.currentHP), relation.P1Type, relation.P1Description)
	r.updateCardStats(p2.getLastSubmittedCard(), int(p2BeforeHP), int(p2.currentHP), relation.P2Type, relation.P2Description)
}

// hpDeltasForElementalRelation returns HP deltas for (p1, p2) from p1's relation to p2.
func hpDeltasForElementalRelation(p1Relation string, p1Card, p2Card *dao.Card) (int, int) {
	switch p1Relation {
	case RelationOverpower:
		return 0, -max(p1Card.Attack-p2Card.Defense, 0)
	case RelationOverpowered:
		return -max(p2Card.Attack-p1Card.Defense, 0), 0
	case RelationNurture:
		return 0, max(p1Card.LifeForce, 0)
	case RelationNurtured:
		return max(p2Card.LifeForce, 0), 0
	default:
		return 0, 0
	}
}

func (r *round) updatePlayerHPFromEffects(player *gamePlayer, hpDelta int) {
	player.currentHP += int64(hpDelta)
	if player.currentHP < 0 {
		player.currentHP = 0
	}
	maxHP := r.game.GameArgs.MaxHP
	if player.currentHP > maxHP {
		player.currentHP = maxHP
	}
}

func (r *round) updateCardStats(card *dao.TurnSubmittedCard, beforeHP, afterHP int, relationType, description string) {
	card.HealthBefore = uint32(beforeHP)
	card.HealthAfter = uint32(afterHP)
	card.Description = description
	card.ElementRelation = r.mapElementRelationStringToEnum(relationType)
}

func (r *round) getCard(cardID int) (*dao.Card, error) {
	dbCard, err := db.GetCardByID(cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get card [ID:%d]: %v", cardID, err)
	}
	return dbCard, nil
}

func (r *round) mapElementRelationStringToEnum(s string) proto.ElementRelation {
	switch s {
	case RelationOverpower:
		return proto.ElementRelation_OVER_POWER
	case RelationOverpowered:
		return proto.ElementRelation_OVER_POWERED
	case RelationNurture:
		return proto.ElementRelation_NURTURE
	case RelationNurtured:
		return proto.ElementRelation_NURTURED
	default:
		return proto.ElementRelation_TIE
	}
}
