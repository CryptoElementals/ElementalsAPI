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

type RoomContractCreated struct {
	GameID              uint
	RoomContractAddress string
	TimeStamp           int64
}

type PlayerCommitmentOnChain struct {
	GameID      uint
	Address     PlayerAddress
	RoundNumber uint32
	Commitment  []byte
	TimeStamp   int64
}

type PlayerCardsOnChain struct {
	GameID      uint
	Address     PlayerAddress
	RoundNumber uint32
	Salt        []byte
	Cards       []uint
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
