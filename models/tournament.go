package dao

import "time"

// TournamentStatus aligns with docs/design/tournament-schema.md.
type TournamentStatus string

const (
	TournamentStatusRegistrationOpen TournamentStatus = "registration_open"
	TournamentStatusInProgress       TournamentStatus = "in_progress"
	TournamentStatusFinished         TournamentStatus = "finished"
	TournamentStatusCancelled        TournamentStatus = "cancelled"
)

type Tournament struct {
	ID                   uint             `gorm:"primarykey"`
	TournamentID         string           `gorm:"not null;uniqueIndex;size:64" json:"tournament_id"`
	Status               TournamentStatus `gorm:"type:varchar(32);not null;index" json:"status"`
	ScheduledStartAt     time.Time        `gorm:"not null;uniqueIndex" json:"scheduled_start_at"`
	ScheduledEndDeadline time.Time        `gorm:"not null;index" json:"scheduled_end_deadline"`
	RegistrationDeadline time.Time        `gorm:"not null;index" json:"registration_deadline"`
	EntryFee             int32            `gorm:"default:0" json:"entry_fee"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Tournament) TableName() string { return "tournaments" }

type TournamentParticipantStatus string

const (
	TournamentParticipantStatusQueued          TournamentParticipantStatus = "queued"
	TournamentParticipantStatusKickedNotEnough TournamentParticipantStatus = "kicked_not_enough"
	TournamentParticipantStatusKickedOverflow  TournamentParticipantStatus = "kicked_overflow"
	TournamentParticipantStatusInProgress      TournamentParticipantStatus = "in_progress"
	TournamentParticipantStatusEliminated      TournamentParticipantStatus = "eliminated"
	TournamentParticipantStatusChampion        TournamentParticipantStatus = "champion"
)

type TournamentParticipant struct {
	ID           uint                        `gorm:"primarykey"`
	TournamentID string                      `gorm:"not null;uniqueIndex:uq_tournament_player,priority:1;index" json:"tournament_id"`
	PlayerID     int64                       `gorm:"not null;uniqueIndex:uq_tournament_player,priority:2;index" json:"player_id"`
	TempAddress  string                      `gorm:"not null;size:128;uniqueIndex:uq_tournament_player,priority:3" json:"temp_address"`
	Status       TournamentParticipantStatus `gorm:"type:varchar(32);not null;index" json:"status"`
	CreatedAt    time.Time                   `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time                   `gorm:"autoUpdateTime" json:"updated_at"`
}

func (TournamentParticipant) TableName() string { return "tournament_participants" }

type TournamentRoundStatus string

const (
	TournamentRoundStatusMatched   TournamentRoundStatus = "matched"
	TournamentRoundStatusPlaying   TournamentRoundStatus = "playing"
	TournamentRoundStatusCompleted TournamentRoundStatus = "completed"
)

type TournamentRound struct {
	BaseModel

	TournamentID string                `gorm:"not null;uniqueIndex:uq_tournament_round,priority:1;index" json:"tournament_id"`
	RoundNo      uint32                `gorm:"not null;uniqueIndex:uq_tournament_round,priority:2" json:"round_no"`
	Status       TournamentRoundStatus `gorm:"type:varchar(32);not null;index" json:"status"`
}

func (TournamentRound) TableName() string { return "tournament_rounds" }

type TournamentMatchStatus string

const (
	TournamentMatchStatusMatched   TournamentMatchStatus = "matched"
	TournamentMatchStatusPlaying   TournamentMatchStatus = "playing"
	TournamentMatchStatusCompleted TournamentMatchStatus = "completed"
	TournamentMatchStatusBye       TournamentMatchStatus = "bye"
)

