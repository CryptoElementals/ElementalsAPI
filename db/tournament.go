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

// TournamentGetLatestRegistrationOpenPastScheduled returns one latest registration_open tournament
// whose scheduled_start_at has arrived.
func TournamentGetLatestRegistrationOpenWithSlot(slot time.Time) (*dao.Tournament, error) {
	var t dao.Tournament
	err := Get().
		Where("status = ? AND scheduled_start_at = ?", dao.TournamentStatusRegistrationOpen, slot).
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
