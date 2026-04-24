package dao

import (
	"time"

	"github.com/CryptoElementals/common/snowflake"
	"gorm.io/gorm"
)

const (
	GameMatchStatusPending     = "pending"
	GameMatchStatusCreating    = "creating" // transient: match claimed, game row being inserted
	GameMatchStatusGameCreated = "game_created"
	GameMatchStatusCancelled   = "cancelled"
)

// GameMatch is a pre-game pairing from PVP matchmaking; both players confirm before a games row exists.
type GameMatch struct {
	ID int64 `gorm:"column:id;type:bigint;primaryKey" json:"id"`

	Player1ID          int64      `gorm:"not null;index:idx_game_match_p1,priority:1" json:"player1_id"`
	Player1TempAddress string     `gorm:"not null;size:128;index:idx_game_match_p1,priority:2" json:"player1_temp_address"`
	Player2ID          int64      `gorm:"not null;index:idx_game_match_p2,priority:1" json:"player2_id"`
	Player2TempAddress string     `gorm:"not null;size:128;index:idx_game_match_p2,priority:2" json:"player2_temp_address"`
	Player1ConfirmedAt *time.Time `json:"player1_confirmed_at"`
	Player2ConfirmedAt *time.Time `json:"player2_confirmed_at"`
	GameType           uint       `gorm:"not null" json:"game_type"`
	LastGameID         int64      `gorm:"not null;default:0" json:"last_game_id"`
	Status             string     `gorm:"not null;size:32;index" json:"status"`
	GameID             *int64     `gorm:"index" json:"game_id"`
	CreatedAt          time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (GameMatch) TableName() string { return "game_match" }

// BeforeCreate assigns a snowflake primary key when missing.
func (m *GameMatch) BeforeCreate(tx *gorm.DB) error {
	if m.ID == 0 {
		m.ID = snowflake.GenerateID()
	}
	if m.Status == "" {
		m.Status = GameMatchStatusPending
	}
	return nil
}

// PlayerQueueEntry tracks PVP lobby queue membership (GameMatchID=0) or pending game_match linkage (GameMatchID>0).
type PlayerQueueEntry struct {
	BaseModel
	PlayerID    int64  `gorm:"not null;index:uniq_player_queue_player"`
	TempAddress string `gorm:"not null;size:128;index:uniq_player_queue_player"`
	GameMatchID int64  `gorm:"not null;default:0;index"`
}

func (PlayerQueueEntry) TableName() string { return "player_queue_entries" }

// PlayerGameEntry records players currently in an active game (post-confirmation).
type PlayerGameEntry struct {
	BaseModel
	PlayerID    int64  `gorm:"not null;index:uniq_player_game_player"`
	TempAddress string `gorm:"not null;size:128;index:uniq_player_game_player"`
	GameID      int64  `gorm:"not null;index"`
}

func (PlayerGameEntry) TableName() string { return "player_game_entries" }
