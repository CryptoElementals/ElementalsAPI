package dao

import "time"

// LockToken 锁定代币表 - 记录用户加入匹配队列时锁定的代币
type LockToken struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Address     string     `gorm:"type:varchar(42);not null;index" json:"address"` // 用户地址
	TempAddress string     `gorm:"type:varchar(42);not null" json:"temp_address"`  // 临时地址
	Token       int        `gorm:"not null;default:10000" json:"token"`            // 锁定的代币数量
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`               // 创建时间
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`               // 更新时间
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at"`                        // 删除时间（软删除）
}

// TableName 指定表名
func (LockToken) TableName() string {
	return "lock_tokens"
}
