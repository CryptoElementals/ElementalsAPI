package types

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
	Commitment  []uint32
}
