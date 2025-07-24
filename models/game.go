package dao

import "github.com/CryptoElementals/common/rpc/proto"

type Game struct {
	BaseModel
	RoomContract string            `gorm:"index" json:"room_contract"` // 房间合约地址
	Type         uint              `gorm:"not null" json:"type"`       // 游戏模式
	Status       proto.GameStatus  `gorm:"not null" json:"status"`
	InitialHP    int64             `json:"initial_hp"`
	Players      []*GamePlayerInfo `json:"players"`
	Rounds       []*Round          `json:"rounds"`
	GameResult   *GameResult       `json:"game_result"`
	MaxRounds    int64
	RoundTimeout int64
}

// Round 回合记录
type Round struct {
	BaseModel
	GameID           uint               `json:"game_id"`            // 匹配唯一ID
	RoundNumber      uint32             `json:"round_number"`       // 回合数
	Status           proto.RoundStatus  `json:"status"`             // 状态: waiting, matched, confirmed, cancelled
	PlayerRoundInfos []*PlayerRoundInfo `json:"player_round_infos"` // 回合玩家记录
	IsLastRound      bool               `json:"is_last_round"`
}

// PlayerRoundInfo 回合玩家记录
type PlayerRoundInfo struct {
	BaseModel
	RoundID             uint                  `json:"round_id"`
	WalletAddress       string                `json:"wallet_address"`
	TemporaryAddress    string                `json:"temporary_address"`
	PlayerReady         bool                  `json:"player_ready"`
	Salt                []byte                `json:"salt"`
	LostHP              int32                 `json:"lost_hp"`
	SubmittedCommitment []byte                `json:"submitted_commitment"` // 牌面哈希值
	SubmittedCards      []*RoundSubmittedCard `json:"submitted_cards"`      // 回合牌面记录
}

// RoundSubmittedCard 回合牌面记录
type RoundSubmittedCard struct {
	BaseModel
	PlayerRoundInfoID uint                  `json:"player_round_info_id"` // 回合唯一ID
	CardID            uint                  `json:"card"`                 // 使用过的卡牌
	CardNumber        uint32                `json:"card_number"`
	HealthBefore      uint32                `json:"health_before"`
	HealthAfter       uint32                `json:"health_after"`
	MultiplierBefore  uint32                `json:"multiplier_before"`
	MultiplierAfter   uint32                `json:"multiplier_after"`
	Description       string                `json:"description"`
	ElementRelation   proto.ElementRelation `json:"element_relation"`
	CardEffects       []*CardEffect         `json:"card_effects"`
}

type CardEffect struct {
	BaseModel
	RoundSubmittedCardID   uint
	Type                   proto.BattleEffectType
	Value                  int32
	Description            string
	TargetWalletAddress    string
	TargetTemporaryAddress string
}

type GamePlayerInfo struct {
	BaseModel
	GameID           uint   `json:"game_id"`
	WalletAddress    string `gorm:"not null;index:address" json:"wallet_address"`
	TemporaryAddress string `gorm:"not null;index:address" json:"temporary_address"`
}

type PlayerReward struct {
	BaseModel
	BattleRewardID   uint
	WalletAddress    string
	TemporaryAddress string
	TokenChange      int32
	PointChange      int32
}

type BattleReward struct {
	BaseModel
	GameResultID  uint
	SystemFee     int32
	PlayerRewards []*PlayerReward
}

type GameResult struct {
	BaseModel
	GameID                 uint
	Multiplier             int32
	WinnerWalletAddress    string
	WinnerTemporaryAddress string
	GameResultType         proto.GameResultType
	BattleReword           *BattleReward
}
