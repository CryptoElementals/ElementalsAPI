package types

import "github.com/CryptoElementals/common/rpc/proto"

type RequireGameCreationEvent struct {
	GameID         int64
	InitialHP      int64
	RoundTimeout   int64
	MaxRoundNumber int64
	TournamentID   int64
	TierNo         int64
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

// GameCompletedEvent is published when a match ends.
type GameCompletedEvent struct {
	GameID   int64
	GameType proto.GameType
}

type GameMatchedEvent struct {
	Players             []PlayerAddress
	ConfirmationTimeout int64 // Timeout for game match confirmation
	// GameType defaults to proto.GameType_PVP when unset.
	GameType proto.GameType
}

