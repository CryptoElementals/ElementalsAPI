package types

import dao "github.com/CryptoElementals/common/models"

type GameCreatedEvent struct {
	GameID              uint
	Players             []PlayerAddress
	IsContinueGame      bool
	ConfirmationTimeout int64 // Timeout for game match confirmation
}

type GameReadyEvent struct {
	GameID            uint
	MaxRoundNum       uint32
	MaxTurnNum        uint32
	InitialHP         uint32
	InitialMultiplier uint32
	Players           []PlayerAddress
}

type TurnReadyEvent struct {
	GameID                      uint
	RoundNumber                 uint32
	TurnNumber                  uint32
	CommitmentSubmissionTimeout int64
}

type PlayerTurnInfo struct {
	PlayerAddress PlayerAddress
	SubmittedCard *dao.TurnSubmittedCard
}

type TurnCompletedEvent struct {
	GameID              uint
	RoundNumber         uint32
	TurnNumber          uint32
	IsRoundComplete     bool
	IsGameComplete      bool
	PlayerTurnInfo      []*PlayerTurnInfo
	GameResult          *dao.GameResult // Only set when IsGameComplete is true
	ConfirmationTimeout *int64          // Optional: timeout for round confirmation after turn completes
	GameContinueTimeout *int64          // Optional: timeout for game continue (only when game is complete)
}

type RoundPartialReadyEvent struct {
	GameID       uint
	RoundNumber  uint32
	ReadyAddress PlayerAddress
}

type RoundReadyEvent struct {
	GameID         uint
	RoundNumber    uint32
	RoundStartedAt int64
}

type CommitmentsOnChainEvent struct {
	GameID                uint
	RoundNumber           uint32
	TurnNumber            uint32
	CardSubmissionTimeout int64
}

type RoundCompletedEvent struct {
	GameID      uint
	RoundNumber uint32
}

type GameCompletedEvent struct {
	GameID   uint
	GameInfo *dao.Game
}

type ContinueCanceledEvent struct {
	GameID uint
}

type SyncEvent GameCompletedEvent
