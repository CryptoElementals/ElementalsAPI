package game

import (
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/proto"
)

// Round represents a game round with its players
type Round struct {
	round        *dao.Round
	gamePlayers  map[string]*gamePlayer
	battleStates map[string]*playerBattleState // Tracks HP/multipliers during progressive battle
	turnNumber   uint32                        // Current turn number within this round (1-3), runtime state only
}

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
	playerStatusOnline      playerStatus = 0
	playerStatusOffline     playerStatus = 1
	playerStatusSurrendered playerStatus = 2
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

// ExecuteRound executes the round battle logic directly on the Round and returns a simple result
func (r *Round) ExecuteRound() (bool, *dao.GameResult, error) {
	// Validate round
	if err := r.validateRound(); err != nil {
		return false, nil, err
	}

	// Check for server timeout scenarios
	if r.isServerTimeout() {
		return r.handleServerTimeout()
	}

	playerCount := len(r.round.PlayerRoundInfos)
	hasSurrenderedPlayer, hasOfflinePlayer := r.checkPlayerStatuses()

	// Fetch player cards
	playerCards := r.fetchPlayerCards(playerCount, hasOfflinePlayer)

	// Initialize player states
	states := r.initializePlayerStates(playerCount)

	// Execute battle if all players are online
	if !hasOfflinePlayer && !hasSurrenderedPlayer {
		r.executeBattle(states, playerCards, playerCount)
	}

	// Update gamePlayers with final state
	r.updateGamePlayersFromStates(states)

	// Build game result
	isGameOver, gameResult := r.buildGameResult(states, playerCount)

	return isGameOver, gameResult, nil
}

// validateRound validates the round data
func (r *Round) validateRound() error {
	if r.round == nil {
		return fmt.Errorf("round is nil")
	}

	playerCount := len(r.round.PlayerRoundInfos)
	if playerCount < 2 {
		return fmt.Errorf("at least 2 players required")
	}

	if r.round.RoundNumber < 1 || r.round.RoundNumber > 3 {
		return fmt.Errorf("round parameter must be between 1 and 3")
	}

	return nil
}

// isServerTimeout checks if round ended due to server timeout
func (r *Round) isServerTimeout() bool {
	return r.round.CompleteReason == proto.RoundCompleteReason_ROUND_COMPLETE_SERVER_CHAIN_TIMEOUT ||
		r.round.CompleteReason == proto.RoundCompleteReason_ROUND_COMPLETE_SERVER_INTERNAL_TIMEOUT
}

// checkPlayerStatuses checks for surrendered and offline players
func (r *Round) checkPlayerStatuses() (bool, bool) {
	hasSurrenderedPlayer := false
	hasOfflinePlayer := false

	for _, p := range r.round.PlayerRoundInfos {
		if p.Surrendered {
			hasSurrenderedPlayer = true
		}
		// Check commitment from first card (if exists) and card count
		hasCommitment := false
		if len(p.SubmittedCards) > 0 && len(p.SubmittedCards[0].SubmittedCommitment) > 0 {
			hasCommitment = true
		}
		if !hasCommitment || len(p.SubmittedCards) != 3 {
			hasOfflinePlayer = true
		}
		// Early exit if both conditions are already found
		if hasSurrenderedPlayer && hasOfflinePlayer {
			break
		}
	}

	return hasSurrenderedPlayer, hasOfflinePlayer
}

// fetchPlayerCards fetches cards for all players
func (r *Round) fetchPlayerCards(playerCount int, hasOfflinePlayer bool) [][]*dao.Card {
	if hasOfflinePlayer {
		return nil
	}

	playerCards := make([][]*dao.Card, playerCount)
	for i, p := range r.round.PlayerRoundInfos {
		cardIDs := make([]int, 0, len(p.SubmittedCards))
		for _, c := range p.SubmittedCards {
			cardIDs = append(cardIDs, int(c.CardID))
		}
		cards, err := r.getCards(cardIDs)
		if err != nil {
			return nil
		}
		playerCards[i] = cards
	}
	return playerCards
}

