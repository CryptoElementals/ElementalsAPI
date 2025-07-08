package dao

import (
	"time"
)

// Room 游戏房间模型
type Room struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	RoomID      string    `gorm:"uniqueIndex;not null" json:"room_id"`      // 房间唯一ID
	Player1ID   string    `gorm:"not null" json:"player1_id"`               // 玩家1 ID
	Player2ID   string    `gorm:"not null" json:"player2_id"`               // 玩家2 ID
	Player1Addr string    `gorm:"not null" json:"player1_addr"`             // 玩家1 地址
	Player2Addr string    `gorm:"not null" json:"player2_addr"`             // 玩家2 地址
	Status      string    `gorm:"not null;default:'waiting'" json:"status"` // 房间状态: waiting, playing, finished
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Room) TableName() string {
	return "rooms"
}
