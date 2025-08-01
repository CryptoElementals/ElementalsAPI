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

// PlayerStatus 玩家状态
type PlayerStatus int32

const (
	PLAYER_ONLINE      PlayerStatus = 0
	PLAYER_OFFLINE     PlayerStatus = 1
	PLAYER_SURRENDERED PlayerStatus = 2
)

// RoundInput battle input parameters
type PlayerRoundInput struct {
	WalletAddress    string       `json:"WalletAddress"`
	TemporaryAddress string       `json:"TemporaryAddress"`
	Cards            []int        `json:"Cards"`
	HP               int          `json:"HP"`
	LostHP           int          `json:"LostHP"`
	Commitment       []byte       `json:"Commitment"`       // 本回合提交的承诺（为空表示未提交）
	Status           PlayerStatus `json:"Status,omitempty"` // 玩家状态，默认 online 为0
	Surrendered      bool         `json:"Surrendered"`      // 是否投降
}

type RoundInput struct {
	RoundNumber uint32             `json:"RoundNumber"`
	Players     []PlayerRoundInput `json:"Players"`
}

// BattleResult battle result
// GameResult 表示游戏结束时的结果数据（回合未结束时为 nil）
type GameResult struct {
	Multiplier             uint32         `json:"Multiplier"`             // 最终倍率
	WinnerWalletAddress    string         `json:"WinnerWalletAddress"`    // 胜者钱包地址（tie 时为 ""）
	WinnerTemporaryAddress string         `json:"WinnerTemporaryAddress"` // 胜者临时地址
	GameResultType         GameResultType `json:"GameResultType"`         // 游戏结果类型
	Reward                 BattleReward   `json:"Reward"`                 // 奖励信息
}

// RoundResult 表示单回合结果，若游戏已结束，则 GameResult 不为 nil
type RoundResult struct {
	Players     []PlayerRoundStat `json:"Players"`     // 所有玩家的回合数据
	RoundNumber uint32            `json:"RoundNumber"` // 回合号
	IsGameOver  bool              `json:"IsGameOver"`  // 是否游戏结束
	GameResult  *GameResult       `json:"GameResult"`  // 游戏结果（仅游戏结束时返回）
}

// ElementalRelation elemental relation
type ElementalRelation struct {
	P1Type        string `json:"P1Type"`        // P1's relation type
	P2Type        string `json:"P2Type"`        // P2's relation type
	P1Description string `json:"P1Description"` // Description from P1's perspective
	P2Description string `json:"P2Description"` // Description from P2's perspective
}

// BattleEffect battle effect
// 代表对自身产生的效果
type BattleEffect struct {
	Type                   BattleEffectType `json:"Type"`
	Value                  int              `json:"Value"`
	TargetWalletAddress    string           `json:"TargetWalletAddress"`
	TargetTemporaryAddress string           `json:"TargetTemporaryAddress"`
	Description            string           `json:"Description"`
}

// BattleReward battle reward
type BattleReward struct {
	PlayerRewards []PlayerReward `json:"PlayerRewards"` // 每个玩家的奖励变化
	SystemFee     int            `json:"SystemFee"`     // 系统手续费
}

// PlayerReward 单个玩家的奖励
type PlayerReward struct {
	WalletAddress    string `json:"WalletAddress"`    // 玩家地址
	TemporaryAddress string `json:"TemporaryAddress"` // 玩家临时地址
	TokenChange      int    `json:"TokenChange"`      // Token变化
	PointChange      int    `json:"PointChange"`      // 积分变化
	IsOffline        bool   `json:"IsOffline"`        // 是否离线
	IsSurrendered    bool   `json:"IsSurrendered"`    // 是否投降
}

// 单个玩家每张卡的详细数据
type PlayerCardStat struct {
	CardNumber       int             `json:"CardNumber"`
	CardID           int             `json:"CardID"`
	HPBefore         int             `json:"HPBefore"`
	HPAfter          int             `json:"HPAfter"`
	MultiplierBefore uint32          `json:"MultiplierBefore"`
	MultiplierAfter  uint32          `json:"MultiplierAfter"`
	Effects          []BattleEffect  `json:"Effects"`
	Description      string          `json:"Description"`
	ElementRelation  ElementRelation `json:"ElementRelation"`
}

// 单个玩家的所有卡数据
// PlayerRoundStat 表示每个玩家本轮的所有卡片数据
type PlayerRoundStat struct {
	WalletAddress    string           `json:"WalletAddress"`
	TemporaryAddress string           `json:"TemporaryAddress"`
	LostHP           int              `json:"LostHP"`
	CardStats        []PlayerCardStat `json:"CardStats"`
}

// ----------- 枚举同步 proto -----------

type ElementRelation int32

const (
	OVER_POWER ElementRelation = iota
	OVER_POWERED
	NURTURE
	NURTURED
	TIE
)

type GameResultType int32

const (
	GAME_NORMAL GameResultType = iota
	GAME_KO
	GAME_TIE
)

type BattleEffectType int32

const (
	ATTACK BattleEffectType = iota
	HEAL
)
