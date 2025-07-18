package dao

import "time"

// CardStat 卡牌统计表 - 存储用户每张卡牌的使用统计
type CardStat struct {
	ID          uint      `gorm:"primaryKey"`
	Address     string    `gorm:"type:varchar(42);not null;index:idx_addr_card,unique"` // 关联用户地址
	CardName    string    `gorm:"type:varchar(50);not null;index:idx_addr_card,unique"` // 卡牌名称
	Frequency   float64   `gorm:"default:0"`                                            // 使用次数
	WinningRate float64   `gorm:"default:0.0"`                                          // 胜率
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (CardStat) TableName() string {
	return "card_stats"
}
