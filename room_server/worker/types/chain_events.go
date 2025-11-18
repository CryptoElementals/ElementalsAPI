package types

type RequireContractCreationEvent struct {
	GameID         uint
	InitialHP      int64
	RoundTimeout   int64
	MaxRoundNumber int64
	Players        []PlayerAddress
}

type RequireSetupNewRoundEvent struct {
	GameID          uint
	RoundNumber     uint32
	ContractAddress string
}

type RequireSetupNewTurnEvent struct {
	GameID      uint
	RoundNumber uint32
	TurnNumber  uint32
	// ContractAddress removed - always uses RoomV2 contract address
}
