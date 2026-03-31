package db

import (
	"errors"
	"time"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
)

// TournamentListEnabledSchedules returns schedules with Enabled=true.
func TournamentListEnabledSchedules() ([]dao.TournamentSchedule, error) {
	var rows []dao.TournamentSchedule
	if err := Get().Where("enabled = ?", true).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// TournamentGetSchedule loads a schedule by id.
func TournamentGetSchedule(id uint) (*dao.TournamentSchedule, error) {
	var s dao.TournamentSchedule
	if err := Get().First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// TournamentGetOpenByScheduleID returns the open registration tournament for this schedule, if any.
func TournamentGetOpenByScheduleID(scheduleID uint) (*dao.Tournament, error) {
	var t dao.Tournament
	err := Get().Where("tournament_schedule_id = ? AND status = ?", scheduleID, dao.TournamentStatusOpen).
		First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TournamentGetLatestByScheduleID returns the tournament with the highest instance_index for the schedule.
func TournamentGetLatestByScheduleID(scheduleID uint) (*dao.Tournament, error) {
	var t dao.Tournament
	err := Get().Where("tournament_schedule_id = ?", scheduleID).
		Order("instance_index DESC").First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TournamentCreate persists a new tournament row.
func TournamentCreate(t *dao.Tournament) error {
	return Get().Create(t).Error
}

// TournamentSave updates an existing tournament.
func TournamentSave(t *dao.Tournament) error {
	return Get().Save(t).Error
}

// TournamentListOpenPastScheduled lists tournaments that are still open but scheduled_start_at <= before.
func TournamentListOpenPastScheduled(before time.Time) ([]dao.Tournament, error) {
	var rows []dao.Tournament
	err := Get().Where("status = ? AND scheduled_start_at <= ?", dao.TournamentStatusOpen, before).
		Find(&rows).Error
	return rows, err
}

// TournamentNextJoinSequence returns max(join_sequence)+1 for the tournament (starts at 1).
func TournamentNextJoinSequence(tx *gorm.DB, tournamentID uint) (int64, error) {
	var last dao.TournamentEntry
	err := tx.Where("tournament_id = ?", tournamentID).
		Order("join_sequence DESC").
		First(&last).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	return last.JoinSequence + 1, nil
}

// TournamentGetEntryByPlayer returns a non-deleted entry for this player in this tournament, if any.
func TournamentGetEntryByPlayer(tournamentID uint, playerID int64, tempAddr string) (*dao.TournamentEntry, error) {
	var e dao.TournamentEntry
	err := Get().Where("tournament_id = ? AND player_id = ? AND LOWER(temporary_address) = LOWER(?)",
		tournamentID, playerID, tempAddr).First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// TournamentCreateEntry inserts an entry (use inside a transaction with TournamentNextJoinSequence).
func TournamentCreateEntry(tx *gorm.DB, e *dao.TournamentEntry) error {
	return tx.Create(e).Error
}

// TournamentSaveEntry updates an entry.
func TournamentSaveEntry(tx *gorm.DB, e *dao.TournamentEntry) error {
	return tx.Save(e).Error
}

// TournamentListQueuedEntries returns entries with status Queued ordered by join_sequence.
func TournamentListQueuedEntries(tx *gorm.DB, tournamentID uint) ([]dao.TournamentEntry, error) {
	var rows []dao.TournamentEntry
	err := tx.Where("tournament_id = ? AND status = ?", tournamentID, dao.TournamentEntryStatusQueued).
		Order("join_sequence ASC").Find(&rows).Error
	return rows, err
}

// TournamentListMatchesForRound returns matches for a round ordered by match_index.
func TournamentListMatchesForRound(tx *gorm.DB, tournamentID uint, roundNumber uint32) ([]dao.TournamentMatch, error) {
	var rows []dao.TournamentMatch
	err := tx.Where("tournament_id = ? AND round_number = ?", tournamentID, roundNumber).
		Order("match_index ASC").Find(&rows).Error
	return rows, err
}

// TournamentGetMatchByGameID finds a bracket match linked to a game.
func TournamentGetMatchByGameID(gameID uint) (*dao.TournamentMatch, error) {
	var m dao.TournamentMatch
	err := Get().Where("game_id = ?", gameID).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// TournamentCreateMatch inserts a match row.
func TournamentCreateMatch(tx *gorm.DB, m *dao.TournamentMatch) error {
	return tx.Create(m).Error
}

// TournamentSaveMatch updates a match.
func TournamentSaveMatch(tx *gorm.DB, m *dao.TournamentMatch) error {
	return tx.Save(m).Error
}

// TournamentLoadMatchByID loads a match with EntryA and EntryB preloaded (for join_sequence).
func TournamentLoadMatchByID(tx *gorm.DB, id uint) (*dao.TournamentMatch, error) {
	var m dao.TournamentMatch
	err := tx.Preload("EntryA").Preload("EntryB").First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// TournamentLoadEntry loads an entry by id.
func TournamentLoadEntry(tx *gorm.DB, id uint) (*dao.TournamentEntry, error) {
	var e dao.TournamentEntry
	if err := tx.First(&e, id).Error; err != nil {
		return nil, err
	}
	return &e, nil
}
