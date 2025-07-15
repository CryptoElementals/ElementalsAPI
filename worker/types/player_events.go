package types

import dao "github.com/CryptoElementals/common/models"

type GameCreatedEvent struct {
	GameID  uint
	Players []PlayerAddress
}

type GameReadyEvent struct {
	GameID          uint
	ContractAddress string
}

type RoundReadyEvent struct {
	GameID      uint
	RoundNumber int
}

type CommitmentsOnChainEvent struct {
	GameID      uint
	RoundNumber int
}

type CardsOnChainEvent struct {
	GameID      uint
	RoundNumber int
}

type RoundCompletedEvent struct {
	GameID    uint
	RoundInfo *dao.Round
}

type GameCompletedEvent struct {
	GameID   uint
	GameInfo *dao.GameInfo
}

type SyncEvent GameCompletedEvent
