package dao

import (
	"gorm.io/gorm"
)

type BlockSync struct {
	gorm.Model
	Type        string `gorm:"type:varchar(50);index" json:"type"` // e.g. "head"
	BlockHeight uint64 `gorm:"type:bigint unsigned" json:"block_height"`
}

func (BlockSync) TableName() string {
	return "block_sync"
}
