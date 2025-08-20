package dao

import "time"

// UserProfile 用户档案表 - 存储用户基本信息
type UserProfile struct {
	Address           string     `gorm:"type:varchar(42);primaryKey;not null" json:"address"`
	Name              string     `gorm:"type:varchar(42);not null" json:"name"`
	AvatarURL         string     `gorm:"type:varchar(200)" json:"avatar_url"`
	BackgroundURL     string     `gorm:"type:varchar(200)" json:"background_url"`
	OverallGame       int        `gorm:"default:0" json:"overall_game"`
	WinCount          int        `gorm:"default:0" json:"win_count"`
	WinningRate       float64    `gorm:"default:0.0" json:"winning_rate"`
	CollectedRewardAt *time.Time `gorm:"default:null" json:"collected_reward_at"` // 记录用户领取每日奖励的时间
	IsBot             bool       `gorm:"is_bot,index" json:"is_bot"`
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

type DevTempKey struct {
	ID             uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	TempPrivateKey string `gorm:"type:varchar(100);not null" json:"temp_private_key"`
	TempAddress    string `gorm:"type:varchar(100);not null" json:"temp_address"`
	Address        string `gorm:"type:varchar(100)" json:"address"`
}

// TableName 指定表名
func (UserProfile) TableName() string {
	return "user_profiles"
}

func (DevTempKey) TableName() string {
	return "dev_temp_keys"
}
