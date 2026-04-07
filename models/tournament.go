// Tournament persistence: schedules define recurrence and bracket size (2^n).
// Each Tournament is one run. Entries enqueue with JoinSequence; at lock, keep the first 2^n by JoinSequence,
// mark the rest KickedOverflow. Matches link bracket pairs to Game rows; on tie, promote the entry with smaller JoinSequence.
package dao

import (
	"time"
)

// TournamentBracketCapacity returns 2^n, or 0 if n is outside [TournamentBracketExpMin, TournamentBracketExpMax].
func TournamentBracketCapacity(exponent uint8) uint32 {
	if exponent < TournamentBracketExpMin || exponent > TournamentBracketExpMax {
		return 0
	}
	return uint32(1) << exponent
}

// --- Tournament schedule (recurrence + bracket size) ---

// TournamentBracketExponent bounds: capacity = 2^n, n in [6, 13] => 64 .. 8192.
const (
	TournamentBracketExpMin = 6
	TournamentBracketExpMax = 13
)

type TournamentSchedule struct {
	BaseModel

	Name string `gorm:"size:128" json:"name"`

	// BracketExponent is n where max players = 2^n (must be 6..13).
	BracketExponent uint8 `gorm:"not null" json:"bracket_exponent"`

	// IntervalSeconds is the period between consecutive tournament starts for this schedule.
	IntervalSeconds int64 `gorm:"not null" json:"interval_seconds"`

	// FirstStartAt is when the first tournament instance of this schedule starts (UTC).
	FirstStartAt time.Time `gorm:"not null" json:"first_start_at"`

	// GameArgsID optionally pins tournament matches to the same rules row as PVP (optional FK).
	GameArgsID uint `gorm:"index" json:"game_args_id"`

	Enabled bool `gorm:"not null;default:true" json:"enabled"`

	GameArgs *GameArgs `json:"game_args,omitempty"`
}

func (TournamentSchedule) TableName() string { return "tournament_schedules" }

// --- One bracket instance (one run of a schedule) ---

type TournamentStatus uint8

const (
	TournamentStatusUnknown TournamentStatus = iota
	TournamentStatusPending          // created, queue not open yet (optional use)
	TournamentStatusOpen             // accepting queue
	TournamentStatusLocked           // sealed: overflow kicked, seeds assigned, bracket ready
	TournamentStatusInProgress       // matches running
	TournamentStatusCompleted
	TournamentStatusCancelled
)

type Tournament struct {
	BaseModel

	TournamentScheduleID uint `gorm:"not null;uniqueIndex:uq_tournament_schedule_instance,priority:1;index" json:"tournament_schedule_id"`

	// InstanceIndex is the 0-based occurrence for this schedule (0 = first at FirstStartAt).
	InstanceIndex uint32 `gorm:"not null;uniqueIndex:uq_tournament_schedule_instance,priority:2" json:"instance_index"`

	ScheduledStartAt time.Time `gorm:"not null;index" json:"scheduled_start_at"`

	// BracketExponent and MaxParticipants are copied from the schedule at open/lock for immutability.
	BracketExponent uint8  `gorm:"not null" json:"bracket_exponent"`
	MaxParticipants uint32 `gorm:"not null" json:"max_participants"` // = 2^BracketExponent

	// BracketSlots is the effective bracket width at lock (power of two, <= MaxParticipants, >= in-bracket count).
	BracketSlots uint32 `gorm:"not null" json:"bracket_slots"`

	Status TournamentStatus `gorm:"not null;index" json:"status"`

	LockedAt    *time.Time `json:"locked_at,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	Schedule *TournamentSchedule `json:"schedule,omitempty"`
	Entries  []*TournamentEntry  `json:"entries,omitempty"`
	Matches  []*TournamentMatch  `json:"matches,omitempty"`
}

func (Tournament) TableName() string { return "tournaments" }

// --- Entrant in a tournament (queue + bracket position) ---

type TournamentEntryStatus uint8

const (
	TournamentEntryStatusUnknown TournamentEntryStatus = iota
	TournamentEntryStatusQueued           // in queue before lock
	TournamentEntryStatusInBracket        // among the first 2^n after lock
	TournamentEntryStatusKickedOverflow   // joined but excluded because count > 2^n at lock
	TournamentEntryStatusEliminated
	TournamentEntryStatusWinner // final winner
	TournamentEntryStatusWithdrew
)

type TournamentEntry struct {
	BaseModel

	TournamentID uint `gorm:"not null;uniqueIndex:uq_tournament_player,priority:1;index:idx_tournament_entries_join,priority:1" json:"tournament_id"`

	// JoinSequence is assigned at enqueue time, monotonic per tournament. On game tie, smaller JoinSequence promotes.
	JoinSequence int64 `gorm:"not null;index:idx_tournament_entries_join,priority:2" json:"join_sequence"`

	PlayerId         int64  `gorm:"not null;uniqueIndex:uq_tournament_player,priority:2;index" json:"player_id"`
	TemporaryAddress string `gorm:"not null;size:128;uniqueIndex:uq_tournament_player,priority:3" json:"temporary_address"`

	Status TournamentEntryStatus `gorm:"not null;index" json:"status"`

	// SeedPosition is assigned at lock: 0 .. MaxParticipants-1 for InBracket; unset (0) for kicked if we clear — use status KickedOverflow and SeedPosition 0 meaning N/A.
	SeedPosition uint32 `gorm:"not null" json:"seed_position"`

	// EliminatedRound is 0 while active or winner; set when eliminated (1 = first elimination round, ...).
	EliminatedRound uint32 `gorm:"not null" json:"eliminated_round"`

	Tournament *Tournament `json:"tournament,omitempty"`
}

func (TournamentEntry) TableName() string { return "tournament_entries" }

// --- Bracket match (links two entries to a game; winner promotion) ---

type TournamentMatch struct {
	BaseModel

	TournamentID uint `gorm:"not null;uniqueIndex:uq_tournament_round_match,priority:1;index" json:"tournament_id"`
	RoundNumber  uint32 `gorm:"not null;uniqueIndex:uq_tournament_round_match,priority:2" json:"round_number"` // 1 = first KO round
	MatchIndex   uint32 `gorm:"not null;uniqueIndex:uq_tournament_round_match,priority:3" json:"match_index"`   // 0-based within round

	EntryAID *uint `gorm:"index" json:"entry_a_id,omitempty"`
	EntryBID *uint `gorm:"index" json:"entry_b_id,omitempty"`

	GameID *uint `gorm:"index" json:"game_id,omitempty"`

	// WinnerEntryID is set when the match is decided. On a tie, the entry with the smaller JoinSequence is promoted (application logic).
	WinnerEntryID *uint `gorm:"index" json:"winner_entry_id,omitempty"`

	EntryA *TournamentEntry `json:"entry_a,omitempty"`
	EntryB *TournamentEntry `json:"entry_b,omitempty"`
	Game   *Game            `json:"game,omitempty"`
	Winner *TournamentEntry `json:"winner,omitempty"`
}

func (TournamentMatch) TableName() string { return "tournament_matches" }
