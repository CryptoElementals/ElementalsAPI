package game

import (
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
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
	"Metal": "Wood",
	"Wood":  "Earth",
	"Earth": "Water",
	"Water": "Fire",
	"Fire":  "Metal",
}

// Sheng (Nurture) relations - Metal nurtures Water, Water nurtures Wood, etc.
var shengRelations = map[string]string{
	"Metal": "Water",
	"Water": "Wood",
	"Wood":  "Fire",
	"Fire":  "Earth",
	"Earth": "Metal",
}

// BattleEffect represents an effect in battle
type battleEffect struct {
	Type                   proto.BattleEffectType
	Value                  int
	TargetPlayerId         int64
	TargetTemporaryAddress string
	Description            string
}

// PlayerStatus represents a player's status
type playerStatus int32

const (
	playerStatusOnline playerStatus = iota
	playerStatusOffline
	playerStatusSurrendered
)

// GameResultType represents the type of game result
type gameResultType int32

const (
	gameResultNormal gameResultType = iota
	gameResultKO
	gameResultTie
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
	isGameOver, gameResult := r.checkGameOverFromGamePlayers()
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

	// Check if game is over (gamePlayers are already updated in place during battle)
	isGameOver, gameResult = r.checkGameOverFromGamePlayers()
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

// checkPlayerStatuses checks for surrendered and offline players
func (r *round) checkPlayerStatuses() (bool, bool) {
	hasSurrenderedPlayer := false
	hasOfflinePlayer := false

	for _, gamePlayer := range r.gamePlayers {
		if gamePlayer.isSurrendered() {
			hasSurrenderedPlayer = true
		}
		// Check commitment from submitted cards
		submittedCards := gamePlayer.getLastSubmittedCard()
		if submittedCards == nil || len(submittedCards.CommitmentHash) == 0 {
			hasOfflinePlayer = true
		}
		// Early exit if both conditions are already found
		if hasSurrenderedPlayer && hasOfflinePlayer {
			break
		}
	}

	return hasSurrenderedPlayer, hasOfflinePlayer
}

// processCardBattle processes a single card battle between two players
func (r *round) processCardBattle(p1, p2 *gamePlayer, card1, card2 *dao.Card) {
	// Get elemental relation
	relation := r.getElementalRelation(card1, card2, p1.player.PlayerId, p2.player.PlayerId)
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
	damage := int(beforeHP) - int(player.currentHP)
	if damage < 0 {
		damage = 0
	}

	// Accumulate LostHP and clamp to initial HP cap
	player.totalLostHP += int64(damage)
	if player.totalLostHP > config.GameParams.InitialHP {
		player.totalLostHP = config.GameParams.InitialHP
	}

	// Recompute multiplier from total lost HP
	player.multiplier = r.calculateMultiplierByLostHP(int(player.totalLostHP))
}

// checkGameOverFromGamePlayers checks if game is over using current game players
func (r *round) checkGameOverFromGamePlayers() (bool, *dao.GameResult) {
	// Convert game players to game end states
	playerCount := len(r.gamePlayers)
	gameEndStates := make([]*gameEndState, 0, playerCount)
	for _, player := range r.gamePlayers {
		submittedCard := player.getLastSubmittedCard()
		hasSubmittedCommitment := submittedCard != nil && len(submittedCard.CommitmentHash) > 0
		if player.status != playerStatusSurrendered && !hasSubmittedCommitment {
			player.status = playerStatusOffline
		}
		gameEndStates = append(gameEndStates, &gameEndState{
			HP:               int(player.currentHP),
			Multiplier:       player.multiplier,
			PlayerId:         player.player.PlayerId,
			TemporaryAddress: player.player.TemporaryAddress,
			Status:           player.status,
		})
	}

	// Check if game is over
	isGameOver, grType, winner, temporaryAddress, finalMul := r.checkGameOver(gameEndStates, r.round.RoundNumber)

	// Build game result if game is over
	var gameResult *dao.GameResult
	if isGameOver {
		// Build player status map
		playerStatuses := make(map[string]playerStatus)
		for _, player := range r.gamePlayers {
			playerStatuses[player.player.TemporaryAddress] = player.status
		}

		// Build battle reward using game players
		battleReward := r.calculateBattleRewardFromGamePlayers(r.gamePlayers, grType, winner, temporaryAddress, finalMul, playerStatuses)

		// Parse first winner ID from winner string (can be multiple separated by "|")
		var winnerPlayerId int64
		if winner != "" {
			winnerIds := strings.Split(winner, "|")
			if len(winnerIds) > 0 {
				fmt.Sscanf(winnerIds[0], "%d", &winnerPlayerId)
			}
		}
		gameResult = &dao.GameResult{
			GameID:                 r.round.GameID,
			Multiplier:             int32(finalMul),
			WinnerPlayerId:         winnerPlayerId,
			WinnerTemporaryAddress: temporaryAddress,
			GameResultType:         proto.GameResultType(grType),
			BattleReward:           battleReward,
		}
	}

	return isGameOver, gameResult
}

// updateCardStats updates the card stats in the DAO structure
func (r *round) updateCardStats(card *dao.TurnSubmittedCard, beforeHP, afterHP int, beforeMul, afterMul uint32, relationType, description string, effects []battleEffect) {
	card.HealthBefore = uint32(beforeHP)
	card.HealthAfter = uint32(afterHP)
	card.MultiplierBefore = beforeMul
	card.MultiplierAfter = afterMul
	card.Description = description
	card.ElementRelation = r.mapElementRelationStringToEnum(relationType)
	card.CardEffects = r.battleEffectsToDaoEffects(effects)
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
	newMultiplier := 1 + bonusMultiplier
	if newMultiplier > 9 {
		newMultiplier = 9
	}
	return newMultiplier
}

func (r *round) getElementalRelation(card1, card2 *dao.Card, playerId1, playerId2 int64) *elementalRelation {
	// Check Ke (overpower) relations
	if target, hasRelation := keRelations[card1.ElementType]; hasRelation && target == card2.ElementType {
		return r.buildRelation("overpower", "overpowered", card1.ElementType, playerId1, card2.ElementType, playerId2)
	}
	if target, hasRelation := keRelations[card2.ElementType]; hasRelation && target == card1.ElementType {
		return r.buildRelation("overpowered", "overpower", card1.ElementType, playerId1, card2.ElementType, playerId2)
	}

	// Check Sheng (nurture) relations
	if target, hasRelation := shengRelations[card1.ElementType]; hasRelation && target == card2.ElementType {
		return r.buildRelation("nurture", "nurtured", card1.ElementType, playerId1, card2.ElementType, playerId2)
	}
	if target, hasRelation := shengRelations[card2.ElementType]; hasRelation && target == card1.ElementType {
		return r.buildRelation("nurtured", "nurture", card1.ElementType, playerId1, card2.ElementType, playerId2)
	}

	// Even (tie)
	return &elementalRelation{
		P1Type:        "even",
		P2Type:        "even",
		P1Description: fmt.Sprintf("%s(%d) and %s(%d) are even", card1.ElementType, playerId1, card2.ElementType, playerId2),
		P2Description: fmt.Sprintf("%s(%d) and %s(%d) are even", card2.ElementType, playerId2, card1.ElementType, playerId1),
	}
}

// buildRelation builds an elemental relation structure
func (r *round) buildRelation(p1Type, p2Type string, elem1 string, playerId1 int64, elem2 string, playerId2 int64) *elementalRelation {
	var p1Desc, p2Desc string
	switch p1Type {
	case "overpower":
		p1Desc = fmt.Sprintf("%s(%d) overpowers %s(%d)", elem1, playerId1, elem2, playerId2)
		p2Desc = fmt.Sprintf("%s(%d) is overpowered by %s(%d)", elem2, playerId2, elem1, playerId1)
	case "overpowered":
		p1Desc = fmt.Sprintf("%s(%d) is overpowered by %s(%d)", elem1, playerId1, elem2, playerId2)
		p2Desc = fmt.Sprintf("%s(%d) overpowers %s(%d)", elem2, playerId2, elem1, playerId1)
	case "nurture":
		p1Desc = fmt.Sprintf("%s(%d) nurtures %s(%d)", elem1, playerId1, elem2, playerId2)
		p2Desc = fmt.Sprintf("%s(%d) is nurtured by %s(%d)", elem2, playerId2, elem1, playerId1)
	case "nurtured":
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

func (r *round) buildPlayerEffects(playerType string, selfCard, opponentCard *dao.Card, selfPlayerId int64, selfTemp string, opponentPlayerId int64) []battleEffect {
	var effects []battleEffect

	desc := func(action string) string {
		return fmt.Sprintf("%s(%d) %s %s(%d)", selfCard.ElementType, selfPlayerId, action, opponentCard.ElementType, opponentPlayerId)
	}

	switch playerType {
	case "overpower":
		// No effects
	case "overpowered":
		attackValue := opponentCard.Attack - selfCard.Defense
		// Two attacks for overpowered
		for _, action := range []string{"is attacked by", "is double attacked by"} {
			effects = append(effects, battleEffect{
				Type:                   proto.BattleEffectType_ATTACK,
				Value:                  attackValue,
				Description:            desc(action),
				TargetPlayerId:         selfPlayerId,
				TargetTemporaryAddress: selfTemp,
			})
		}
	case "nurture":
		// No effects
	case "nurtured":
		effects = append(effects, battleEffect{
			Type:                   proto.BattleEffectType_HEAL,
			Value:                  selfCard.LifeForce,
			Description:            desc("is healed by"),
			TargetPlayerId:         selfPlayerId,
			TargetTemporaryAddress: selfTemp,
		})
	case "even":
		effects = append(effects, battleEffect{
			Type:                   proto.BattleEffectType_ATTACK,
			Value:                  opponentCard.Attack - selfCard.Defense,
			Description:            desc("is attacked by"),
			TargetPlayerId:         selfPlayerId,
			TargetTemporaryAddress: selfTemp,
		})
	}

	return effects
}

func (r *round) executeEffects(effects []battleEffect) int {
	hpDelta := 0
	for _, effect := range effects {
		value := effect.Value
		if value < 0 {
			value = 0
		}
		switch effect.Type {
		case proto.BattleEffectType_ATTACK:
			hpDelta -= value
		case proto.BattleEffectType_HEAL:
			hpDelta += value
		}
	}
	return hpDelta
}

func (r *round) battleEffectsToDaoEffects(effects []battleEffect) []*dao.CardEffect {
	daoEffects := make([]*dao.CardEffect, len(effects))
	for i, effect := range effects {
		daoEffects[i] = &dao.CardEffect{
			Type:                   effect.Type,
			Value:                  int32(effect.Value),
			Description:            effect.Description,
			TargetPlayerId:         effect.TargetPlayerId,
			TargetTemporaryAddress: effect.TargetTemporaryAddress,
		}
	}
	return daoEffects
}

func (r *round) mapElementRelationStringToEnum(s string) proto.ElementRelation {
	switch s {
	case "overpower":
		return proto.ElementRelation_OVER_POWER
	case "overpowered":
		return proto.ElementRelation_OVER_POWERED
	case "nurture":
		return proto.ElementRelation_NURTURE
	case "nurtured":
		return proto.ElementRelation_NURTURED
	default:
		return proto.ElementRelation_TIE
	}
}

func (r *round) checkGameOver(states []*gameEndState, round uint32) (bool, gameResultType, string, string, uint32) {
	// Count surrendered players
	surrenderedCount := 0
	maxLoserMul := uint32(1)
	var winnerPlayerIds []int64
	var winnerTemps []string

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
			return true, gameResultTie, "", "", 1
		}
		// Build winner lists
		for _, state := range states {
			if state.Status != playerStatusSurrendered {
				winnerPlayerIds = append(winnerPlayerIds, state.PlayerId)
				winnerTemps = append(winnerTemps, state.TemporaryAddress)
			}
		}
		return r.buildResult(true, gameResultKO, winnerPlayerIds, winnerTemps, maxLoserMul)
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
			return true, gameResultTie, "", "", 1
		}
		for _, state := range states {
			if state.Status == playerStatusOnline {
				winnerPlayerIds = append(winnerPlayerIds, state.PlayerId)
				winnerTemps = append(winnerTemps, state.TemporaryAddress)
			}
		}
		return r.buildResult(true, gameResultNormal, winnerPlayerIds, winnerTemps, offlineMaxMul)
	}

	// Check by HP
	return r.checkGameOverByHP(states, round, false)
}

