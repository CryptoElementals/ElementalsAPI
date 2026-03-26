package game

// round_battle_types holds shared battle / game-end enums used by round_battle_execute and round_battle_resolve.

// gameResultType represents the type of game result (internal to battle resolution).
type gameResultType int32

const (
	gameResultNormal gameResultType = iota
	gameResultKO
	gameResultTie
)

// Battle action description constants (human-readable card effect lines).
const (
	ActionAttackedBy       = "is attacked by"
	ActionDoubleAttackedBy = "is double attacked by"
	ActionHealedBy         = "is healed by"
)

// gameEndState is a player's snapshot for win/loss/tie logic.
type gameEndState struct {
	HP               int
	Multiplier       uint32
	PlayerId         int64
	TemporaryAddress string
	Status           playerStatus
}