// initializePlayerStates initializes the player battle states
func (r *Round) initializePlayerStates(playerCount int) []*playerBattleState {
	states := make([]*playerBattleState, playerCount)
	for i, p := range r.round.PlayerRoundInfos {
		player := r.gamePlayers[p.TemporaryAddress]
		status := r.determinePlayerStatus(p)

		states[i] = &playerBattleState{
			HP:               int(player.currentHP),
			Multiplier:       r.calculateMultiplierByLostHP(int(player.totalLostHP)),
			LostHP:           int(player.totalLostHP),
			PlayerId:         p.PlayerId,
			TemporaryAddress: p.TemporaryAddress,
			Status:           status,
			SubmittedCards:   p.SubmittedCards,
			Surrendered:      p.Surrendered,
		}
	}
	return states
}

// determinePlayerStatus determines the status of a player
func (r *Round) determinePlayerStatus(p *dao.PlayerRoundInfo) playerStatus {
	if p.Surrendered {
		return playerStatusSurrendered
	}
	// Check commitment from first card (if exists) and card count
	hasCommitment := false
	if len(p.SubmittedCards) > 0 && len(p.SubmittedCards[0].SubmittedCommitment) > 0 {
		hasCommitment = true
	}
	if !hasCommitment || len(p.SubmittedCards) != 3 {
		return playerStatusOffline
	}
	return playerStatusOnline
}

// executeBattle executes the card battles
func (r *Round) executeBattle(states []*playerBattleState, playerCards [][]*dao.Card, playerCount int) {
	for cardIdx := 0; cardIdx < 3; cardIdx++ {
		for i := 0; i < playerCount; i++ {
			for j := i + 1; j < playerCount; j++ {
				r.processCardBattle(states[i], states[j], playerCards[i][cardIdx], playerCards[j][cardIdx], cardIdx)
			}
		}

		// Stop if any player has 0 HP
		if r.hasAnyPlayerZeroHP(states) {
			break
		}
	}
}

// processCardBattle processes a single card battle between two players
func (r *Round) processCardBattle(p1, p2 *playerBattleState, card1, card2 *dao.Card, cardIdx int) {
	// Get elemental relation
	relation := r.getElementalRelation(card1, card2, p1.PlayerId, p2.PlayerId)
	effects1 := r.buildPlayerEffects(relation.P1Type, card1, card2, p1.PlayerId, p1.TemporaryAddress, p2.PlayerId)
	effects2 := r.buildPlayerEffects(relation.P2Type, card2, card1, p2.PlayerId, p2.TemporaryAddress, p1.PlayerId)

	// Record initial state
	p1BeforeHP := p1.HP
	p2BeforeHP := p2.HP
	p1BeforeMul := p1.Multiplier
	p2BeforeMul := p2.Multiplier

	// Execute effects and update state
	r.updatePlayerHPAndLostHP(p1, p1BeforeHP, r.executeEffects(effects1))
	r.updatePlayerHPAndLostHP(p2, p2BeforeHP, r.executeEffects(effects2))

	// Update submitted card stats
	r.updateCardStats(p1.SubmittedCards, cardIdx, p1BeforeHP, p1.HP, p1BeforeMul, p1.Multiplier, relation.P1Type, relation.P1Description, effects1)
	r.updateCardStats(p2.SubmittedCards, cardIdx, p2BeforeHP, p2.HP, p2BeforeMul, p2.Multiplier, relation.P2Type, relation.P2Description, effects2)
}

// hasAnyPlayerZeroHP checks if any player has zero HP
func (r *Round) hasAnyPlayerZeroHP(states []*playerBattleState) bool {
	for _, st := range states {
		if st.HP == 0 {
			return true
		}
	}
	return false
}

