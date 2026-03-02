package dao

import (
	"gorm.io/gorm"
)

// CardStat 卡牌统计表 - 存储用户每张卡牌的使用统计
type CardStat struct {
	gorm.Model
	PlayerID              int64 `gorm:"type:bigint;not null;uniqueIndex:idx_player_card" json:"player_id"` // 关联用户ID
	CardID                uint  `gorm:"not null;default:0;uniqueIndex:idx_player_card" json:"card_id"`     // 卡牌ID
	RoundCount            uint  `gorm:"not null;default:0" json:"round_count"`                             // 该玩家的玩的游戏轮数
	UsageCount            uint  `gorm:"not null;default:0" json:"usage_count"`                             // 使用该卡牌的次数
	WinCount              uint  `gorm:"not null;default:0" json:"win_count"`
	LoseCount             uint  `gorm:"not null;default:0" json:"lose_count"`
	TieCount              uint  `gorm:"not null;default:0" json:"tie_count"`
	LastPlayerRoundInfoID uint  `gorm:"not null;default:0;index" json:"last_player_round_info_id"`
}

// TableName 指定表名
func (CardStat) TableName() string {
	return "card_stats"
}
