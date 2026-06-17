package dao

import (
	"gorm.io/gorm"
)

type BlockSync struct {
	gorm.Model
	ChainID     uint64 `gorm:"type:bigint unsigned;uniqueIndex:idx_chain_type" json:"chain_id"`
	Type        string `gorm:"type:varchar(50);uniqueIndex:idx_chain_type" json:"type"` // e.g. "head", "finalized"
	BlockHeight uint64 `gorm:"type:bigint unsigned" json:"block_height"`
}

func (BlockSync) TableName() string {
	return "block_sync"
}
