package game

import (
	"fmt"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

// round_battle_execute: per-turn card resolution, HP/multiplier updates, and elemental effects.

// executeCardIndex runs the battle for the current card slot between the two players.
// Returns (isGameOver, gameResult, error).
func (r *round) executeCardIndex() (bool, *dao.GameResult, error) {
	if err := r.validateRound(); err != nil {
		return false, nil, err
	}
	if r.isServerTimeout() {
		return r.handleServerTimeout()
	}
	isGameOver, gameResult := r.checkGameOverFromGamePlayersPreExecution()
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

	isGameOver, gameResult = r.checkGameOverFromGamePlayersPostExecution()
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
	effects1 := r.buildPlayerEffects(relation.P1Type, card1, card2, p1.player.PlayerId, p1.player.TemporaryAddress, p2.player.PlayerId)
	effects2 := r.buildPlayerEffects(relation.P2Type, card2, card1, p2.player.PlayerId, p2.player.TemporaryAddress, p1.player.PlayerId)

	p1BeforeHP := p1.currentHP
	p2BeforeHP := p2.currentHP
	p1BeforeMul := p1.multiplier
	p2BeforeMul := p2.multiplier

	r.updatePlayerHPAndLostHP(p1, p1BeforeHP, r.executeEffects(effects1))
	r.updatePlayerHPAndLostHP(p2, p2BeforeHP, r.executeEffects(effects2))

	r.updateCardStats(p1.getLastSubmittedCard(), int(p1BeforeHP), int(p1.currentHP), p1BeforeMul, p1.multiplier, relation.P1Type, relation.P1Description, effects1)
	r.updateCardStats(p2.getLastSubmittedCard(), int(p2BeforeHP), int(p2.currentHP), p2BeforeMul, p2.multiplier, relation.P2Type, relation.P2Description, effects2)
}

func (r *round) updatePlayerHPAndLostHP(player *gamePlayer, beforeHP int64, hpDelta int) {
	player.currentHP += int64(hpDelta)
	if player.currentHP < 0 {
		player.currentHP = 0
	}

	damage := max(int(beforeHP)-int(player.currentHP), 0)

	player.totalLostHP += int64(damage)
	if player.totalLostHP > config.GameParams.InitialHP {
		player.totalLostHP = config.GameParams.InitialHP
	}

	player.multiplier = r.calculateMultiplierByLostHP(int(player.totalLostHP))
}

func (r *round) updateCardStats(card *dao.TurnSubmittedCard, beforeHP, afterHP int, beforeMul, afterMul uint32, relationType, description string, effects []*dao.CardEffect) {
	card.HealthBefore = uint32(beforeHP)
	card.HealthAfter = uint32(afterHP)
	card.MultiplierBefore = beforeMul
	card.MultiplierAfter = afterMul
	card.Description = description
	card.ElementRelation = r.mapElementRelationStringToEnum(relationType)
	card.CardEffects = effects
}

func (r *round) getCard(cardID int) (*dao.Card, error) {
	dbCard, err := db.GetCardByID(cardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get card [ID:%d]: %v", cardID, err)
	}
	return dbCard, nil
}

func (r *round) calculateMultiplierByLostHP(lostHP int) uint32 {
	if lostHP <= 2000 {
		return 1
	}
	excessHP := lostHP - 2000
	bonusMultiplier := uint32(excessHP) / 500
	newMultiplier := min(1+bonusMultiplier, 9)
	return newMultiplier
}

func (r *round) buildPlayerEffects(playerType string, selfCard, opponentCard *dao.Card, selfPlayerId int64, selfTemp string, opponentPlayerId int64) []*dao.CardEffect {
	var effects []*dao.CardEffect

	desc := func(action string) string {
		return fmt.Sprintf("%s(%d) %s %s(%d)", selfCard.ElementType, selfPlayerId, action, opponentCard.ElementType, opponentPlayerId)
	}

	switch playerType {
	case RelationOverpower:
	case RelationOverpowered:
		attackValue := opponentCard.Attack - selfCard.Defense
		for _, action := range []string{ActionAttackedBy, ActionDoubleAttackedBy} {
			effects = append(effects, &dao.CardEffect{
				Type:                   proto.BattleEffectType_ATTACK,
				Value:                  int32(attackValue),
				Description:            desc(action),
				TargetPlayerId:         selfPlayerId,
				TargetTemporaryAddress: selfTemp,
			})
		}
	case RelationNurture:
	case RelationNurtured:
		effects = append(effects, &dao.CardEffect{
			Type:                   proto.BattleEffectType_HEAL,
			Value:                  int32(selfCard.LifeForce),
			Description:            desc(ActionHealedBy),
			TargetPlayerId:         selfPlayerId,
			TargetTemporaryAddress: selfTemp,
		})
	case RelationEven:
		effects = append(effects, &dao.CardEffect{
			Type:                   proto.BattleEffectType_ATTACK,
			Value:                  int32(opponentCard.Attack - selfCard.Defense),
			Description:            desc(ActionAttackedBy),
			TargetPlayerId:         selfPlayerId,
			TargetTemporaryAddress: selfTemp,
		})
	}

	return effects
}

func (r *round) executeEffects(effects []*dao.CardEffect) int {
	hpDelta := 0
	for _, effect := range effects {
		value := max(int(effect.Value), 0)
		switch effect.Type {
		case proto.BattleEffectType_ATTACK:
			hpDelta -= value
		case proto.BattleEffectType_HEAL:
			hpDelta += value
		}
	}
	return hpDelta
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
