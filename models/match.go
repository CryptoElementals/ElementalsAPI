package dao

import "time"

// Match 匹配记录表
type Match struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`       // 自增主键ID
	MatchID   string    `gorm:"not null;index" json:"match_id"`           // 匹配唯一ID（两个用户共享）
	Address   string    `gorm:"not null;index" json:"address"`            // 玩家地址
	PublicKey string    `gorm:"not null" json:"public_key"`               // 玩家公钥
	Mode      string    `gorm:"not null;index" json:"mode"`               // 游戏模式
	Status    string    `gorm:"not null;default:'waiting'" json:"status"` // 状态: waiting, matched, confirmed, cancelled
	RoomID    string    `gorm:"default:''" json:"room_id"`                // 房间ID（匹配确认后生成）
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Match) TableName() string {
	return "matches"
}
