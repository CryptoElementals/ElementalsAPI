package scanner

// TokenCollectorEvent is a parsed TokenCollector deposit or withdraw event for logging.
type TokenCollectorEvent struct {
	ChainID          uint64 `json:"chain_id"`
	BlockNumber      uint64 `json:"block_number"`
	BlockHash        string `json:"block_hash"`
	Timestamp        uint64 `json:"timestamp"`
	TxHash           string `json:"tx_hash"`
	LogIndex         uint32 `json:"log_index"`
	CollectorAddress string `json:"collector_address"`
	EventType        string `json:"event_type"`

	Deposit  *DepositPayload  `json:"deposit,omitempty"`
	Withdraw *WithdrawPayload `json:"withdraw,omitempty"`
}

// DepositPayload holds Deposited event fields.
type DepositPayload struct {
	PlayerID    int64  `json:"player_id"`
	FromAddress string `json:"from_address"`
	Amount      string `json:"amount"`
	NewCredited string `json:"new_credited"`
}

// WithdrawPayload holds Withdrawn event fields.
type WithdrawPayload struct {
	PlayerID  int64  `json:"player_id"`
	Operator  string `json:"operator"`
	ToAddress string `json:"to_address"`
	Amount    string `json:"amount"`
}
