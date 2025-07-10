package dao

import "time"

// Match 匹配记录表
type Match struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`               // 自增主键ID
	MatchID   string    `gorm:"not null;index;size:64" json:"match_id"`           // 匹配唯一ID (UUID长度为36，预留一些空间)
	Address   string    `gorm:"not null;index;size:42" json:"address"`            // 玩家地址 (以太坊地址长度为42)
	PublicKey string    `gorm:"not null;size:130" json:"public_key"`              // 玩家公钥 (ECDSA公钥长度为130字符)
	Mode      string    `gorm:"not null;index;size:20" json:"mode"`               // 游戏模式
	Status    string    `gorm:"not null;default:'waiting';size:20" json:"status"` // 状态: waiting, matched, confirmed, cancelled
	RoomID    string    `gorm:"default:'';index;size:64" json:"room_id"`          // 房间ID (UUID长度为36，预留一些空间)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Match) TableName() string {
	return "matches"
}
