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

// RoundInput battle input parameters
// 支持多玩家
// PlayerRoundInput 表示每个玩家的输入
type PlayerRoundInput struct {
	Address    string  `json:"Address"`
	Cards      []int   `json:"Cards"`
	HP         int     `json:"HP"`
	Multiplier float64 `json:"Multiplier"`
	LostHP     int     `json:"LostHP"`
}

type RoundInput struct {
	Players []PlayerRoundInput `json:"Players"`
}

// BattleResult battle result
type RoundResult struct {
	Players             []PlayerRoundStat `json:"Players"`             // 所有玩家的回合数据
	Round               uint              `json:"Round"`               // Round number
	GameFinalMultiplier float64           `json:"GameFinalMultiplier"` // Game final multiplier (take loser's multiplier, tie is 1)
	Winner              string            `json:"Winner"`              // Winner address
	IsGameOver          bool              `json:"IsGameOver"`          // Whether game is over
	GameResultType      string            `json:"GameResultType"`      // Game result type
	Reward              *BattleReward     `json:"Reward"`              // Battle reward (token and point)
}

type FightResult struct {
	FightNumber            int            `json:"FightNumber"`            // Fight number
	Player1CardID          int            `json:"Player1CardID"`          // Player 1 used card ID
	Player2CardID          int            `json:"Player2CardID"`          // Player 2 used card ID
	RelationType           string         `json:"RelationType"`           // Elemental relation type
	Actions                []BattleEffect `json:"Actions"`                // Executed action list
	Player1HPDelta         int            `json:"Player1HPDelta"`         // Player 1 HP change (负值为受伤，正值为治疗)
	Player2HPDelta         int            `json:"Player2HPDelta"`         // Player 2 HP change (负值为受伤，正值为治疗)
	Player1HPAfter         int            `json:"Player1HPAfter"`         // Player 1 health after fight
	Player2HPAfter         int            `json:"Player2HPAfter"`         // Player 2 health after fight
	Player1MultiplierAfter float64        `json:"Player1MultiplierAfter"` // Player 1 multiplier after fight
	Player2MultiplierAfter float64        `json:"Player2MultiplierAfter"` // Player 2 multiplier after fight
	Description            string         `json:"Description"`            // Fight description
}

// ElementalRelation elemental relation
type ElementalRelation struct {
	Type        string `json:"Type"`        // Relation type: "overpower"(overpower), "overpowered"(overpowered), "nurture"(nurture), "nurtured"(nurtured), "even"(even)
	Description string `json:"Description"` // Relation description
}

// BattleEffect battle effect
// 代表对自身产生的效果
type BattleEffect struct {
	Type        string `json:"Type"`
	Value       int    `json:"Value"`
	Description string `json:"Description"`
	Target      string `json:"Target"`
}

// BattleReward battle reward
type BattleReward struct {
	PlayerRewards []PlayerReward `json:"PlayerRewards"` // 每个玩家的奖励变化
	SystemFee     int            `json:"SystemFee"`     // 系统手续费
}

// PlayerReward 单个玩家的奖励
type PlayerReward struct {
	PlayerAddress string `json:"PlayerAddress"` // 玩家地址
	TokenChange   int    `json:"TokenChange"`   // Token变化
	PointChange   int    `json:"PointChange"`   // 积分变化
}

// 单个玩家每张卡的详细数据
type PlayerCardStat struct {
	CardNumber       int            `json:"CardNumber"`
	CardID           int            `json:"CardID"`
	HPBefore         int            `json:"HPBefore"`
	HPAfter          int            `json:"HPAfter"`
	MultiplierBefore float64        `json:"MultiplierBefore"`
	MultiplierAfter  float64        `json:"MultiplierAfter"`
	Effects          []BattleEffect `json:"Effects"`
	Description      string         `json:"Description"`
	ElementRelation  string         `json:"elementRelation"`
}

// 单个玩家的所有卡数据
// PlayerRoundStat 表示每个玩家本轮的所有卡片数据
type PlayerRoundStat struct {
	PlayerAddress string           `json:"PlayerAddress"`
	CardStats     []PlayerCardStat `json:"CardStats"`
}
