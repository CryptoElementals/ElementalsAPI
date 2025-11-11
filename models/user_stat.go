package dao

import "github.com/google/uuid"

type UserStat struct {
	BaseModel
	UserID             uuid.UUID `gorm:"type:char(36);uniqueIndex;not null" json:"user_id"`
	TotalGameCount     uint      `gorm:"default:0" json:"total_game_count"`
	WinCount           uint      `gorm:"default:0" json:"win_count"`
	LoseCount          uint      `gorm:"default:0" json:"lose_count"`
	TieCount           uint      `gorm:"default:0" json:"tie_count"`
	LastPlayerRewardID uint      `gorm:"default:0" json:"last_player_reward_id"`
}

// TableName 指定表名
func (UserStat) TableName() string {
	return "user_stats"
}
