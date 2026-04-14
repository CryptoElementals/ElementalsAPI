package types

type RequireGameCreationEvent struct {
	GameID         int64
	InitialHP      int64
	RoundTimeout   int64
	MaxRoundNumber int64
	Players        []PlayerAddress
}

type RequireSetupNewTurnEvent struct {
	GameID      int64
	RoundNumber uint32
	TurnNumber  uint32
}

type RoomContractTask struct {
	Index uint8
	Task  []byte
}

// GameCreatedEvent is used by in-process worker tests / harnesses (not for PubSub TYPE_MATCHED).
type GameCreatedEvent struct {
	GameID              int64
	Players             []PlayerAddress
	IsContinueGame      bool
	ConfirmationTimeout int64
	MatchID             int64
}

// GameCompletedEvent is published when a match ends. GameType is GameTypePVP or GameTypeTournament (0 treated as PVP).
type GameCompletedEvent struct {
	GameID   int64
	GameType uint
}

type GameMatchedEvent struct {
	Players             []PlayerAddress
	ConfirmationTimeout int64 // Timeout for game match confirmation
	// GameType is types.GameTypePVP (default) or types.GameTypeTournament; 0 means PVP.
	GameType uint
}

