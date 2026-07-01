package dao

const MaxChainTokenFailReasonLen = 256

type ChainTokenLedgerStatus string

const (
	ChainTokenLedgerStatusPending   ChainTokenLedgerStatus = "pending"
	ChainTokenLedgerStatusAuditing  ChainTokenLedgerStatus = "auditing"
	ChainTokenLedgerStatusFinalized ChainTokenLedgerStatus = "finalized"
	ChainTokenLedgerStatusFailed    ChainTokenLedgerStatus = "failed"
)

type ChainTokenLedgerEventType string

const (
	ChainTokenLedgerEventDeposit  ChainTokenLedgerEventType = "deposit"
	ChainTokenLedgerEventWithdraw ChainTokenLedgerEventType = "withdraw"
)

type ChainTokenLedger struct {
	BaseModel

	RequestID        *string                   `gorm:"size:36;uniqueIndex" json:"request_id"`
	ChainID          int64                     `gorm:"not null;uniqueIndex:uq_chain_token_ledger,priority:1;index" json:"chain_id"`
	TxHash           string                    `gorm:"not null;size:66;uniqueIndex:uq_chain_token_ledger,priority:2;index" json:"tx_hash"`
	LogIndex         uint32                    `gorm:"not null;uniqueIndex:uq_chain_token_ledger,priority:3" json:"log_index"`
	BlockNumber      uint64                    `gorm:"not null;index" json:"block_number"`
	BlockHash        string                    `gorm:"not null;size:66" json:"block_hash"`
	EventType        ChainTokenLedgerEventType `gorm:"type:varchar(16);not null;index;index:idx_ctl_player_evt_status,priority:2;index:idx_ctl_evt_status_id,priority:1" json:"event_type"`
	PlayerID         int64                     `gorm:"not null;index;index:idx_ctl_player_evt_status,priority:1" json:"player_id"`
	CollectorAddress string                    `gorm:"not null;size:42;index" json:"collector_address"`
	AmountWei        string                    `gorm:"not null;size:78" json:"amount_wei"`
	TokenDelta       int32                     `gorm:"not null" json:"token_delta"`
	Status           ChainTokenLedgerStatus    `gorm:"type:varchar(16);not null;index;index:idx_ctl_player_evt_status,priority:3;index:idx_ctl_evt_status_id,priority:2" json:"status"`
	FailReason       string                    `gorm:"type:varchar(256)" json:"fail_reason"`
	Signature        string                    `gorm:"size:132" json:"signature"`
	FromAddress      string                    `gorm:"size:42" json:"from_address"`
	ToAddress        string                    `gorm:"size:42" json:"to_address"`
	Operator         string                    `gorm:"size:42" json:"operator"`
	NewCreditedWei   string                    `gorm:"size:78" json:"new_credited_wei"`
}

func (ChainTokenLedger) TableName() string { return "chain_token_ledgers" }