// updateGamePlayersFromStates updates gamePlayers from final states
func (r *Round) updateGamePlayersFromStates(states []*playerBattleState) {
	for _, st := range states {
		player := r.gamePlayers[st.TemporaryAddress]
		player.totalLostHP = int64(st.LostHP)
		player.roundPlayer.LostHP = int32(st.LostHP)
		// HP from last card if available, otherwise use current HP
		if len(st.SubmittedCards) > 0 {
			player.currentHP = int64(st.SubmittedCards[len(st.SubmittedCards)-1].HealthAfter)
		} else {
			player.currentHP = int64(st.HP)
		}
	}
}

// buildGameResult builds the game result from player states
func (r *Round) buildGameResult(states []*playerBattleState, playerCount int) (bool, *dao.GameResult) {
	// Convert to game end states
	gameEndStates := make([]*gameEndState, playerCount)
	for i, st := range states {
		gameEndStates[i] = &gameEndState{
			HP:               st.HP,
			Multiplier:       st.Multiplier,
			PlayerId:         st.PlayerId,
			TemporaryAddress: st.TemporaryAddress,
			Status:           st.Status,
		}
	}

	// Check if game is over
	isGameOver, grType, winner, temporaryAddress, finalMul := r.checkGameOver(gameEndStates, r.round.RoundNumber)

	// Build game result if game is over
	var gameResult *dao.GameResult
	if isGameOver {
		// Build player status map for reward calculation
		playerStatuses := make(map[string]playerStatus)
		for _, st := range states {
			playerStatuses[st.TemporaryAddress] = st.Status
		}

		// Build battle reward
		battleReward := r.calculateBattleReward(states, grType, winner, temporaryAddress, finalMul, playerStatuses)

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

// ===== Helper Functions =====

// updatePlayerHPAndLostHP calculates HP changes and updates LostHP and Multiplier
func (r *Round) updatePlayerHPAndLostHP(state *playerBattleState, beforeHP int, hpDelta int) {
	// Apply delta and clamp HP to >= 0
	state.HP += hpDelta
	if state.HP < 0 {
		state.HP = 0
	}

	// Compute non-negative damage
	damage := beforeHP - state.HP
	if damage < 0 {
		damage = 0
	}

	// Accumulate LostHP and clamp to initial HP cap
	state.LostHP += damage
	if state.LostHP > int(config.GameParams.InitialHP) {
		state.LostHP = int(config.GameParams.InitialHP)
	}

	// Recompute multiplier from total lost HP
	state.Multiplier = r.calculateMultiplierByLostHP(state.LostHP)
}

// InitializeBattleStates initializes the battleStates map from gamePlayers
func (r *Round) InitializeBattleStates() error {
	if r.battleStates == nil {
		r.battleStates = make(map[string]*playerBattleState)
	}

	// Initialize states for all players
	for _, p := range r.round.PlayerRoundInfos {
		player := r.gamePlayers[p.TemporaryAddress]
		r.battleStates[p.TemporaryAddress] = &playerBattleState{
			HP:               int(player.currentHP),
			Multiplier:       r.calculateMultiplierByLostHP(int(player.totalLostHP)),
			LostHP:           int(player.totalLostHP),
			PlayerId:         p.PlayerId,
			TemporaryAddress: p.TemporaryAddress,
			Status:           r.determinePlayerStatus(p),
			SubmittedCards:   p.SubmittedCards,
			Surrendered:      p.Surrendered,
		}
	}
	return nil
}

// ApplyCardPair applies a single pair of cards between two players, updates states and checks game over.
// tempAddr1 and tempAddr2 are players' temporary addresses participating in this pair battle.
// card1 and card2 are the cards used by the respective players for this pair.
// cardIdx is the index of the card within this round (0..2) to update submitted card stats.
// It returns (isGameOver, gameResult, error). If not over, gameResult is nil.
func (r *Round) ApplyCardPair(tempAddr1, tempAddr2 string, card1, card2 *dao.Card, cardIdx int) (bool, *dao.GameResult, error) {
	if err := r.validateRound(); err != nil {
		return false, nil, err
	}
	if r.isServerTimeout() {
		return r.handleServerTimeout()
	}

	// Initialize battleStates if not already done
	if r.battleStates == nil {
		if err := r.InitializeBattleStates(); err != nil {
			return false, nil, err
		}
	}

	// Get existing battle states (they persist across pair applications)
	st1, ok1 := r.battleStates[tempAddr1]
	st2, ok2 := r.battleStates[tempAddr2]
	if !ok1 || !ok2 {
		return false, nil, fmt.Errorf("battle state not found for players")
	}

	// If either is offline/surrendered, skip battle application
	if st1.Status != playerStatusOnline || st2.Status != playerStatusOnline {
		// Return current state without modification
		isGameOver, gameResult := r.checkGameOverFromBattleStates()
		return isGameOver, gameResult, nil
	}

	// Process the card battle (updates st1 and st2 in place)
	r.processCardBattle(st1, st2, card1, card2, cardIdx)

	// Check game over using all battle states
	isGameOver, gameResult := r.checkGameOverFromBattleStates()
	return isGameOver, gameResult, nil
}

// checkGameOverFromBattleStates checks if game is over using current battle states
func (r *Round) checkGameOverFromBattleStates() (bool, *dao.GameResult) {
	// Convert battle states to game end states
	playerCount := len(r.round.PlayerRoundInfos)
	gameEndStates := make([]*gameEndState, 0, playerCount)

	for _, st := range r.battleStates {
		gameEndStates = append(gameEndStates, &gameEndState{
			HP:               st.HP,
			Multiplier:       st.Multiplier,
			PlayerId:         st.PlayerId,
			TemporaryAddress: st.TemporaryAddress,
			Status:           st.Status,
		})
	}

	// Check if game is over
	isGameOver, grType, winner, temporaryAddress, finalMul := r.checkGameOver(gameEndStates, r.round.RoundNumber)

	// Build game result if game is over
	var gameResult *dao.GameResult
	if isGameOver {
		// Build player status map
		playerStatuses := make(map[string]playerStatus)
		for _, st := range r.battleStates {
			playerStatuses[st.TemporaryAddress] = st.Status
		}

		// Convert to states array for reward calculation
		states := make([]*playerBattleState, 0, len(r.battleStates))
		for _, st := range r.battleStates {
			states = append(states, st)
		}

		// Build battle reward
		battleReward := r.calculateBattleReward(states, grType, winner, temporaryAddress, finalMul, playerStatuses)

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
func (r *Round) updateCardStats(cards []*dao.RoundSubmittedCard, cardIdx int, beforeHP, afterHP int, beforeMul, afterMul uint32, relationType, description string, effects []battleEffect) {
	if cardIdx < len(cards) {
		cards[cardIdx].HealthBefore = uint32(beforeHP)
		cards[cardIdx].HealthAfter = uint32(afterHP)
		cards[cardIdx].MultiplierBefore = beforeMul
		cards[cardIdx].MultiplierAfter = afterMul
		cards[cardIdx].Description = description
		cards[cardIdx].ElementRelation = r.mapElementRelationStringToEnum(relationType)
		cards[cardIdx].CardEffects = r.battleEffectsToDaoEffects(effects)
	}
}

func (r *Round) getCards(cardIDs []int) ([]*dao.Card, error) {
	cards := make([]*dao.Card, 0, len(cardIDs))
	for _, cardID := range cardIDs {
		dbCard, err := db.GetCardByID(cardID)
		if err != nil {
			return nil, fmt.Errorf("failed to get card [ID:%d]: %v", cardID, err)
		}
		cards = append(cards, dbCard)
	}
	return cards, nil
}

// ExecuteCardIndex executes battles for a single card index (0, 1, or 2) between all players
// It initializes battle states if needed and processes all player pairs for the given card index
// Returns (isGameOver, gameResult, error)
func (r *Round) ExecuteCardIndex(cardIdx int) (bool, *dao.GameResult, error) {
	if err := r.validateRound(); err != nil {
		return false, nil, err
	}
	if r.isServerTimeout() {
		return r.handleServerTimeout()
	}

	// Initialize battleStates if not already done
	if r.battleStates == nil {
		if err := r.InitializeBattleStates(); err != nil {
			return false, nil, err
		}
	}

	playerCount := len(r.round.PlayerRoundInfos)
	hasSurrenderedPlayer, hasOfflinePlayer := r.checkPlayerStatuses()

	// If there are offline or surrendered players, skip battle but still check game over
	if hasOfflinePlayer || hasSurrenderedPlayer {
		isGameOver, gameResult := r.checkGameOverFromBattleStates()
		return isGameOver, gameResult, nil
	}

	// Fetch cards for all players for this card index
	playerCards := make([][]*dao.Card, playerCount)
	for i, p := range r.round.PlayerRoundInfos {
		if cardIdx >= len(p.SubmittedCards) {
			return false, nil, fmt.Errorf("card index %d not found for player %s", cardIdx, p.TemporaryAddress)
		}
		cardID := int(p.SubmittedCards[cardIdx].CardID)
		cards, err := r.getCards([]int{cardID})
		if err != nil {
			return false, nil, fmt.Errorf("failed to get card for player %s: %v", p.TemporaryAddress, err)
		}
		if len(cards) == 0 {
			return false, nil, fmt.Errorf("card not found for player %s", p.TemporaryAddress)
		}
		playerCards[i] = cards
	}

	// Execute battles for this card index between all player pairs
	for i := 0; i < playerCount; i++ {
		for j := i + 1; j < playerCount; j++ {
			st1 := r.battleStates[r.round.PlayerRoundInfos[i].TemporaryAddress]
			st2 := r.battleStates[r.round.PlayerRoundInfos[j].TemporaryAddress]

			// Only process if both players are online
			if st1.Status == playerStatusOnline && st2.Status == playerStatusOnline {
				r.processCardBattle(st1, st2, playerCards[i][0], playerCards[j][0], cardIdx)
			}
		}
	}

	// Update gamePlayers with current state
	r.updateGamePlayersFromBattleStates()

	// Check if game is over
	isGameOver, gameResult := r.checkGameOverFromBattleStates()
	return isGameOver, gameResult, nil
}

// updateGamePlayersFromBattleStates updates gamePlayers from battleStates
func (r *Round) updateGamePlayersFromBattleStates() {
	for _, st := range r.battleStates {
		player := r.gamePlayers[st.TemporaryAddress]
		player.totalLostHP = int64(st.LostHP)
		player.roundPlayer.LostHP = int32(st.LostHP)
		// HP from last card if available, otherwise use current HP
		if len(st.SubmittedCards) > 0 {
			player.currentHP = int64(st.SubmittedCards[len(st.SubmittedCards)-1].HealthAfter)
		} else {
			player.currentHP = int64(st.HP)
		}
	}
}

func (r *Round) calculateMultiplierByLostHP(lostHP int) uint32 {
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

func (r *Round) getElementalRelation(card1, card2 *dao.Card, playerId1, playerId2 int64) *elementalRelation {
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
func (r *Round) buildRelation(p1Type, p2Type string, elem1 string, playerId1 int64, elem2 string, playerId2 int64) *elementalRelation {
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

func truncateAddress(address string) string {
	if len(address) <= 10 {
		return address
	}
	return address[:6] + "..." + address[len(address)-4:]
}

func (r *Round) buildPlayerEffects(playerType string, selfCard, opponentCard *dao.Card, selfPlayerId int64, selfTemp string, opponentPlayerId int64) []battleEffect {
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

func (r *Round) executeEffects(effects []battleEffect) int {
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

func (r *Round) battleEffectsToDaoEffects(effects []battleEffect) []*dao.CardEffect {
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

func (r *Round) mapElementRelationStringToEnum(s string) proto.ElementRelation {
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

func (r *Round) checkGameOver(states []*gameEndState, round uint32) (bool, gameResultType, string, string, uint32) {
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

func (r *Round) checkGameOverByHP(states []*gameEndState, round uint32, hasOffline bool) (bool, gameResultType, string, string, uint32) {
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

func (r *Round) buildResult(over bool, gType gameResultType, winnerPlayerIds []int64, winnerTemps []string, mul uint32) (bool, gameResultType, string, string, uint32) {
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

// playerBattleState represents player state during battle
type playerBattleState struct {
	HP               int
	Multiplier       uint32
	LostHP           int
	PlayerId         int64
	TemporaryAddress string
	Status           playerStatus
	SubmittedCards   []*dao.RoundSubmittedCard
	Surrendered      bool
}

func (r *Round) calculateBattleReward(states []*playerBattleState, grType gameResultType, winnerPlayerIdsStr, temporaryAddress string, finalMul uint32, playerStatuses map[string]playerStatus) *dao.BattleReward {
	baseStake := config.GameParams.BaseStake
	var playerRewards []*dao.PlayerReward
	var systemFee int

	// Use states directly, no need for conversion
	switch grType {
	case gameResultTie:
		tokenDeduction := int(float64(baseStake) * 0.008)
		pointGain := int(float64(baseStake) * 0.008)
		playerRewards = make([]*dao.PlayerReward, 0, len(states))

		for _, st := range states {
			status := playerStatuses[st.TemporaryAddress]
			playerRewards = append(playerRewards, &dao.PlayerReward{
				PlayerId:               st.PlayerId,
				TemporaryAddress:       st.TemporaryAddress,
				TokenChange:            int32(-tokenDeduction),
				PointChange:            int32(pointGain),
				IsOffline:              status == playerStatusOffline,
				Surrendered:            status == playerStatusSurrendered,
				PlayerGameResultStatus: proto.PlayerGameResultStatus_PLAYER_TIE,
			})
		}
		systemFee = tokenDeduction * len(states)

	case gameResultNormal, gameResultKO:
		if winnerPlayerIdsStr != "" {
			winnerTemporaryList := strings.Split(temporaryAddress, "|")
			winnerTemporaryAddresses := make(map[string]bool, len(winnerTemporaryList))
			for _, addr := range winnerTemporaryList {
				winnerTemporaryAddresses[addr] = true
			}

			winnerCount := len(winnerTemporaryList)
			loserCount := len(states) - winnerCount
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

			playerRewards = make([]*dao.PlayerReward, 0, len(states))
			for _, st := range states {
				status := playerStatuses[st.TemporaryAddress]
				isWinner := winnerTemporaryAddresses[st.TemporaryAddress]

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
					PlayerId:               st.PlayerId,
					TemporaryAddress:       st.TemporaryAddress,
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

func (r *Round) handleServerTimeout() (bool, *dao.GameResult, error) {
	var playerRewards []*dao.PlayerReward

	for _, p := range r.round.PlayerRoundInfos {
		player := r.gamePlayers[p.TemporaryAddress]
		player.roundPlayer.LostHP = p.LostHP
		player.totalLostHP = int64(p.LostHP)

		// Check commitment from first card (if exists)
		hasCommitment := false
		if len(p.SubmittedCards) > 0 && len(p.SubmittedCards[0].SubmittedCommitment) > 0 {
			hasCommitment = true
		}
		playerRewards = append(playerRewards, &dao.PlayerReward{
			PlayerId:               p.PlayerId,
			TemporaryAddress:       p.TemporaryAddress,
			TokenChange:            0,
			PointChange:            0,
			IsOffline:              !hasCommitment || len(p.SubmittedCards) < 3,
			Surrendered:            p.Surrendered,
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
