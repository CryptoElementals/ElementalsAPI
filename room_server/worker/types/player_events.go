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

type RoundPartialReadyEvent struct {
	GameID       uint
	RoundNumber  uint32
	ReadyAddress PlayerAddress
}

type RoundReadyEvent struct {
	GameID         uint
	RoundNumber    uint32
	RoundStartedAt int64
	RoundTimeout   int64
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

type GamePurgeEvent struct {
	GameID uint
}

type ContinueCanceledEvent struct {
	GameID uint
}

type SyncEvent GameCompletedEvent
