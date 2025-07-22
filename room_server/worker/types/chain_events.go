package types

type RequireContractCreationEvent struct {
	GameID  uint
	Players []PlayerAddress
}

type RequireSetupNewRoundEvent struct {
	GameID          uint
	RoundNumber     uint32
	ContractAddress string
}
