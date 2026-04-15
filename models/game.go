package dao

import (
	"time"

	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

// HPDiffPerMultiplierUnit maps raw HP spread to the integer multiplier (rewards, APIs).
const HPDiffPerMultiplierUnit int64 = 1000

// MultiplierFromHPSpread returns spread / HPDiffPerMultiplierUnit for spread >= 0.
func MultiplierFromHPSpread(spread int64) uint32 {
	if spread < 0 {
		return 0
	}
	return uint32(spread / HPDiffPerMultiplierUnit)
}

type GameArgs struct {
	BaseModel `validate:"-"`

	// MaxNormalRounds maps to legacy column max_rounds.
	MaxNormalRounds int64 `validate:"gt=0" gorm:"column:max_rounds;not null;default:3" json:"max_normal_rounds"`
	// MaxExtraRounds / MaxTurnsPerExtraRound are unused for match layout: overtime exists only for tournament (see Game.OvertimeRoundsCap).
	MaxExtraRounds int64 `validate:"gte=0" gorm:"not null;default:0" json:"max_extra_rounds"`
	// MaxTurnsPerNormalRound maps to legacy column max_turns_per_round.
	MaxTurnsPerNormalRound int64 `validate:"gt=0" gorm:"column:max_turns_per_round;not null;default:3" json:"max_turns_per_normal_round"`
	MaxTurnsPerExtraRound  int64 `validate:"gte=0" gorm:"not null;default:0" json:"max_turns_per_extra_round"`

	InitialHP int64 `validate:"gt=0" gorm:"not null" json:"initial_hp"`
	MaxHP     int64 `validate:"gte=0" gorm:"not null" json:"max_hp"`
	// BaseStake is copied from the room template row into each match; lobby settlement uses this (not config).
	BaseStake int64 `validate:"gt=0" gorm:"not null;default:1000" json:"base_stake"`

	// timeouts
	ConfirmationTimeout         int64 `validate:"gt=0" gorm:"not null" json:"confirmation_timeout"`          // Timeout for game match and round confirmation
	CommitmentSubmissionTimeout int64 `validate:"gt=0" gorm:"not null" json:"commitment_submission_timeout"` // Timeout for commitment submission
	CardSubmissionTimeout       int64 `validate:"gt=0" gorm:"not null" json:"card_submission_timeout"`       // Timeout for card submission
	GameContinueTimeout         int64 `validate:"gt=0" gorm:"not null" json:"game_continue_timeout"`         // Timeout for game continue

	// timeout redundancy
	ConfirmationTimeoutRedundancy         int64 `validate:"gte=0" gorm:"not null" json:"confirmation_timeout_redundancy"`          // Redundancy for game match and round confirmation
	CommitmentSubmissionTimeoutRedundancy int64 `validate:"gte=0" gorm:"not null" json:"commitment_submission_timeout_redundancy"` // Redundancy for commitment submission
	CardSubmissionTimeoutRedundancy       int64 `validate:"gte=0" gorm:"not null" json:"card_submission_timeout_redundancy"`       // Redundancy for card submission
	GameContinueTimeoutRedundancy         int64 `validate:"gte=0" gorm:"not null" json:"game_continue_timeout_redundancy"`         // Redundancy for game continue
}

func (GameArgs) TableName() string { return "game_args" }

type Game struct {
	ID        int64 `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	GameArgsID uint `gorm:"not null;index" json:"game_args_id"`

	RoomContract string `gorm:"index" json:"room_contract"` // 房间合约地址
	Type         uint   `gorm:"not null" json:"type"`       // 游戏模式
	// RegulationRounds is the count of full regulation rounds (e.g. tournament); 0 means infer from GameArgs for older rows.
	RegulationRounds uint32 `gorm:"not null;default:0" json:"regulation_rounds"`
	// OvertimeRoundsCap is the max tie-breaker rounds after regulation for tournament matches (1 turn each).
	OvertimeRoundsCap uint32           `gorm:"not null;default:0" json:"overtime_rounds_cap"`
	Status            proto.GameStatus `gorm:"not null" json:"status"`
	// QueueMatchID is the game_match snowflake when this game was created after queue ConfirmMatch; 0 otherwise.
	QueueMatchID int64 `gorm:"not null;default:0" json:"queue_match_id"`

	Players    []*GamePlayerInfo `json:"players"`
	Turns      []*Turn           `json:"turns"`
	GameResult *GameResult       `json:"game_result"`

	GameArgs *GameArgs `json:"game_args,omitempty"`
}

type GameChainID struct {
	BaseModel
	GameID  int64 `gorm:"not null;uniqueIndex:uq_game_chain_game" json:"game_id"`
	ChainID int64 `gorm:"not null" json:"chain_id"`
}

func (GameChainID) TableName() string { return "game_chain_ids" }

type Turn struct {
	BaseModel
	GameID      int64  `gorm:"not null;uniqueIndex:uq_game_turn,priority:1" json:"game_id"`
	RoundNumber uint32 `gorm:"not null;uniqueIndex:uq_game_turn,priority:2" json:"round_number"`
	TurnNumber  uint32 `gorm:"not null;uniqueIndex:uq_game_turn,priority:3" json:"turn_number"`

	TurnStatus  uint32 `gorm:"not null" json:"turn_status"`
	TurnStartAt int64  `gorm:"not null" json:"turn_start_at"`

	PlayerTurnInfos []*PlayerTurnInfo `json:"player_turn_infos"`
}

func (Turn) TableName() string { return "turns" }

type PlayerTurnInfo struct {
	BaseModel
	TurnID            uint                   `gorm:"not null;index;uniqueIndex:uq_turn_player,priority:1" json:"turn_id"`
	PlayerID          int64                  `json:"player_id"`
	PlayerStatus      proto.PlayerTurnStatus `json:"player_status"`
	TemporaryAddress  string                 `gorm:"not null;size:128;uniqueIndex:uq_turn_player,priority:2" json:"temporary_address"`
	TurnSubmittedCard *TurnSubmittedCard     `gorm:"embedded" json:"turn_submitted_card"`
}

// Not a table
type TurnSubmittedCard struct {
	CommitmentHash []byte `json:"commitment_hash"`
	CardID         uint32 `json:"card_id"`
	Salt           []byte `json:"salt"`

	HealthBefore    uint32                `json:"health_before"`
	HealthAfter     uint32                `json:"health_after"`
	Description     string                `json:"description"`
	ElementRelation proto.ElementRelation `json:"element_relation"`
}

type GamePlayerInfo struct {
	BaseModel
	GameID           int64  `gorm:"index" json:"game_id"`
	PlayerId         int64  `gorm:"not null;index:address" json:"player_id"`
	TemporaryAddress string `gorm:"not null;index:address" json:"temporary_address"`
}

type PlayerReward struct {
	BaseModel
	BattleRewardID uint  `gorm:"index"`
	PlayerId       int64 `gorm:"not null;index:wallet_address" json:"player_id"`
	// TokenChange and PointChange are computed in lobby settlement (see battlereward.ComputeBattleRewardAmounts); room persists zeros until then.
	TokenChange int32
	PointChange int32
}

type BattleRewardPVP struct {
	BaseModel
	GameID        int64 `gorm:"index"`
	SystemFee     int32
	PlayerRewards []*PlayerReward `gorm:"foreignKey:BattleRewardID"`
}

// TableName keeps the legacy table name used when the type was BattleReward.
func (BattleRewardPVP) TableName() string { return "battle_rewards" }

type PlayerResultInfo struct {
	BaseModel
	GameResultID           uint  `gorm:"index"`
	PlayerId               int64 `gorm:"not null;index:idx_player_result_player_id" json:"player_id"`
	TemporaryAddress       string
	IsWinner               bool
	PlayerGameResultStatus proto.PlayerGameResultStatus
}

type GameResult struct {
	BaseModel
	GameID            int64 `gorm:"index"`
	GameType          proto.GameType
	Multiplier        int32
	GameResultType    proto.GameResultType
	PlayerResultInfos []*PlayerResultInfo `gorm:"foreignKey:GameResultID"`
}