type TournamentMatch struct {
	BaseModel

	TournamentID string `gorm:"not null;uniqueIndex:uq_tournament_round_match,priority:1;index" json:"tournament_id"`
	RoundNo      uint32 `gorm:"not null;uniqueIndex:uq_tournament_round_match,priority:2" json:"round_no"`
	MatchNo      uint32 `gorm:"not null;uniqueIndex:uq_tournament_round_match,priority:3" json:"match_no"`

	Player1ID          int64                 `gorm:"not null;index" json:"player_1_id"`
	Player1TempAddress string                `gorm:"not null;size:128;index" json:"player_1_temp_address"`
	Player2ID          int64                 `gorm:"not null;index" json:"player_2_id"`
	Player2TempAddress string                `gorm:"not null;size:128;index" json:"player_2_temp_address"`
	WinnerPlayerID     *int64                `gorm:"index" json:"winner_player_id,omitempty"`
	WinnerTempAddress  *string               `gorm:"size:128;index" json:"winner_temp_address,omitempty"`
	GameID             *int64                `gorm:"index" json:"game_id,omitempty"`
	Status             TournamentMatchStatus `gorm:"type:varchar(32);not null;default:'matched';index" json:"status"`
}

func (TournamentMatch) TableName() string { return "tournament_matches" }

// TournamentReward records per-match token/point deltas for bracket play (separate from PVP player_rewards / battle_rewards).
type TournamentReward struct {
	BaseModel

	TournamentID string `gorm:"not null;size:64;uniqueIndex:uq_tournament_reward_player,priority:1;index" json:"tournament_id"`
	RoundNo      uint32 `gorm:"not null;uniqueIndex:uq_tournament_reward_player,priority:2" json:"round_no"`
	MatchNo      uint32 `gorm:"not null;uniqueIndex:uq_tournament_reward_player,priority:3" json:"match_no"`
	PlayerID     int64  `gorm:"not null;uniqueIndex:uq_tournament_reward_player,priority:4;index" json:"player_id"`
	TempAddress  string `gorm:"not null;size:128;uniqueIndex:uq_tournament_reward_player,priority:5;index" json:"temp_address"`
	TokenChange  int32  `gorm:"not null" json:"token_change"`
	PointChange  int32  `gorm:"not null" json:"point_change"`
}

func (TournamentReward) TableName() string { return "tournament_rewards" }

type TournamentEntryLedgerDirection string

const (
	TournamentEntryLedgerDirectionEntryDeduct TournamentEntryLedgerDirection = "entry_deduct"
	TournamentEntryLedgerDirectionEntryRefund TournamentEntryLedgerDirection = "entry_refund"
)

type TournamentEntryLedger struct {
	BaseModel

	TournamentID string                         `gorm:"not null;size:64;index;uniqueIndex:uq_tournament_entry_ledger,priority:1" json:"tournament_id"`
	PlayerID     int64                          `gorm:"not null;index;uniqueIndex:uq_tournament_entry_ledger,priority:2" json:"player_id"`
	TempAddress  string                         `gorm:"not null;size:128;index;uniqueIndex:uq_tournament_entry_ledger,priority:3" json:"temp_address"`
	Amount       int32                          `gorm:"not null" json:"amount"`
	Direction    TournamentEntryLedgerDirection `gorm:"type:varchar(16);not null;index;uniqueIndex:uq_tournament_entry_ledger,priority:4" json:"direction"`
	Reason       string                         `gorm:"type:varchar(64);not null;index;uniqueIndex:uq_tournament_entry_ledger,priority:5" json:"reason"`
}

func (TournamentEntryLedger) TableName() string { return "tournament_entry_ledgers" }

// TournamentTierRewardConfig stores tournament reward/point templates by tier.
type TournamentTierRewardConfig struct {
	BaseModel

	TotalPlayerCount int32 `gorm:"not null;index;uniqueIndex:uq_tournament_tier_reward_cfg,priority:1" json:"total_player_count"`
	EntryFee         int32 `gorm:"not null;default:0;uniqueIndex:uq_tournament_tier_reward_cfg,priority:2" json:"entry_fee"`
	Commission       int32 `gorm:"not null;default:0" json:"commission"`
	TotalTierCount   int32 `gorm:"not null" json:"total_tier_count"`
	TierNo           int32 `gorm:"not null;index;uniqueIndex:uq_tournament_tier_reward_cfg,priority:3" json:"tier_no"`
	RewardToken      int32 `gorm:"not null;default:0" json:"reward_token"`
	Point            int32 `gorm:"not null;default:0" json:"point"`
}

func (TournamentTierRewardConfig) TableName() string { return "tournament_tier_reward_configs" }
