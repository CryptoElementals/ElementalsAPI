package dao

import "github.com/CryptoElementals/common/rpc/proto"

type GameArgs struct {
	MaxRounds         int64
	InitialHP         int64 `json:"initial_hp"`
	InitialMultiplier int64
	// timeouts
	ConfirmationTimeout         int64 `json:"confirmation_timeout"`          // Timeout for game match and round confirmation
	CommitmentSubmissionTimeout int64 `json:"commitment_submission_timeout"` // Timeout for commitment submission
	CardSubmissionTimeout       int64 `json:"card_submission_timeout"`       // Timeout for card submission
	GameContinueTimeout         int64 `json:"game_continue_timeout"`         // Timeout for game continue

	// timeout redundancy
	ConfirmationTimeoutRedundancy         int64 `json:"confirmation_timeout_redundancy"`          // Redundancy for game match and round confirmation
	CommitmentSubmissionTimeoutRedundancy int64 `json:"commitment_submission_timeout_redundancy"` // Redundancy for commitment submission
	CardSubmissionTimeoutRedundancy       int64 `json:"card_submission_timeout_redundancy"`       // Redundancy for card submission
	GameContinueTimeoutRedundancy         int64 `json:"game_continue_timeout_redundancy"`         // Redundancy for game continue

	// pool processing interval in seconds
	PoolProcessingInterval int64
}

type Game struct {
	BaseModel
	RoomContract string           `gorm:"index" json:"room_contract"` // 房间合约地址
	Type         uint             `gorm:"not null" json:"type"`       // 游戏模式
	Status       proto.GameStatus `gorm:"not null" json:"status"`

	Players    []*GamePlayerInfo `json:"players"`
	Rounds     []*Round          `json:"rounds"`
	GameResult *GameResult       `json:"game_result"`

	GameArgs
}

// Round 回合记录
type Round struct {
	BaseModel
	GameID         uint                      `json:"game_id"`      // 匹配唯一ID
	RoundNumber    uint32                    `json:"round_number"` // 回合数
	Turns          []*Turn                   `json:"turns"`
	IsLastRound    bool                      `json:"is_last_round"`
	CompleteReason proto.RoundCompleteReason `json:"complete_reason"`
}

type Turn struct {
	BaseModel
	RoundID         uint              `json:"round_id"`
	TurnNumber      uint32            `json:"turn_number"`
	TurnStartAt     int64             `json:"turn_start_at"`
	PlayerTurnInfos []*PlayerTurnInfo `json:"player_turn_infos"`
}

type PlayerTurnInfo struct {
	BaseModel
	TurnID            uint                   `json:"turn_id"`
	PlayerID          int64                  `json:"player_id"`
	PlayerStatus      proto.PlayerTurnStatus `json:"player_status"`
	TemporaryAddress  string                 `json:"temporary_address"`
	TurnSubmittedCard *TurnSubmittedCard     `gorm:"embedded" json:"turn_submitted_card"`
}

// Not a table
type TurnSubmittedCard struct {
	CommitmentHash []byte `json:"commitment_hash"`
	CardID         uint32 `json:"card_id"`
	Salt           []byte `json:"salt"`

	HealthBefore     uint32                `json:"health_before"`
	HealthAfter      uint32                `json:"health_after"`
	MultiplierBefore uint32                `json:"multiplier_before"`
	MultiplierAfter  uint32                `json:"multiplier_after"`
	Description      string                `json:"description"`
	ElementRelation  proto.ElementRelation `json:"element_relation"`
	CardEffects      []*CardEffect         `json:"card_effects"`
}

type CardEffect struct {
	BaseModel
	PlayerTurnInfoID       uint
	Type                   proto.BattleEffectType
	Value                  int32
	Description            string
	TargetPlayerId         int64
	TargetTemporaryAddress string
}

type GamePlayerInfo struct {
	BaseModel
	GameID           uint   `json:"game_id"`
	PlayerId         int64  `gorm:"not null;index:address" json:"player_id"`
	TemporaryAddress string `gorm:"not null;index:address" json:"temporary_address"`
}

type PlayerReward struct {
	BaseModel
	BattleRewardID         uint
	PlayerId               int64 `gorm:"not null;index:wallet_address" json:"player_id"`
	TemporaryAddress       string
	TokenChange            int32
	PointChange            int32
	PlayerGameResultStatus proto.PlayerGameResultStatus
	IsOffline              bool
	Surrendered            bool
}

type BattleReward struct {
	BaseModel
	GameResultID  uint
	SystemFee     int32
	PlayerRewards []*PlayerReward
}

type GameResult struct {
	BaseModel
	GameID                 uint
	Multiplier             int32
	WinnerPlayerId         int64
	WinnerTemporaryAddress string
	GameResultType         proto.GameResultType
	BattleReward           *BattleReward
}
