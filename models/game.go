package dao

import "github.com/CryptoElementals/common/rpc/proto"

type GameArgs struct {
	MaxRounds int64
	InitialHP int64 `json:"initial_hp"`

	// timeouts
	GameMatchTimeout    int64
	RoundConfirmTimeout int64
	RoundTimeout        int64
	ContinueTimeout     int64

	// timeout redundancy
	GameMatchTimeoutRedundancy    int64
	RoundConfirmTimeoutRedundancy int64
	RoundTimeoutRedundancy        int64
	ContinueTimeoutRedundancy     int64
}

type Game struct {
	BaseModel
	RoomContract string           `gorm:"index" json:"room_contract"` // 房间合约地址
	Type         uint             `gorm:"not null" json:"type"`       // 游戏模式
	Status       proto.GameStatus `gorm:"not null" json:"status"`

	Players    []*GamePlayerInfo `json:"players"`
	Rounds     []*Round          `json:"rounds"`
	GameResult *GameResult       `json:"game_result"`

	GameArgs
}

// Round 回合记录
type Round struct {
	BaseModel
	GameID           uint                      `json:"game_id"`            // 匹配唯一ID
	RoundNumber      uint32                    `json:"round_number"`       // 回合数
	Status           proto.RoundStatus         `json:"status"`             // 状态: waiting, matched, confirmed, cancelled
	PlayerRoundInfos []*PlayerRoundInfo        `json:"player_round_infos"` // 回合玩家记录
	SetupOnChainAt   int64                     `json:"setup_on_chain_at"`
	IsLastRound      bool                      `json:"is_last_round"`
	CompleteReason   proto.RoundCompleteReason `json:"complete_reason"`
	RoundEndTime     int64                     `json:"round_end_at"`
}

// PlayerRoundInfo 回合玩家记录
type PlayerRoundInfo struct {
	BaseModel
	RoundID             uint                  `json:"round_id"`
	WalletAddress       string                `gorm:"not null;index:idx_wallet_address,length:42;size:42" json:"wallet_address"`
	TemporaryAddress    string                `json:"temporary_address"`
	PlayerReady         bool                  `json:"player_ready"`
	Salt                []byte                `json:"salt"`
	LostHP              int32                 `json:"lost_hp"`
	SubmittedCommitment []byte                `json:"submitted_commitment"` // 牌面哈希值
	SubmittedCards      []*RoundSubmittedCard `json:"submitted_cards"`      // 回合牌面记录
	Surrendered         bool                  `json:"surrendered"`
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
	BattleRewardID         uint
	WalletAddress          string `gorm:"not null;index:wallet_address" json:"wallet_address"`
	TemporaryAddress       string
	TokenChange            int32
	PointChange            int32
	PlayerGameResultStatus proto.PlayerGameResultStatus
	IsOffline              bool
	Surrendered            bool
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
	BattleReward           *BattleReward
}
