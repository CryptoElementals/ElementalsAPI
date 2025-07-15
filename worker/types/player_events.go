package types

type GameCreatedEvent struct {
	GameID      uint
	GamePlayers []PlayerAddress
}

type RoundReadyEvent struct {
	GameID      uint
	RoundNumber int
}

type CommitmentsObservedEvent struct{}

type CardsSubmittedEvent struct{}

type RoundCompleteEvent struct{}

type GameCompleteEvent struct{}

type SyncEvent GameCompleteEvent
