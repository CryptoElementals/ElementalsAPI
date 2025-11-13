package types

import dao "github.com/CryptoElementals/common/models"

type GameCreatedEvent struct {
	GameID         uint
	Players        []PlayerAddress
	IsContinueGame bool
}

type GameReadyEvent struct {
	GameID          uint
	ContractAddress string
}

type TurnReadyEvent struct {
	GameID      uint
	RoundNumber uint32
	TurnNumber  uint32
}

type PlayerTurnInfo struct {
	PlayerAddress PlayerAddress
	SubmittedCard *dao.RoundSubmittedCard
}

type TurnCompletedEvent struct {
	GameID         uint
	RoundNumber    uint32
	TurnNumber     uint32
	PlayerTurnInfo []*PlayerTurnInfo
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

type CardsOnChainEvent struct {
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

type ContinueCanceledEvent struct {
	GameID uint
}

type SyncEvent GameCompletedEvent
