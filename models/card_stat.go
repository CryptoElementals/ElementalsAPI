package dao

import (
	"gorm.io/gorm"
)

// CardStat 卡牌统计表 - 存储用户每张卡牌的使用统计
type CardStat struct {
	gorm.Model
	Address    string `gorm:"type:varchar(42);not null;uniqueIndex:idx_addr_card" json:"address"` // 关联用户地址
	CardID     uint   `gorm:"not null;default:0" json:"card_id"`                                  // 卡牌ID
	RoundCount uint   `gorm:"not null;default:0" json:"round_count"`                              // 该玩家的玩的游戏轮数
	UsageCount uint   `gorm:"not null;default:0" json:"usage_count"`                              // 使用该卡牌的次数
	WinCount   uint   `gorm:"not null;default:0" json:"win_count"`                                // 使用该卡牌赢的次数
	LastGameID uint   `gorm:"not null;default:0" json:"last_game_id"`
}

// TableName 指定表名
func (CardStat) TableName() string {
	return "card_stats"
}
