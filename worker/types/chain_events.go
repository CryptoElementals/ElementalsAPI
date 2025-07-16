package types

type RequireContractCreationEvent struct {
	Players []PlayerAddress
}

type SetupNewRoundEvent struct {
	ContractAddress string
	RoundNumber     uint32
}
