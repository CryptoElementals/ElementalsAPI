package dao

import "github.com/CryptoElementals/common/rpc/proto"

type Game struct {
	BaseModel
	RoomContract string            `gorm:"index" json:"room_contract"` // 房间合约地址
	Type         uint              `gorm:"not null" json:"type"`       // 游戏模式
	Status       proto.GameStatus  `gorm:"not null" json:"status"`
	Players      []*GamePlayerInfo `gorm:"not null;default:''" json:"players"`
	Rounds       []*Round          `json:"rounds"`
}

// Round 回合记录
type Round struct {
	BaseModel
	GameID           uint               `json:"game_id"`            // 匹配唯一ID
	RoundNumber      uint32             `json:"round_number"`       // 回合数
	Status           proto.RoundStatus  `json:"status"`             // 状态: waiting, matched, confirmed, cancelled
	PlayerRoundInfos []*PlayerRoundInfo `json:"player_round_infos"` // 回合玩家记录
}

// PlayerRoundInfo 回合玩家记录
type PlayerRoundInfo struct {
	BaseModel
	RoundID             uint                  `json:"round_id"`
	WalletAddress       string                `json:"wallet_address"`
	TemporaryAddress    string                `json:"temporary_address"`
	PlayerReady         bool                  `json:"player_ready"`
	RoundSubmittedCards []*RoundSubmittedCard `json:"round_submitted_cards"` // 回合牌面记录
	SubmittedCommitment []byte                `json:"submitted_commitment"`  // 牌面哈希值
	Salt                []byte                `json:"salt"`
}

// RoundSubmittedCard 回合牌面记录
type RoundSubmittedCard struct {
	BaseModel
	PlayerRoundInfoID uint   `json:"player_round_info_id"` // 回合唯一ID
	HealthBefore      uint32 `json:"health_before"`
	HealthAfter       uint32 `json:"health_after"`
	Multiplier        uint32 `json:"multiplier"`
	CardID            uint32 `json:"card"` // 使用过的卡牌
}

type GamePlayerInfo struct {
	BaseModel
	GameID           uint   `json:"game_id"`
	WalletAddress    string `gorm:"not null;index:address" json:"wallet_address"`
	TemporaryAddress string `gorm:"not null;index:address" json:"temporary_address"`
	TokenDelta       int32  `json:"token_delta"`
	Points           int32
	Status           proto.GameResultPlayerStatus
}
