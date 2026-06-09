package dao

type TokenCollectLedger struct {
	BaseModel

	PlayerID         int64  `gorm:"not null;index" json:"player_id"`
	PlayerAddress    string `gorm:"not null;size:42;index" json:"player_address"`
	WalletIndex      uint64 `gorm:"not null" json:"wallet_index"`
	CollectorAddress string `gorm:"not null;size:42" json:"collector_address"`
	TokenAmount      string `gorm:"not null;size:78" json:"token_amount"`
	TxHash           string `gorm:"size:66;index" json:"tx_hash"`
	ChainID          int64  `gorm:"not null;index" json:"chain_id"`
}

func (TokenCollectLedger) TableName() string { return "token_collect_ledgers" }
