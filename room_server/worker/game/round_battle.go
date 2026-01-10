package game

import (
	"fmt"
	"slices"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

// GameResultType represents the type of game result
type gameResultType int32

const (
	gameResultNormal gameResultType = iota
	gameResultKO
	gameResultTie
)

// Battle action description constants
const (
	ActionAttackedBy       = "is attacked by"
	ActionDoubleAttackedBy = "is double attacked by"
	ActionHealedBy         = "is healed by"
)

// GameEndState represents a player's state for game end calculation
type gameEndState struct {
	HP               int
	Multiplier       uint32
	PlayerId         int64
	TemporaryAddress string
	Status           playerStatus
}

// executeCardIndex executes battles for a single card index (0, 1, or 2) between all players
// It initializes battle states if needed and processes all player pairs for the given card index
// Returns (isGameOver, gameResult, error)
func (r *round) executeCardIndex() (bool, *dao.GameResult, error) {
	if err := r.validateRound(); err != nil {
		return false, nil, err
	}
	if r.isServerTimeout() {
		return r.handleServerTimeout()
	}
	// Check if game is over (before card execution - don't check round/turn limits yet)
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

	// Check if game is over (after card execution - now we can check round/turn limits)
	isGameOver, gameResult = r.checkGameOverFromGamePlayersPostExecution()
	return isGameOver, gameResult, nil
}

// validateRound validates the round data
func (r *round) validateRound() error {
	if r.round == nil {
		return fmt.Errorf("round is nil")
	}

	playerCount := len(r.gamePlayers)
	if playerCount < 2 {
		return fmt.Errorf("at least 2 players required")
	}

	if r.round.RoundNumber < 1 || r.round.RoundNumber > 3 {
		return fmt.Errorf("round parameter must be between 1 and 3")
	}

	return nil
}

// isServerTimeout checks if round ended due to server timeout
func (r *round) isServerTimeout() bool {
	return r.round.CompleteReason == proto.RoundCompleteReason_ROUND_COMPLETE_SERVER_CHAIN_TIMEOUT ||
		r.round.CompleteReason == proto.RoundCompleteReason_ROUND_COMPLETE_SERVER_INTERNAL_TIMEOUT
}

// processCardBattle processes a single card battle between two players
func (r *round) processCardBattle(p1, p2 *gamePlayer, card1, card2 *dao.Card) {
	// Get elemental relation
	relation := getElementalRelation(card1, card2, p1.player.PlayerId, p2.player.PlayerId)
	effects1 := r.buildPlayerEffects(relation.P1Type, card1, card2, p1.player.PlayerId, p1.player.TemporaryAddress, p2.player.PlayerId)
	effects2 := r.buildPlayerEffects(relation.P2Type, card2, card1, p2.player.PlayerId, p2.player.TemporaryAddress, p1.player.PlayerId)

	// Record initial state
	p1BeforeHP := p1.currentHP
	p2BeforeHP := p2.currentHP
	p1BeforeMul := p1.multiplier
	p2BeforeMul := p2.multiplier

	// Execute effects and update state
	r.updatePlayerHPAndLostHP(p1, p1BeforeHP, r.executeEffects(effects1))
	r.updatePlayerHPAndLostHP(p2, p2BeforeHP, r.executeEffects(effects2))

	r.updateCardStats(p1.getLastSubmittedCard(), int(p1BeforeHP), int(p1.currentHP), p1BeforeMul, p1.multiplier, relation.P1Type, relation.P1Description, effects1)
	r.updateCardStats(p2.getLastSubmittedCard(), int(p2BeforeHP), int(p2.currentHP), p2BeforeMul, p2.multiplier, relation.P2Type, relation.P2Description, effects2)
}

// ===== Helper Functions =====

// updatePlayerHPAndLostHP calculates HP changes and updates LostHP and Multiplier
func (r *round) updatePlayerHPAndLostHP(player *gamePlayer, beforeHP int64, hpDelta int) {
	// Apply delta and clamp HP to >= 0
	player.currentHP += int64(hpDelta)
	if player.currentHP < 0 {
		player.currentHP = 0
	}

	// Compute non-negative damage
	damage := max(int(beforeHP)-int(player.currentHP), 0)

	// Accumulate LostHP and clamp to initial HP cap
	player.totalLostHP += int64(damage)
	if player.totalLostHP > config.GameParams.InitialHP {
		player.totalLostHP = config.GameParams.InitialHP
	}

	// Recompute multiplier from total lost HP
	player.multiplier = r.calculateMultiplierByLostHP(int(player.totalLostHP))
}

// buildGameEndStates converts game players to game end states
func (r *round) buildGameEndStates() []*gameEndState {
	playerCount := len(r.gamePlayers)
	gameEndStates := make([]*gameEndState, 0, playerCount)
	for _, player := range r.gamePlayers {
		// Skip if player has already surrendered
		if player.status == playerStatusSurrendered {
			gameEndStates = append(gameEndStates, &gameEndState{
				HP:               int(player.currentHP),
				Multiplier:       player.multiplier,
				PlayerId:         player.player.PlayerId,
				TemporaryAddress: player.player.TemporaryAddress,
				Status:           player.status,
			})
			continue
		}

		submittedCard := player.getLastSubmittedCard()
		hasSubmittedCommitment := submittedCard != nil && len(submittedCard.CommitmentHash) > 0
		hasSubmittedCard := submittedCard != nil && submittedCard.CardID > 0

		// Mark player as offline based on turn status and what they've submitted
		// If turn is waiting for commitments and player has no commitment, mark as offline
		// If turn is waiting for cards and player has no card, mark as offline
		switch r.turnStatus {
		case proto.TurnStatus_TURN_WAITTING_COMMITMENTS:
			// Turn is waiting for commitments - check if player has submitted commitment
			if !hasSubmittedCommitment {
				player.status = playerStatusOffline
			}
		case proto.TurnStatus_TURN_WAITTING_CARDS:
			// Turn is waiting for cards - check if player has submitted card
			// Note: A card submission implies a commitment was submitted, so we only check for card
			if !hasSubmittedCard {
				player.status = playerStatusOffline
			}
		}

		gameEndStates = append(gameEndStates, &gameEndState{
			HP:               int(player.currentHP),
			Multiplier:       player.multiplier,
			PlayerId:         player.player.PlayerId,
			TemporaryAddress: player.player.TemporaryAddress,
			Status:           player.status,
		})
	}
	return gameEndStates
}

// buildGameResult builds a GameResult from the checkGameOver results
func (r *round) buildGameResult(grType gameResultType, winnerPlayerId int64, temporaryAddress string, finalMul uint32) *dao.GameResult {
	// Build player status map
	playerStatuses := make(map[string]playerStatus)
	for _, player := range r.gamePlayers {
		playerStatuses[player.player.TemporaryAddress] = player.status
	}

	// Build battle reward using game players
	battleReward := r.calculateBattleRewardFromGamePlayers(r.gamePlayers, grType, winnerPlayerId, temporaryAddress, finalMul, playerStatuses)

	return &dao.GameResult{
		GameID:                 r.round.GameID,
		Multiplier:             int32(finalMul),
		WinnerPlayerId:         winnerPlayerId,
		WinnerTemporaryAddress: temporaryAddress,
		GameResultType:         proto.GameResultType(grType),
		BattleReward:           battleReward,
	}
}

// checkGameOverFromGamePlayersPreExecution checks if game is over before card execution
// This should NOT check round/turn limits to allow the last turn to execute
func (r *round) checkGameOverFromGamePlayersPreExecution() (bool, *dao.GameResult) {
	gameEndStates := r.buildGameEndStates()

	// Check if game is over (without checking round/turn limits)
	isGameOver, grType, winnerPlayerId, temporaryAddress, finalMul := r.checkGameOver(gameEndStates, r.round.RoundNumber, false)

	// Build game result if game is over
	var gameResult *dao.GameResult
	if isGameOver {
		gameResult = r.buildGameResult(grType, winnerPlayerId, temporaryAddress, finalMul)
	}

	return isGameOver, gameResult
}

// checkGameOverFromGamePlayersPostExecution checks if game is over after card execution
// This CAN check round/turn limits since cards have been executed
func (r *round) checkGameOverFromGamePlayersPostExecution() (bool, *dao.GameResult) {
	gameEndStates := r.buildGameEndStates()

	// Check if game is over (with round/turn limit checking)
	isGameOver, grType, winnerPlayerId, temporaryAddress, finalMul := r.checkGameOver(gameEndStates, r.round.RoundNumber, true)

	// Build game result if game is over
	var gameResult *dao.GameResult
	if isGameOver {
		gameResult = r.buildGameResult(grType, winnerPlayerId, temporaryAddress, finalMul)
	}

	return isGameOver, gameResult
}

// updateCardStats updates the card stats in the DAO structure
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
		// No effects
	case RelationOverpowered:
		attackValue := opponentCard.Attack - selfCard.Defense
		// Two attacks for overpowered
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
		// No effects
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

// isGameEndsByRoundAndTurn checks if the game ends by reaching the final round and turn
func (r *round) isGameEndsByRoundAndTurn() bool {
	return r.round.RoundNumber == 3 && r.getCurrentTurnNumber() == 3
}

func (r *round) checkGameOver(states []*gameEndState, round uint32, checkRoundTurnLimit bool) (bool, gameResultType, int64, string, uint32) {
	// Count surrendered players
	surrenderedCount := 0
	maxLoserMul := uint32(1)

	for _, state := range states {
		if state.Status == playerStatusSurrendered {
			surrenderedCount++
			if state.Multiplier > maxLoserMul {
				maxLoserMul = state.Multiplier
			}
		}
	}

	if surrenderedCount > 0 {
		if surrenderedCount == len(states) {
			return true, gameResultTie, 0, "", 1
		}
		// Find the first non-surrendered player as winner
		for _, state := range states {
			if state.Status != playerStatusSurrendered {
				return true, gameResultKO, state.PlayerId, state.TemporaryAddress, maxLoserMul
			}
		}
	}

	// Check offline players
	offlineCount := 0
	var offlineMaxMul uint32 = 1
	for _, state := range states {
		if state.Status == playerStatusOffline {
			offlineCount++
			if state.Multiplier > offlineMaxMul {
				offlineMaxMul = state.Multiplier
			}
		}
	}

	if offlineCount > 0 {
		if offlineCount == len(states) {
			return true, gameResultTie, 0, "", 1
		}
		// Find the first online player as winner
		for _, state := range states {
			if state.Status == playerStatusOnline {
				return true, gameResultNormal, state.PlayerId, state.TemporaryAddress, offlineMaxMul
			}
		}
	}

	// Check by HP
	return r.checkGameOverByHP(states, round, false, checkRoundTurnLimit)
}

func (r *round) checkGameOverByHP(states []*gameEndState, round uint32, hasOffline bool, checkRoundTurnLimit bool) (bool, gameResultType, int64, string, uint32) {
	hps := make([]int, len(states))
	multipliers := make([]uint32, len(states))

	for i, state := range states {
		hps[i] = state.HP
		multipliers[i] = state.Multiplier
	}

	allSameHP := true
	firstHP := hps[0]
	for _, hp := range hps[1:] {
		if hp != firstHP {
			allSameHP = false
			break
		}
	}

	if allSameHP {
		if firstHP == 0 {
			return true, gameResultTie, 0, "", 1
		}
		if hasOffline || (checkRoundTurnLimit && r.isGameEndsByRoundAndTurn()) {
			return true, gameResultTie, 0, "", 1
		}
		return false, gameResultNormal, 0, "", 1
	}

	hasZeroHP := slices.Contains(hps, 0)

	var winnerPlayerId int64
	var winnerTemp string
	var gType gameResultType
	var finalMultiplier uint32 = 1

	if hasZeroHP {
		gType = gameResultKO
		maxLoserMul := uint32(1)
		// Find the first player with HP > 0 as winner
		for i, state := range states {
			if hps[i] > 0 {
				winnerPlayerId = state.PlayerId
				winnerTemp = state.TemporaryAddress
			} else {
				if multipliers[i] > maxLoserMul {
					maxLoserMul = multipliers[i]
				}
			}
		}
		finalMultiplier = maxLoserMul
	} else {
		if !hasOffline && !(checkRoundTurnLimit && r.isGameEndsByRoundAndTurn()) {
			return false, gameResultNormal, 0, "", 1
		}

		gType = gameResultNormal
		maxHP := -1
		for _, hp := range hps {
			if hp > maxHP {
				maxHP = hp
			}
		}

		maxLoserMul := uint32(1)
		// Find the first player with max HP as winner
		for i, state := range states {
			if hps[i] == maxHP {
				winnerPlayerId = state.PlayerId
				winnerTemp = state.TemporaryAddress
			} else {
				if multipliers[i] > maxLoserMul {
					maxLoserMul = multipliers[i]
				}
			}
		}
		finalMultiplier = maxLoserMul
	}

	return true, gType, winnerPlayerId, winnerTemp, finalMultiplier
}

func (r *round) calculateBattleRewardFromGamePlayers(players map[string]*gamePlayer, grType gameResultType, winnerPlayerId int64, temporaryAddress string, finalMul uint32, playerStatuses map[string]playerStatus) *dao.BattleReward {
	baseStake := config.GameParams.BaseStake
	var playerRewards []*dao.PlayerReward
	var systemFee int

	switch grType {
	case gameResultTie:
		tokenDeduction := int(float64(baseStake) * 0.008)
		pointGain := int(float64(baseStake) * 0.008)
		playerRewards = make([]*dao.PlayerReward, 0, len(players))

		for _, player := range players {
			status := playerStatuses[player.player.TemporaryAddress]
			playerRewards = append(playerRewards, &dao.PlayerReward{
				PlayerId:               player.player.PlayerId,
				TemporaryAddress:       player.player.TemporaryAddress,
				TokenChange:            int32(-tokenDeduction),
				PointChange:            int32(pointGain),
				IsOffline:              status == playerStatusOffline,
				Surrendered:            status == playerStatusSurrendered,
				PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE,
			})
		}
		systemFee = tokenDeduction * len(players)

	case gameResultNormal, gameResultKO:
		if winnerPlayerId != 0 && temporaryAddress != "" {
			loserCount := len(players) - 1
			totalPool := int(float64(baseStake) * float64(finalMul))

			winnerTokenPerPlayer := int(float64(totalPool) * (1.0 - 0.016))
			loserTokenPerPlayer := totalPool / loserCount

			var winnerPointPerPlayer, loserPointPerPlayer int
			if grType == gameResultNormal {
				winnerPointPerPlayer = int(float64(totalPool) * 0.012)
				loserPointPerPlayer = int(float64(totalPool)*0.004) / loserCount
			} else {
				winnerPointPerPlayer = int(float64(totalPool) * 0.016)
				loserPointPerPlayer = 0
			}

			playerRewards = make([]*dao.PlayerReward, 0, len(players))
			for _, player := range players {
				status := playerStatuses[player.player.TemporaryAddress]
				isWinner := player.player.TemporaryAddress == temporaryAddress

				var tokenChange, pointChange int
				var gameResultStatus proto.PlayerGameResultStatus
				if isWinner {
					tokenChange = winnerTokenPerPlayer
					pointChange = winnerPointPerPlayer
					gameResultStatus = proto.PlayerGameResultStatus_PLAYER_WIN
				} else {
					tokenChange = -loserTokenPerPlayer
					pointChange = loserPointPerPlayer
					gameResultStatus = proto.PlayerGameResultStatus_PLAYER_LOSE
				}

				playerRewards = append(playerRewards, &dao.PlayerReward{
					PlayerId:               player.player.PlayerId,
					TemporaryAddress:       player.player.TemporaryAddress,
					TokenChange:            int32(tokenChange),
					PointChange:            int32(pointChange),
					IsOffline:              status == playerStatusOffline,
					Surrendered:            status == playerStatusSurrendered,
					PlayerGameResultStatus: gameResultStatus,
				})
			}

			systemFee = int(float64(totalPool) * 0.016)
		}
	}

	return &dao.BattleReward{
		PlayerRewards: playerRewards,
		SystemFee:     int32(systemFee),
	}
}

func (r *round) handleServerTimeout() (bool, *dao.GameResult, error) {
	var playerRewards []*dao.PlayerReward

	for _, gamePlayer := range r.gamePlayers {
		submittedCard := gamePlayer.getLastSubmittedCard()
		//gamePlayer.totalLostHP = int64(gamePlayer.getLostHP())
		hasCommitment := len(submittedCard.CommitmentHash) > 0
		// Check commitment from first card (if exists)
		playerRewards = append(playerRewards, &dao.PlayerReward{
			PlayerId:               gamePlayer.player.PlayerId,
			TemporaryAddress:       gamePlayer.player.TemporaryAddress,
			TokenChange:            0,
			PointChange:            0,
			IsOffline:              !hasCommitment,
			Surrendered:            gamePlayer.isSurrendered(),
			PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE,
		})
	}

	gameResult := &dao.GameResult{
		GameID:                 r.round.GameID,
		Multiplier:             0,
		WinnerPlayerId:         0,
		WinnerTemporaryAddress: "",
		GameResultType:         proto.GameResultType_GAME_TIE,
		BattleReward: &dao.BattleReward{
			PlayerRewards: playerRewards,
			SystemFee:     0,
		},
	}

	return true, gameResult, nil
}
