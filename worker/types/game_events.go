package types

type NewGameEvent struct {
	MsgSender string
	GameId    uint
	Players   []PlayerAddress
}

type PlayerReadyEvent struct {
	GameId        uint
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