func (r *round) checkGameOverByHP(states []*gameEndState, round uint32, hasOffline bool) (bool, gameResultType, string, string, uint32) {
	hps := make([]int, len(states))
	playerIds := make([]int64, len(states))
	temps := make([]string, len(states))
	multipliers := make([]uint32, len(states))

	for i, state := range states {
		hps[i] = state.HP
		playerIds[i] = state.PlayerId
		temps[i] = state.TemporaryAddress
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
			return true, gameResultTie, "", "", 1
		}
		if hasOffline || round == 3 {
			return true, gameResultTie, "", "", 1
		}
		return false, gameResultNormal, "", "", 1
	}

	hasZeroHP := false
	for _, hp := range hps {
		if hp == 0 {
			hasZeroHP = true
			break
		}
	}

	var winnerPlayerIds []int64
	var winnerTemps []string
	var gType gameResultType
	var finalMultiplier uint32 = 1

	if hasZeroHP {
		gType = gameResultKO
		maxLoserMul := uint32(1)
		for i, hp := range hps {
			if hp > 0 {
				winnerPlayerIds = append(winnerPlayerIds, playerIds[i])
				winnerTemps = append(winnerTemps, temps[i])
			} else {
				if multipliers[i] > maxLoserMul {
					maxLoserMul = multipliers[i]
				}
			}
		}
		finalMultiplier = maxLoserMul
	} else {
		if !hasOffline && round != 3 {
			return false, gameResultNormal, "", "", 1
		}

		gType = gameResultNormal
		maxHP := -1
		for _, hp := range hps {
			if hp > maxHP {
				maxHP = hp
			}
		}

		maxLoserMul := uint32(1)
		for i, hp := range hps {
			if hp == maxHP {
				winnerPlayerIds = append(winnerPlayerIds, playerIds[i])
				winnerTemps = append(winnerTemps, temps[i])
			} else {
				if multipliers[i] > maxLoserMul {
					maxLoserMul = multipliers[i]
				}
			}
		}
		finalMultiplier = maxLoserMul
	}

	return r.buildResult(true, gType, winnerPlayerIds, winnerTemps, finalMultiplier)
}

