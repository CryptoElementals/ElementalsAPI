package db

import (
	"time"

	dao "github.com/CryptoElementals/common/models"
	"gorm.io/gorm"
)

// TournamentGetLatestByScheduledStart returns the newest tournament by scheduled_start_at.
func TournamentGetLatestByScheduledStart() (*dao.Tournament, error) {
	var t dao.Tournament
	err := Get().Order("scheduled_start_at DESC").First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TournamentGetByScheduledStart finds a tournament exactly at scheduled_start_at.
func TournamentGetByScheduledStart(at time.Time) (*dao.Tournament, error) {
	var t dao.Tournament
	err := Get().Where("scheduled_start_at = ?", at).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TournamentCreate inserts a tournament.
func TournamentCreate(t *dao.Tournament) error {
	return Get().Create(t).Error
}

// TournamentSave updates a tournament row.
func TournamentSave(t *dao.Tournament) error {
	return Get().Save(t).Error
}

// TournamentGetByID loads tournament by id.
func TournamentGetByID(id uint) (*dao.Tournament, error) {
	var t dao.Tournament
	if err := Get().First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// TournamentGetByTournamentID loads a tournament by business tournament_id (string).
func TournamentGetByTournamentID(tournamentID string) (*dao.Tournament, error) {
	var t dao.Tournament
	err := Get().Where("tournament_id = ?", tournamentID).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TournamentGetLatestInProgress returns the newest in_progress tournament by scheduled_start_at.
func TournamentGetLatestInProgress() (*dao.Tournament, error) {
	var t dao.Tournament
	err := Get().Where("status = ?", dao.TournamentStatusInProgress).
		Order("scheduled_start_at DESC").First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TournamentPlayerInActiveBracket is true when the player is still tied to the latest
// in-progress tournament bracket only (queued before first match or currently in a match).
// This intentionally ignores older in_progress rows (possible dirty historical data).
func TournamentPlayerInActiveBracket(playerID int64, tempAddress string) (bool, error) {
	latestInProgress, err := TournamentGetLatestInProgress()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	if !time.Now().UTC().Before(latestInProgress.ScheduledEndDeadline) {
		return false, nil
	}

	var n int64
	err = Get().Raw(`
		SELECT COUNT(1) FROM tournament_participants p
		WHERE p.player_id = ? AND LOWER(p.temp_address) = LOWER(?)
		AND p.tournament_id = ?
		AND p.status IN (?, ?)`,
		playerID, tempAddress,
		latestInProgress.TournamentID,
		dao.TournamentParticipantStatusQueued,
		dao.TournamentParticipantStatusInProgress,
	).Scan(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// TournamentCountParticipantsForPool counts participants whose entry still counts toward the displayed prize pool
// (excludes kicked_not_enough and kicked_overflow).
func TournamentCountParticipantsForPool(tournamentBusinessID string) (int64, error) {
	var n int64
	err := Get().Model(&dao.TournamentParticipant{}).
		Where("tournament_id = ? AND status NOT IN ?",
			tournamentBusinessID,
			[]string{
				string(dao.TournamentParticipantStatusKickedNotEnough),
				string(dao.TournamentParticipantStatusKickedOverflow),
			}).
		Count(&n).Error
	return n, err
}

// TournamentGetLatestRegistrationOpen returns latest registration_open tournament by scheduled_start_at.
func TournamentGetLatestRegistrationOpen() (*dao.Tournament, error) {
	var t dao.Tournament
	err := Get().Where("status = ?", dao.TournamentStatusRegistrationOpen).
		Order("scheduled_start_at DESC").First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TournamentGetLatestRegistrationOpenBeforeDeadline returns latest registration_open tournament
// whose registration_deadline is still in the future.
func TournamentGetLatestRegistrationOpenBeforeDeadline(now time.Time) (*dao.Tournament, error) {
	var t dao.Tournament
	err := Get().
		Where("status = ? AND registration_deadline > ?", dao.TournamentStatusRegistrationOpen, now).
		Order("scheduled_start_at DESC").
		First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TournamentGetLatestRegistrationOpenBeforeStart returns latest registration_open tournament
// whose scheduled_start_at is still in the future.
func TournamentGetLatestRegistrationOpenBeforeStart(now time.Time) (*dao.Tournament, error) {
	var t dao.Tournament
	err := Get().
		Where("status = ? AND scheduled_start_at > ?", dao.TournamentStatusRegistrationOpen, now).
		Order("scheduled_start_at DESC").
		First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TournamentGetRegistrationOpenByTournamentIDBeforeDeadline returns a specific tournament
// by business tournament_id when registration is still open and not expired.
func TournamentGetRegistrationOpenByTournamentIDBeforeDeadline(tournamentID string, now time.Time) (*dao.Tournament, error) {
	var t dao.Tournament
	err := Get().
		Where("tournament_id = ? AND status = ? AND registration_deadline > ?", tournamentID, dao.TournamentStatusRegistrationOpen, now).
		First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// TournamentListRegistrationOpenPastScheduled lists registration_open tournaments whose scheduled start has arrived.
func TournamentListRegistrationOpenPastScheduled(before time.Time) ([]dao.Tournament, error) {
	var rows []dao.Tournament
	err := Get().Where("status = ? AND scheduled_start_at <= ?", dao.TournamentStatusRegistrationOpen, before).
		Find(&rows).Error
	return rows, err
}

// TournamentListRegistrationOpenReachedDeadline lists registration_open tournaments whose registration deadline has arrived.
func TournamentListRegistrationOpenReachedDeadline(now time.Time) ([]dao.Tournament, error) {
	var rows []dao.Tournament
	err := Get().
		Where("status = ? AND registration_deadline <= ?", dao.TournamentStatusRegistrationOpen, now).
		Order("scheduled_start_at ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

// TournamentListRegistrationOpenInBotFillWindow lists registration_open tournaments whose registration deadline
// is within [now, now+window], used for progressive bot fill before deadline.
func TournamentListRegistrationOpenInBotFillWindow(now time.Time, window time.Duration) ([]dao.Tournament, error) {
	if window <= 0 {
		return []dao.Tournament{}, nil
	}
	var rows []dao.Tournament
	end := now.Add(window)
	err := Get().
		Where("status = ? AND registration_deadline >= ? AND registration_deadline <= ?", dao.TournamentStatusRegistrationOpen, now, end).
		Order("registration_deadline ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

// TournamentGetLatestRegistrationOpenWithinStartGrace returns one latest registration_open tournament
// whose scheduled_start_at is within [now-grace, now]. This allows small scheduler/restart delays.
func TournamentGetLatestRegistrationOpenWithinStartGrace(now time.Time, grace time.Duration) (*dao.Tournament, error) {
	if grace < 0 {
		grace = 0
	}
	startFrom := now.Add(-grace)
	var t dao.Tournament
	err := Get().
		Where("status = ? AND scheduled_start_at >= ? AND scheduled_start_at <= ?",
			dao.TournamentStatusRegistrationOpen, startFrom, now).
		Order("scheduled_start_at DESC, id DESC").
		First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// --- tournament_participants ---

func TournamentGetParticipantByPlayer(tournamentBusinessID string, playerID int64, tempAddr string) (*dao.TournamentParticipant, error) {
	var p dao.TournamentParticipant
	err := Get().Where("tournament_id = ? AND player_id = ? AND LOWER(temp_address) = LOWER(?)",
		tournamentBusinessID, playerID, tempAddr).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// TournamentGetParticipantByPlayerTx loads a participant row inside tx.
func TournamentGetParticipantByPlayerTx(tx *gorm.DB, tournamentBusinessID string, playerID int64, tempAddr string) (*dao.TournamentParticipant, error) {
	var p dao.TournamentParticipant
	err := tx.Where("tournament_id = ? AND player_id = ? AND LOWER(temp_address) = LOWER(?)",
		tournamentBusinessID, playerID, tempAddr).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// TournamentGetParticipantByBusinessTournamentIDAndTemp loads a participant by business tournament_id and temp address.
func TournamentGetParticipantByBusinessTournamentIDAndTemp(tournamentBusinessID string, tempAddr string) (*dao.TournamentParticipant, error) {
	var p dao.TournamentParticipant
	err := Get().Where("tournament_id = ? AND LOWER(temp_address) = LOWER(?)",
		tournamentBusinessID, tempAddr).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func TournamentCreateParticipant(tx *gorm.DB, p *dao.TournamentParticipant) error {
	return tx.Create(p).Error
}

func TournamentSaveParticipant(tx *gorm.DB, p *dao.TournamentParticipant) error {
	return tx.Save(p).Error
}

func TournamentDeleteParticipant(tx *gorm.DB, p *dao.TournamentParticipant) error {
	return tx.Delete(p).Error
}

func TournamentListParticipantsByStatus(tx *gorm.DB, tournamentID string, status dao.TournamentParticipantStatus) ([]dao.TournamentParticipant, error) {
	var rows []dao.TournamentParticipant
	err := tx.Where("tournament_id = ? AND status = ?", tournamentID, status).
		Order("created_at ASC").Find(&rows).Error
	if err != nil {
		return []dao.TournamentParticipant{}, err
	}
	return rows, nil
}

// --- tournament_rounds ---

func TournamentCreateRound(tx *gorm.DB, r *dao.TournamentRound) error {
	return tx.Create(r).Error
}

func TournamentSaveRound(tx *gorm.DB, r *dao.TournamentRound) error {
	return tx.Save(r).Error
}

func TournamentGetRound(tx *gorm.DB, tournamentID string, roundNo uint32) (*dao.TournamentRound, error) {
	var r dao.TournamentRound
	if err := tx.Where("tournament_id = ? AND round_no = ?", tournamentID, roundNo).First(&r).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

// --- tournament_matches ---

func TournamentListMatchesForRound(tx *gorm.DB, tournamentBusinessID string, roundNo uint32) ([]dao.TournamentMatch, error) {
	var rows []dao.TournamentMatch
	err := tx.Where("tournament_id = ? AND round_no = ?", tournamentBusinessID, roundNo).
		Order("match_no ASC").Find(&rows).Error
	return rows, err
}

func TournamentGetMatchByGameID(gameID int64) (*dao.TournamentMatch, error) {
	var m dao.TournamentMatch
	err := Get().Where("game_id = ?", gameID).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// TournamentGetMatchByGameIDTx loads a tournament match by game_id using tx (e.g. with FOR UPDATE).
func TournamentGetMatchByGameIDTx(tx *gorm.DB, gameID int64) (*dao.TournamentMatch, error) {
	var m dao.TournamentMatch
	if err := tx.Where("game_id = ?", gameID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func TournamentCreateMatch(tx *gorm.DB, m *dao.TournamentMatch) error {
	return tx.Create(m).Error
}

func TournamentSaveMatch(tx *gorm.DB, m *dao.TournamentMatch) error {
	return tx.Save(m).Error
}

func TournamentLoadMatchByID(tx *gorm.DB, id uint) (*dao.TournamentMatch, error) {
	var m dao.TournamentMatch
	err := tx.First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// TournamentPlayerQueuedOrInProgressInOpenTournaments inspects tournaments with status
// registration_open or in_progress and the player's participant row(s) in those tournaments.
// If any such row has status in_progress, returns in_progress. Else if any has queued, returns queued.
// Otherwise returns an empty status (no queued/in_progress participation in those tournaments).
func TournamentPlayerQueuedOrInProgressInOpenTournaments(playerID int64, tempAddress string) (dao.TournamentParticipantStatus, error) {
	var statuses []string
	err := Get().Raw(`
		SELECT p.status FROM tournament_participants AS p
		INNER JOIN tournaments AS t ON t.tournament_id = p.tournament_id
		WHERE t.status IN (?, ?)
		  AND p.player_id = ? AND LOWER(p.temp_address) = LOWER(?)
		  AND p.status IN (?, ?)`,
		dao.TournamentStatusRegistrationOpen,
		dao.TournamentStatusInProgress,
		playerID, tempAddress,
		dao.TournamentParticipantStatusQueued,
		dao.TournamentParticipantStatusInProgress,
	).Scan(&statuses).Error
	if err != nil {
		return "", err
	}
	var hasInProgress, hasQueued bool
	for _, s := range statuses {
		switch dao.TournamentParticipantStatus(s) {
		case dao.TournamentParticipantStatusInProgress:
			hasInProgress = true
		case dao.TournamentParticipantStatusQueued:
			hasQueued = true
		}
	}
	if hasInProgress {
		return dao.TournamentParticipantStatusInProgress, nil
	}
	if hasQueued {
		return dao.TournamentParticipantStatusQueued, nil
	}
	return "", nil
}
