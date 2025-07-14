package dao

type GameInfo struct {
	BaseModel
	RoomContract string       `gorm:"index" json:"room_contract"` // 房间合约地址
	Type         string       `gorm:"not null" json:"type"`       // 游戏模式
	Status       uint         `gorm:"not null" json:"status"`
	Players      []GamePlayer `gorm:"not null;default:''" json:"players"`
	Rounds       []Round      `json:"rounds"`
}

// GamePlayer 匹配记录表
type GamePlayer struct {
	BaseModel
	MatchID       uint   `gorm:"not null;" json:"match_id"` // 匹配唯一ID（两个用户共享）
	WalletAddress string `gorm:"not null;index:address" json:"wallet_address"`
	TempAddress   string `gorm:"not null;index:address" json:"temp_address"`
}

// Round 回合记录
type Round struct {
	BaseModel
	MatchID          uint              `gorm:"not null;index" json:"match_id"`                // 匹配唯一ID
	RoundNumber      int               `gorm:"not null;index" json:"round_number"`            // 回合数
	Status           string            `gorm:"not null;default:'waiting'" json:"status"`      // 状态: waiting, matched, confirmed, cancelled
	PlayerRoundInfos []PlayerRoundInfo `gorm:"not null;default:''" json:"player_round_infos"` // 回合玩家记录
}

// PlayerRoundInfo 回合玩家记录
type PlayerRoundInfo struct {
	BaseModel
	RoundID             uint                 `gorm:"not null;index" json:"round_id"`
	GamePlayerID        uint                 `gorm:"not null;" json:"game_player_id"`
	GamePlayer          GamePlayer           `gorm:"not null;index" json:"game_player"`                // 匹配玩家唯一ID
	RoundSubmittedCards []RoundSubmittedCard `gorm:"not null;default:''" json:"round_submitted_cards"` // 回合牌面记录
	SubmittedCommitment []byte               `gorm:"not null;default:''" json:"submitted_commitment"`  // 牌面哈希值
	Salt                []byte               `gorm:"not null;default:''" json:"salt"`
}

// RoundSubmittedCard 回合牌面记录
type RoundSubmittedCard struct {
	BaseModel
	RoundID      uint   `gorm:"not null;index" json:"round_id"` // 回合唯一ID
	HealthBefore uint32 `gorm:"not null;default:0" json:"health_before"`
	HealthAfter  uint32 `gorm:"not null;default:0" json:"health_after"`
	Multiplier   uint32 `gorm:"not null;default:0" json:"multiplier"`
	CardID       uint32 `gorm:"not null;index" json:"card"` // 使用过的卡牌
}
