package dao

import (
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
	"gorm.io/gorm"
)

// UserProfile 用户档案表 - 存储用户基本信息
type UserProfile struct {
	PlayerID          int64      `gorm:"column:player_id;type:bigint;primaryKey" json:"player_id"`
	Address           string     `gorm:"type:varchar(100)" json:"address"`
	Email             string     `gorm:"type:varchar(200)" json:"email"`
	Name              string     `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	AvatarURL         string     `gorm:"type:varchar(200)" json:"avatar_url"`
	BackgroundURL     string     `gorm:"type:varchar(200)" json:"background_url"`
	CollectedRewardAt *time.Time `gorm:"default:null" json:"collected_reward_at"` // 记录用户领取每日奖励的时间
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// BeforeCreate 确保在创建记录时生成主键
func (u *UserProfile) BeforeCreate(tx *gorm.DB) (err error) {
	if u.PlayerID == 0 {
		u.PlayerID = GenerateSnowflakeID()
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

// snowflakeNode 全局雪花ID生成器节点
var (
	snowflakeNode *snowflake.Node
	snowflakeOnce sync.Once
)

// initSnowflakeNode 初始化雪花ID生成器节点
func initSnowflakeNode() {
	snowflakeOnce.Do(func() {
		var err error
		// 使用节点ID 1
		snowflakeNode, err = snowflake.NewNode(1)
		if err != nil {
			log.Fatalf("初始化雪花ID生成器失败: %v", err)
		}
	})
}

// GenerateSnowflakeID returns a new snowflake id (shared by user_profiles, game_match, etc.).
func GenerateSnowflakeID() int64 {
	initSnowflakeNode()
	return snowflakeNode.Generate().Int64()
}
