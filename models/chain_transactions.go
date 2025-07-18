package dao

import "time"

type TxStatus uint64

const (
	TxStatusInit TxStatus = iota
	TxStatusSent
	TxStatusSuccess
	TxStatusFailed
)

type CreateRoomTx struct {
	BaseModel
	GameID         uint
	ContractHash   string
	BlockHash      string
	TransacionHash string
	Status         TxStatus
	RoundTimeout   time.Duration
	MaxRounds      uint64
}

type SetRoundReadyTx struct {
	BaseModel
	GameID         uint
	ContractHash   string
	TransacionHash string
	BlockHash      string
	Status         TxStatus
	RoundNumber    uint64
}

type CommitmentOnChainTx struct {
	BaseModel
	GameID           uint
	ContractHash     string
	TransacionHash   string
	BlockHash        string
	Status           TxStatus
	RoundNumber      uint64
	WalletAddress    string
	TemporaryAddress string
}

type CardsOnChainTx struct {
	BaseModel
	GameID           uint
	ContractHash     string
	TransacionHash   string
	BlockHash        string
	Status           TxStatus
	RoundNumber      uint64
	WalletAddress    string
	TemporaryAddress string
}
