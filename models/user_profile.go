package dao

import (
	"time"

	"github.com/CryptoElementals/common/snowflake"
	"gorm.io/gorm"
)

// UserProfile 用户档案表 - 存储用户基本信息
type UserProfile struct {
	PlayerID                 int64      `gorm:"column:player_id;type:bigint;primaryKey" json:"player_id"`
	Address                  string     `gorm:"type:varchar(100)" json:"address"`
	Email                    string     `gorm:"type:varchar(200)" json:"email"`
	Name                     string     `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	AvatarURL                string     `gorm:"type:varchar(200)" json:"avatar_url"`
	BackgroundURL            string     `gorm:"type:varchar(200)" json:"background_url"`
	CollectedRewardAt        *time.Time `gorm:"default:null" json:"collected_reward_at"`          // 记录用户领取每日奖励的时间
	CollectedNewUserRewardAt *time.Time `gorm:"default:null" json:"collected_new_user_reward_at"` // 记录用户领取新手奖励的时间
	ServerType               string     `gorm:"column:server_type;type:varchar(16);not null;default:trial" json:"server_type"`
	CreatedAt                time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt                time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// BeforeCreate 确保在创建记录时生成主键
func (u *UserProfile) BeforeCreate(tx *gorm.DB) (err error) {
	if u.PlayerID == 0 {
		u.PlayerID = snowflake.GenerateID()
	}
	return nil
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
