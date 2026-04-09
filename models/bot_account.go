package dao

import "time"

// BotAccount stores bot identity and temporary wallet material for bot-server provisioning.
type BotAccount struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PlayerID       int64     `gorm:"column:player_id;type:bigint;not null;uniqueIndex" json:"player_id"`
	TempPrivateKey string    `gorm:"column:temp_private_key;type:varchar(100);not null" json:"temp_private_key"`
	TempAddress    string    `gorm:"column:temp_address;type:varchar(100);not null;uniqueIndex" json:"temp_address"`
	Metadata       string    `gorm:"column:metadata;type:text" json:"metadata"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (BotAccount) TableName() string {
	return "bot_accounts"
}
