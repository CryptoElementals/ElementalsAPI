package battle

// Card card structure
type Card struct {
	ID          int    `json:"ID"`          // Card ID
	ElementType string `json:"ElementType"` // Element type (Metal, Wood, Water, Fire, Earth)
	Level       string `json:"Level"`       // Card level (legendary, epic, rare, normal)
	LifeForce   int    `json:"LifeForce"`   // Life force
	Attack      int    `json:"Attack"`      // Attack power
	Defense     int    `json:"Defense"`     // Defense power
	Name        string `json:"Name"`        // Card name
	Description string `json:"Description"` // Card description
}

// Player player structure
type Player struct {
	Address    string  `json:"Address"`    // Player address
	Cards      []Card  `json:"Cards"`      // Player's cards
	HP         int     `json:"HP"`         // Health points
	Multiplier float64 `json:"Multiplier"` // Score multiplier
}

// BattleInput battle input parameters
type BattleInput struct {
	Player1Address    string  `json:"Player1Address"`    // Player 1 address
	Player2Address    string  `json:"Player2Address"`    // Player 2 address
	Player1HP         int     `json:"Player1HP"`         // Player 1 initial health
	Player2HP         int     `json:"Player2HP"`         // Player 2 initial health
	Player1Multiplier float64 `json:"Player1Multiplier"` // Player 1 score multiplier
	Player2Multiplier float64 `json:"Player2Multiplier"` // Player 2 score multiplier
	Player1Cards      []int   `json:"Player1Cards"`      // Player 1 card ID list
	Player2Cards      []int   `json:"Player2Cards"`      // Player 2 card ID list
	Player1LostHP     int     `json:"Player1LostHP"`     // Player 1 accumulated lost health
	Player2LostHP     int     `json:"Player2LostHP"`     // Player 2 accumulated lost health
}

// BattleResult battle result
type BattleResult struct {
	Player1Address         string        `json:"Player1Address"`         // Player 1 address
	Player2Address         string        `json:"Player2Address"`         // Player 2 address
	Stage                  int           `json:"Stage"`                  // Stage number
	Rounds                 []RoundResult `json:"Rounds"`                 // Round results
	Player1FinalHP         int           `json:"Player1FinalHP"`         // Player 1 final health
	Player2FinalHP         int           `json:"Player2FinalHP"`         // Player 2 final health
	Player1LostHP          int           `json:"Player1LostHP"`          // Player 1 accumulated lost health
	Player2LostHP          int           `json:"Player2LostHP"`          // Player 2 accumulated lost health
	Player1FinalMultiplier float64       `json:"Player1FinalMultiplier"` // Player 1 final multiplier
	Player2FinalMultiplier float64       `json:"Player2FinalMultiplier"` // Player 2 final multiplier
	GameFinalMultiplier    float64       `json:"GameFinalMultiplier"`    // Game final multiplier (take loser's multiplier, tie is 1)
	Winner                 string        `json:"Winner"`                 // Winner address
	IsGameOver             bool          `json:"IsGameOver"`             // Whether game is over
	GameResultType         string        `json:"GameResultType"`         // Game result type
	Reward                 *BattleReward `json:"Reward"`                 // Battle reward (token and point)
}

// RoundResult round result
type RoundResult struct {
	RoundNumber            int            `json:"RoundNumber"`            // Round number
	Player1CardID          int            `json:"Player1CardID"`          // Player 1 used card ID
	Player2CardID          int            `json:"Player2CardID"`          // Player 2 used card ID
	RelationType           string         `json:"RelationType"`           // Elemental relation type
	Actions                []BattleAction `json:"Actions"`                // Executed action list
	Player1HPDelta         int            `json:"Player1HPDelta"`         // Player 1 HP change (负值为受伤，正值为治疗)
	Player2HPDelta         int            `json:"Player2HPDelta"`         // Player 2 HP change (负值为受伤，正值为治疗)
	Player1HPAfter         int            `json:"Player1HPAfter"`         // Player 1 health after round
	Player2HPAfter         int            `json:"Player2HPAfter"`         // Player 2 health after round
	Player1MultiplierAfter float64        `json:"Player1MultiplierAfter"` // Player 1 multiplier after round
	Player2MultiplierAfter float64        `json:"Player2MultiplierAfter"` // Player 2 multiplier after round
	Description            string         `json:"Description"`            // Round description
}

// ElementalRelation elemental relation
type ElementalRelation struct {
	Type        string `json:"Type"`        // Relation type: "overpower"(overpower), "overpowered"(overpowered), "nurture"(nurture), "nurtured"(nurtured), "even"(even)
	Description string `json:"Description"` // Relation description
}

// BattleAction battle action
type BattleAction struct {
	Type        string `json:"Type"`        // Action type: "attack", "heal"
	Target      string `json:"Target"`      // Target: "player1", "player2"
	Value       int    `json:"Value"`       // Action value
	Description string `json:"Description"` // Action description
}

// BattleReward battle reward
type BattleReward struct {
	Player1TokenChange int `json:"Player1TokenChange"`
	Player2TokenChange int `json:"Player2TokenChange"`
	SystemFee          int `json:"SystemFee"`
	Player1PointChange int `json:"Player1PointChange"`
	Player2PointChange int `json:"Player2PointChange"`
}
