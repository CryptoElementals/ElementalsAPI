package types

import "github.com/CryptoElementals/common/rpc/proto"

type GameMatchedEvent struct {
	Players []PlayerAddress
}

type GameContinueEvent struct {
	Players []PlayerAddress
}

type PlayerReadyEvent struct {
	GameId        uint
	RoundNumber   uint32
	TurnNumber    uint32
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
	GameID    uint
	TimeStamp int64
	// RoomContractAddress removed - always uses RoomV2 contract address
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

type SubmitPlayerCommitment struct {
	GameID          uint
	Address         PlayerAddress
	RoundNumber     uint32
	Commitment      []byte
	CommitmentIndex uint32 // Index of the commitment being submitted (1, 2, or 3)
	Signature       []byte
}

type SubmitPlayerCard struct {
	GameID      uint
	Address     PlayerAddress
	RoundNumber uint32
	Salt        []byte
	Card        uint
	CardIndex   uint32 // Index of the card being submitted (1, 2, or 3)
	Signature   []byte
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

// GetGameInfoRequest is a request event that needs acknowledgment to get game info
type GetGameInfoRequest struct {
	RequestID string // Optional request ID for tracking
}

// GetBattleInfoRequest is a request event to get battle info for a specific round
type GetBattleInfoRequest struct {
	RoundNumber uint32 // Round number to get battle info for
}

// GetBattleInfoResponse contains the battle info response
type GetBattleInfoResponse struct {
	RoundResult *proto.RoundResult
	GameResult  *proto.GameResult
}

// GetGamePhaseRequest is a request event to get game phase
type GetGamePhaseRequest struct {
}

type SyncGamePhaseRequest struct {
	Receiver *PlayerAddress
}

// GetGameResultRequest is a request event to get game result
type GetGameResultRequest struct {
}

// GamePhaseSyncEvent is an event that sends game phase directly to player worker
type GamePhaseSyncEvent struct {
	GamePhase *proto.GamePhase
}
