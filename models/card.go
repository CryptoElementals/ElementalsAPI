package dao

import (
	"time"
)

// Card 卡牌表 - 存储所有卡牌的基础信息
type Card struct {
	CardID      int       `gorm:"primaryKey" json:"card_id"`            // 卡牌ID (1, 2, 3, 4, 5...)
	ElementType string    `gorm:"not null;size:10" json:"element_type"` // 五行属性 (Metal, Wood, Water, Fire, Earth)
	Level       string    `gorm:"not null;size:20" json:"level"`        // 卡牌等级 (legendary, epic, rare, normal)
	LifeForce   int       `gorm:"not null;default:0" json:"life_force"` // 生命力
	Attack      int       `gorm:"not null;default:0" json:"attack"`     // 攻击力
	Defense     int       `gorm:"not null;default:0" json:"defense"`    // 防御力
	ImageURL    string    `gorm:"size:500" json:"image_url"`            // 卡面图片URL
	Description string    `gorm:"size:1000" json:"description"`         // 卡牌描述信息
	Name        string    `gorm:"not null;size:100" json:"name"`        // 卡牌名称
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (Card) TableName() string {
	return "cards"
}
