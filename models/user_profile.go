package dao

import (
	"time"

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

// BeforeCreate 确保在创建记录时生成 UUID 主键
func (u *UserProfile) BeforeCreate(tx *gorm.DB) (err error) {
	if u.PlayerID == 0 {
		u.PlayerID = generateSnowflakeID()
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

// ---- Minimal Snowflake-like generator (local to avoid extra imports) ----
// 41 bits timestamp (ms), 10 bits node id (fixed 1), 12 bits sequence
var (
	_lastTs int64
	_seq    uint16
	_nodeID uint16 = 1
)

func nowMs() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func generateSnowflakeID() int64 {
	const epoch = int64(1762819200000) // 2025-11-11
	const nodeBits = 10
	const seqBits = 12
	const maxSeq = (1 << seqBits) - 1

	ts := nowMs()
	if ts == _lastTs {
		_seq = (_seq + 1) & uint16(maxSeq)
		if _seq == 0 {
			for ts <= _lastTs {
				ts = nowMs()
			}
		}
	} else {
		_seq = 0
	}
	_lastTs = ts
	return (int64(ts-epoch) << (nodeBits + seqBits)) |
		(int64(_nodeID) << seqBits) |
		int64(_seq)
}
