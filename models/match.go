package dao

type Match struct {
	BaseModel
	RoomContract string        `gorm:"index" json:"room_contract"` // 房间合约地址
	Mode         string        `gorm:"not null" json:"mode"`       // 游戏模式
	Status       uint          `gorm:"not null" json:"status"`
	Players      []MatchPlayer `gorm:"not null;default:''" json:"players"`
	Rounds       []Round       `json:"rounds"`
}

func (Match) TableName() string {
	return "matches"
}

// MatchPlayer 匹配记录表
type MatchPlayer struct {
	BaseModel
	MatchID       uint   `gorm:"not null;" json:"match_id"` // 匹配唯一ID（两个用户共享）
	WalletAddress string `gorm:"not null;index:address" json:"wallet_address"`
	TempAddress   string `gorm:"not null;index:address" json:"temp_address"`
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
	HealthBefore   uint32 `gorm:"not null;default:0" json:"health_before"`
	HealthAfter    uint32 `gorm:"not null;default:0" json:"health_after"`
	Multiplier     uint32 `gorm:"not null;default:0" json:"multiplier"`
}
