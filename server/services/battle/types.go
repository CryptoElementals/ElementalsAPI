package battle

// Card card structure
type Card struct {
	ID          int    `json:"id"`           // Card ID
	ElementType string `json:"element_type"` // Element type (Metal, Wood, Water, Fire, Earth)
	Level       string `json:"level"`        // Card level (legendary, epic, rare, normal)
	LifeForce   int    `json:"life_force"`   // Life force
	Attack      int    `json:"attack"`       // Attack power
	Defense     int    `json:"defense"`      // Defense power
	Name        string `json:"name"`         // Card name
	Description string `json:"description"`  // Card description
	ImageURL    string `json:"image_url"`    // Card image URL
}

// Player player structure
type Player struct {
	Address    string  `json:"address"`    // Player address
	Cards      []Card  `json:"cards"`      // Player's cards
	HP         int     `json:"hp"`         // Health points
	Multiplier float64 `json:"multiplier"` // Score multiplier
}

// BattleInput battle input parameters
type BattleInput struct {
	Player1Address    string  `json:"player1_address"`    // Player 1 address
	Player2Address    string  `json:"player2_address"`    // Player 2 address
	Player1HP         int     `json:"player1_hp"`         // Player 1 initial health
	Player2HP         int     `json:"player2_hp"`         // Player 2 initial health
	Player1Multiplier float64 `json:"player1_multiplier"` // Player 1 score multiplier
	Player2Multiplier float64 `json:"player2_multiplier"` // Player 2 score multiplier
	Player1Cards      []int   `json:"player1_cards"`      // Player 1 card ID list
	Player2Cards      []int   `json:"player2_cards"`      // Player 2 card ID list
	Player1LostHP     int     `json:"player1_lost_hp"`    // Player 1 accumulated lost health
	Player2LostHP     int     `json:"player2_lost_hp"`    // Player 2 accumulated lost health
}

// BattleResult battle result
type BattleResult struct {
	Player1Address         string        `json:"player1_address"`          // Player 1 address
	Player2Address         string        `json:"player2_address"`          // Player 2 address
	Stage                  int           `json:"stage"`                    // Stage number
	Rounds                 []RoundResult `json:"rounds"`                   // Round results
	Player1FinalHP         int           `json:"player1_final_hp"`         // Player 1 final health
	Player2FinalHP         int           `json:"player2_final_hp"`         // Player 2 final health
	Player1LostHP          int           `json:"player1_lost_hp"`          // Player 1 accumulated lost health
	Player2LostHP          int           `json:"player2_lost_hp"`          // Player 2 accumulated lost health
	Player1FinalMultiplier float64       `json:"player1_final_multiplier"` // Player 1 final multiplier
	Player2FinalMultiplier float64       `json:"player2_final_multiplier"` // Player 2 final multiplier
	GameFinalMultiplier    float64       `json:"game_final_multiplier"`    // Game final multiplier (take loser's multiplier, tie is 1)
	Winner                 string        `json:"winner"`                   // Winner address
	IsGameOver             bool          `json:"is_game_over"`             // Whether game is over
	GameResultType         string        `json:"game_result_type"`         // Game result type
	Reward                 *BattleReward `json:"reward"`                   // Battle reward
}

// RoundResult round result
type RoundResult struct {
	RoundNumber            int            `json:"round_number"`             // Round number
	Player1CardID          int            `json:"player1_card_id"`          // Player 1 used card ID
	Player2CardID          int            `json:"player2_card_id"`          // Player 2 used card ID
	RelationType           string         `json:"relation_type"`            // Elemental relation type
	Actions                []BattleAction `json:"actions"`                  // Executed action list
	Player1Damage          int            `json:"player1_damage"`           // Player 1 damage taken (negative) or healing (positive)
	Player2Damage          int            `json:"player2_damage"`           // Player 2 damage taken (negative) or healing (positive)
	Player1HPAfter         int            `json:"player1_hp_after"`         // Player 1 health after round
	Player2HPAfter         int            `json:"player2_hp_after"`         // Player 2 health after round
	Player1MultiplierAfter float64        `json:"player1_multiplier_after"` // Player 1 multiplier after round
	Player2MultiplierAfter float64        `json:"player2_multiplier_after"` // Player 2 multiplier after round
	Description            string         `json:"description"`              // Round description
}

// ElementalRelation elemental relation
type ElementalRelation struct {
	Type        string `json:"type"`        // Relation type: "overpower"(overpower), "overpowered"(overpowered), "nurture"(nurture), "nurtured"(nurtured), "even"(even)
	Description string `json:"description"` // Relation description
}

// BattleAction battle action
type BattleAction struct {
	Type        string `json:"type"`        // Action type: "attack", "heal"
	Target      string `json:"target"`      // Target: "player1", "player2"
	Value       int    `json:"value"`       // Action value
	Description string `json:"description"` // Action description
}

// TokenReward token reward
type TokenReward struct {
	Player1TokenChange int `json:"player1_token_change"` // Player 1 token change (positive for gain, negative for deduction)
	Player2TokenChange int `json:"player2_token_change"` // Player 2 token change (positive for gain, negative for deduction)
	SystemFee          int `json:"system_fee"`           // System fee
}

// ScoreReward score reward
type ScoreReward struct {
	Player1ScoreChange int `json:"player1_score_change"` // Player 1 score change
	Player2ScoreChange int `json:"player2_score_change"` // Player 2 score change
}

// BattleReward battle reward
type BattleReward struct {
	TokenReward TokenReward `json:"token_reward"` // Token reward
	ScoreReward ScoreReward `json:"score_reward"` // Score reward
}
