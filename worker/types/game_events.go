package types

type GameMatchedEvent struct {
	Players []PlayerAddress
}

type PlayerReadyEvent struct {
	GameId        uint
	RoundNum      uint32
	PlayerAddress PlayerAddress
}

type RoundSetupComplete struct {
	RoomID      uint
	RoundNumber uint32
}

type RoomContractCreated struct {
	RoomID              uint
	RoomContractAddress string
}

type PlayerCommitmentOnChain struct {
	Address     PlayerAddress
	RoundNumber int
	Commitment  []byte
}

type PlayerCardsOnChain struct {
	Address     PlayerAddress
	RoundNumber int
	Salt        []byte
	Cards       []uint32
}
