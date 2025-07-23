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
	GameID          uint
	ContractAddress string
	TxHash          string
	BlockHash       string
	BlockNumber     uint64
	Status          TxStatus
	RoundTimeout    time.Duration
	MaxRounds       uint64
}

type SetRoundReadyTx struct {
	BaseModel
	GameID          uint
	ContractAddress string
	TxHash          string
	BlockHash       string
	BlockNumber     uint64
	Status          TxStatus
	RoundNumber     uint64
}

type CommitmentOnChainTx struct {
	BaseModel
	GameID           uint
	ContractAddress  string
	TxHash           string
	BlockHash        string
	BlockNumber      uint64
	Status           TxStatus
	RoundNumber      uint64
	WalletAddress    string
	TemporaryAddress string
}

type CardsOnChainTx struct {
	BaseModel
	GameID           uint
	ContractAddress  string
	TxHash           string
	BlockHash        string
	BlockNumber      uint64
	Status           TxStatus
	RoundNumber      uint64
	WalletAddress    string
	TemporaryAddress string
}
