package dao

type WithdrawLedger struct {
	BaseModel

	PlayerID         int64  `gorm:"not null;index" json:"player_id"`
	Amount           string `gorm:"not null;size:78" json:"amount"`
	Signature        string `gorm:"not null;size:132" json:"signature"`
	CollectorAddress string `gorm:"not null;size:42;index" json:"collector_address"`
	ChainID          int64  `gorm:"not null;index" json:"chain_id"`
	TxHash           string `gorm:"size:66;index" json:"tx_hash"`
}

func (WithdrawLedger) TableName() string { return "withdraw_ledgers" }
