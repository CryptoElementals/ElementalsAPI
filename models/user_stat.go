package dao

type UserStat struct {
	BaseModel
	PlayerID           int64 `gorm:"column:player_id;type:bigint;uniqueIndex;not null" json:"player_id"`
	TotalGameCount     uint  `gorm:"default:0" json:"total_game_count"`
	WinCount           uint  `gorm:"default:0" json:"win_count"`
	LoseCount          uint  `gorm:"default:0" json:"lose_count"`
	TieCount           uint  `gorm:"default:0" json:"tie_count"`
	LastPlayerRewardID uint  `gorm:"default:0;index" json:"last_player_reward_id"`
}

// TableName 指定表名
func (UserStat) TableName() string {
	return "user_stats"
}
