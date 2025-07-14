package dao

import "time"

// Match 匹配记录表（一行记录包含两个玩家）
type Match struct {
	ID      uint   `gorm:"primaryKey;autoIncrement" json:"id"`      // 自增主键ID
	MatchID string `gorm:"not null;index;size:64" json:"match_id"`  // 匹配唯一ID (UUID长度为36，预留一些空间)
	Mode    string `gorm:"not null;index;size:20" json:"mode"`      // 游戏模式
	RoomID  string `gorm:"default:'';index;size:64" json:"room_id"` // 房间ID (UUID长度为36，预留一些空间)

	// 玩家1信息
	Player1Address     string `gorm:"not null;index;size:42" json:"player1_address"`            // 玩家1地址 (以太坊地址长度为42)
	Player1TempAddress string `gorm:"not null;size:42" json:"player1_temp_address"`             // 玩家1临时地址
	Player1Status      string `gorm:"not null;default:'waiting';size:20" json:"player1_status"` // 玩家1状态: waiting, matched, confirmed, cancelled

	// 玩家2信息
	Player2Address     string `gorm:"not null;index;size:42" json:"player2_address"`            // 玩家2地址 (以太坊地址长度为42)
	Player2TempAddress string `gorm:"not null;size:42" json:"player2_temp_address"`             // 玩家2临时地址
	Player2Status      string `gorm:"not null;default:'waiting';size:20" json:"player2_status"` // 玩家2状态: waiting, matched, confirmed, cancelled

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Match) TableName() string {
	return "matches"
}
