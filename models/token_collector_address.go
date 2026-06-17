package dao

import "gorm.io/gorm"

const (
	TokenCollectorSourceOnChainRefresh = "on_chain_refresh"
	TokenCollectorSourceWalletAdded    = "wallet_added"
	TokenCollectorSourceAddressUpdated = "address_updated"
	TokenCollectorSourceHistory        = "history"
)

type TokenCollectorAddress struct {
	gorm.Model
	ChainID      uint64  `gorm:"type:bigint unsigned;uniqueIndex:idx_chain_collector_addr" json:"chain_id"`
	Address      string  `gorm:"type:varchar(42);uniqueIndex:idx_chain_collector_addr" json:"address"`
	WalletIndex  *uint64 `gorm:"type:bigint unsigned" json:"wallet_index,omitempty"`
	Source       string  `gorm:"type:varchar(32)" json:"source"`
	BlockNumber  *uint64 `gorm:"type:bigint unsigned" json:"block_number,omitempty"`
}

func (TokenCollectorAddress) TableName() string {
	return "token_collector_addresses"
}
