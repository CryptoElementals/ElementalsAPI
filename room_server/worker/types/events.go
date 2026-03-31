package types

import dao "github.com/CryptoElementals/common/models"

type RequireGameCreationEvent struct {
	GameID         uint
	InitialHP      int64
	RoundTimeout   int64
	MaxRoundNumber int64
	Players        []PlayerAddress
}

type RequireSetupNewTurnEvent struct {
	GameID      uint
	RoundNumber uint32
	TurnNumber  uint32
}

type RoomContractTask struct {
	Index uint8
	Task  []byte
}

type GameCreatedEvent struct {
	GameID              uint
	Players             []PlayerAddress
	IsContinueGame      bool
	ConfirmationTimeout int64 // Timeout for game match confirmation
}

type GameCompletedEvent struct {
	GameID   uint
	GameInfo *dao.Game
}

type GameMatchedEvent struct {
	Players             []PlayerAddress
	ConfirmationTimeout int64 // Timeout for game match confirmation
	// GameType is types.GameTypePVP (default) or types.GameTypeTournament; 0 means PVP.
	GameType uint
}

type GameContinueEvent struct {
	Players []PlayerAddress
}
