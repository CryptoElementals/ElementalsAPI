package dao

import "time"

// Room 房间表
type Room struct {
	ID      uint   `gorm:"primaryKey;autoIncrement" json:"id"`       // 自增主键ID
	RoomID  string `gorm:"not null;index;size:64" json:"room_id"`    // 房间唯一ID (UUID长度为36，预留一些空间)
	Address string `gorm:"not null;index;size:42" json:"address"`    // 玩家地址 (以太坊地址长度为42)
	Stage   uint   `gorm:"not null;default:0" json:"stage"`          // 游戏阶段 (0,1,2,3)
	Cards   string `gorm:"not null;default:'';size:50" json:"cards"` // 卡牌标记，例如JMSHT

	// 新增字段：支持单个阶段对战模拟
	PlayerHP    int     `gorm:"not null;default:3000" json:"player_hp"`      // 玩家血量
	Multiplier  float64 `gorm:"not null;default:1.0" json:"multiplier"`      // 积分倍率
	IsStageOver bool    `gorm:"not null;default:false" json:"is_stage_over"` // 当前阶段是否处理完成

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Room) TableName() string {
	return "rooms"
}
