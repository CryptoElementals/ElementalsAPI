package types

type GameMatchedEvent struct {
	Players []PlayerAddress
}

type GameContinueEvent struct {
	Players []PlayerAddress
}

type PlayerReadyEvent struct {
	GameId        uint
	RoundNumber   uint32
	PlayerAddress PlayerAddress
}

type PlayerContinueEvent struct {
	GameId        uint
	PlayerAddress PlayerAddress
}

type NewRoundSetupComplete struct {
	GameID      uint
	RoundNumber uint32
	TimeStamp   int64
}

type NewTurnSetupComplete struct {
	GameID      uint
	RoundNumber uint32
	TurnNumber  uint32
	TimeStamp   int64
}

type RoomContractCreated struct {
	GameID              uint
	RoomContractAddress string
	TimeStamp           int64
}

type PlayerCommitmentOnChain struct {
	GameID          uint
	Address         PlayerAddress
	RoundNumber     uint32
	Commitment      []byte
	CommitmentIndex uint32 // Index of the commitment being submitted (1, 2, or 3)
	TimeStamp       int64
}

type PlayerCardOnChain struct {
	GameID      uint
	Address     PlayerAddress
	RoundNumber uint32
	Salt        []byte
	Card        uint
	CardIndex   uint32 // Index of the card being submitted (1, 2, or 3)
	TimeStamp   int64
}

type GameTimeout struct {
	GameID      uint
	RoundNumber int
	Reason      string
}

type SurrenderEvent struct {
	GameID  uint
	Address PlayerAddress
}

type AbortGame struct {
}
