package dao

type Match struct {
	BaseModel
	RoomContract string        `gorm:"default:''" json:"room_contract"` // 房间合约地址
	Mode         string        `gorm:"not null;index" json:"mode"`      // 游戏模式
	Status       uint          `gorm:"not null;default:''" json:"status"`
	Players      []MatchPlayer `gorm:"not null;default:''" json:"players"`
	Rounds       []Round       `gorm:"not null;default:''" json:"rounds"`
}

func (Match) TableName() string {
	return "matches"
}

type Player struct {
	BaseModel
	Address       string `gorm:"not null;index" json:"address"`
	TempPublicKey string `gorm:"not null;default:''" json:"temp_public_key"`
}

// MatchPlayer 匹配记录表
type MatchPlayer struct {
	BaseModel
	MatchID  uint   `gorm:"not null;" json:"match_id"`  // 匹配唯一ID（两个用户共享）
	PlayerID uint   `gorm:"not null;" json:"player_id"` // 玩家ID
	Player   Player `gorm:"not null;" json:"player"`
	Status   string `gorm:"not null;default:'waiting'" json:"status"` // 状态: waiting, matched, confirmed, cancelled
}

// TableName 指定表名
func (MatchPlayer) TableName() string {
	return "matche_players"
}

// Round 回合记录
type Round struct {
	BaseModel
	MatchID      uint          `gorm:"not null;index" json:"match_id"`           // 匹配唯一ID
	RoundNumber  int           `gorm:"not null;index" json:"round_number"`       // 回合数
	Status       string        `gorm:"not null;default:'waiting'" json:"status"` // 状态: waiting, matched, confirmed, cancelled
	RoundPlayers []RoundPlayer `gorm:"not null;default:''" json:"round_players"` // 回合玩家记录
}

// RoundPlayer 回合玩家记录
type RoundPlayer struct {
	BaseModel
	RoundID       uint        `gorm:"not null;index" json:"round_id"`         // 回合唯一ID
	MatchPlayerID uint        `gorm:"not null;index" json:"player_id"`        // 匹配玩家唯一ID
	RoundCards    []RoundCard `gorm:"not null;default:''" json:"round_cards"` // 回合牌面记录
}

// RoundCard 回合牌面记录
type RoundCard struct {
	BaseModel
	RoundID        uint   `gorm:"not null;index" json:"round_id"`             // 回合唯一ID
	CardCommtiment string `gorm:"not null;default:''" json:"card_commtiment"` // 牌面哈希值
	Card           Card   `gorm:"not null;index" json:"card"`                 // 使用过的卡牌
	Items          []Item `gorm:"not null;default:''" json:"items"`           // 使用过的道具
}
