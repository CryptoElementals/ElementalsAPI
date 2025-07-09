package dao

import "time"

// UserProfile 用户档案表 - 存储用户基本信息
type UserProfile struct {
	Address           string     `gorm:"type:varchar(42);primaryKey;not null"`
	Name              string     `gorm:"type:varchar(42);not null"`
	AvatarURL         string     `gorm:"type:varchar(50)"`
	Points            int        `gorm:"default:0"`
	TokenAmount       int        `gorm:"default:0"`
	OverallGame       int        `gorm:"default:0"`
	WinningRate       float64    `gorm:"default:0.0"`
	CollectedRewardAt *time.Time `gorm:"default:null"` // 记录用户领取每日奖励的时间
	CreatedAt         time.Time  `gorm:"autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (UserProfile) TableName() string {
	return "user_profiles"
}
