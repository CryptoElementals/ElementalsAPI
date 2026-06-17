package dao

type ChainTokenLedgerStatus string

const (
	ChainTokenLedgerStatusApplied  ChainTokenLedgerStatus = "applied"
	ChainTokenLedgerStatusRejected ChainTokenLedgerStatus = "rejected"
)

type ChainTokenLedgerEventType string

const (
	ChainTokenLedgerEventDeposit  ChainTokenLedgerEventType = "deposit"
	ChainTokenLedgerEventWithdraw ChainTokenLedgerEventType = "withdraw"
)

type ChainTokenLedger struct {
	BaseModel

	ChainID          int64                     `gorm:"not null;uniqueIndex:uq_chain_token_ledger,priority:1;index" json:"chain_id"`
	TxHash           string                    `gorm:"not null;size:66;uniqueIndex:uq_chain_token_ledger,priority:2;index" json:"tx_hash"`
	LogIndex         uint32                    `gorm:"not null;uniqueIndex:uq_chain_token_ledger,priority:3" json:"log_index"`
	BlockNumber      uint64                    `gorm:"not null;index" json:"block_number"`
	BlockHash        string                    `gorm:"not null;size:66" json:"block_hash"`
	EventType        ChainTokenLedgerEventType `gorm:"type:varchar(16);not null;index" json:"event_type"`
	PlayerID         int64                     `gorm:"not null;index" json:"player_id"`
	CollectorAddress string                    `gorm:"not null;size:42;index" json:"collector_address"`
	AmountWei        string                    `gorm:"not null;size:78" json:"amount_wei"`
	TokenDelta       int32                     `gorm:"not null" json:"token_delta"`
	Status           ChainTokenLedgerStatus    `gorm:"type:varchar(16);not null;index" json:"status"`
	RejectReason     string                    `gorm:"type:varchar(64)" json:"reject_reason"`
	FromAddress      string                    `gorm:"size:42" json:"from_address"`
	ToAddress        string                    `gorm:"size:42" json:"to_address"`
	Operator         string                    `gorm:"size:42" json:"operator"`
	NewCreditedWei   string                    `gorm:"size:78" json:"new_credited_wei"`
}

func (ChainTokenLedger) TableName() string { return "chain_token_ledgers" }
