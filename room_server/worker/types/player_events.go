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
	RoundNumber uint32
}

type CommitmentsOnChainEvent struct {
	GameID      uint
	RoundNumber uint32
}

type RoundCompletedEvent struct {
	GameID    uint
	RoundInfo *dao.Round
}

type GameCompletedEvent struct {
	GameID   uint
	GameInfo *dao.Game
}

type SyncEvent GameCompletedEvent