func (r *round) buildResult(over bool, gType gameResultType, winnerPlayerIds []int64, winnerTemps []string, mul uint32) (bool, gameResultType, string, string, uint32) {
	winnersStr := ""
	winnerTempsStr := ""
	if len(winnerPlayerIds) > 0 {
		winnerIdStrs := make([]string, len(winnerPlayerIds))
		for i, id := range winnerPlayerIds {
			winnerIdStrs[i] = fmt.Sprintf("%d", id)
		}
		winnersStr = strings.Join(winnerIdStrs, "|")
		winnerTempsStr = strings.Join(winnerTemps, "|")
	}
	return over, gType, winnersStr, winnerTempsStr, mul
}

func (r *round) calculateBattleRewardFromGamePlayers(players map[string]*gamePlayer, grType gameResultType, winnerPlayerIdsStr, temporaryAddress string, finalMul uint32, playerStatuses map[string]playerStatus) *dao.BattleReward {
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
		if winnerPlayerIdsStr != "" {
			winnerTemporaryList := strings.Split(temporaryAddress, "|")
			winnerTemporaryAddresses := make(map[string]bool, len(winnerTemporaryList))
			for _, addr := range winnerTemporaryList {
				winnerTemporaryAddresses[addr] = true
			}

			winnerCount := len(winnerTemporaryList)
			loserCount := len(players) - winnerCount
			totalPool := int(float64(baseStake) * float64(finalMul))

			winnerTokenPerPlayer := int(float64(totalPool)*(1.0-0.016)) / winnerCount
			loserTokenPerPlayer := totalPool / loserCount

			var winnerPointPerPlayer, loserPointPerPlayer int
			if grType == gameResultNormal {
				winnerPointPerPlayer = int(float64(totalPool)*0.012) / winnerCount
				loserPointPerPlayer = int(float64(totalPool)*0.004) / loserCount
			} else {
				winnerPointPerPlayer = int(float64(totalPool)*0.016) / winnerCount
				loserPointPerPlayer = 0
			}

			playerRewards = make([]*dao.PlayerReward, 0, len(players))
			for _, player := range players {
				status := playerStatuses[player.player.TemporaryAddress]
				isWinner := winnerTemporaryAddresses[player.player.TemporaryAddress]

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
