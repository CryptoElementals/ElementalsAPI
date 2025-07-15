package dao

import (
	"time"
)

// Card 卡牌表 - 存储所有卡牌的基础信息
type Card struct {
	CardID             int       `gorm:"primaryKey" json:"CardID"`            // 卡牌ID (1, 2, 3, 4, 5...)
	ElementType        string    `gorm:"not null;size:10" json:"ElementType"` // 五行属性 (Metal, Wood, Water, Fire, Earth)
	Level              string    `gorm:"not null;size:20" json:"Level"`       // 卡牌等级 (legendary, epic, rare, normal)
	LifeForce          int       `gorm:"not null;default:0" json:"LifeForce"` // 生命力
	Attack             int       `gorm:"not null;default:0" json:"Attack"`    // 攻击力
	Defense            int       `gorm:"not null;default:0" json:"Defense"`   // 防御力
	NormalImageURL     string    `gorm:"size:500" json:"NormalImageURL"`      // 普通态图片URL
	ActiveImageURL     string    `gorm:"size:500" json:"ActiveImageURL"`      // 激活态图片URL
	BackgroundImageURL string    `gorm:"size:500" json:"BackgroundImageURL"`  // 背景图URL
	IconURL            string    `gorm:"size:500" json:"IconURL"`             // icon图片URL
	Description        string    `gorm:"size:1000" json:"Description"`        // 卡牌描述信息
	Name               string    `gorm:"not null;size:100" json:"Name"`       // 卡牌名称
	CreatedAt          time.Time `gorm:"autoCreateTime" json:"CreatedAt"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime" json:"UpdatedAt"`
}

// TableName 指定表名
func (Card) TableName() string {
	return "cards"
}
