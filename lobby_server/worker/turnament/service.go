package turnament

import (
	"context"
	"errors"
	"strings"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

// TournamentQueueService runs tournament schedules, registration, bracket lock, and match progression.
type TournamentQueueService struct {
	ctx                 context.Context
	minTokenToJoinQueue int32
	coord               *coordinator
}

// NewTournamentQueueService constructs the tournament worker. Call Start after DB is ready.
func NewTournamentQueueService(ctx context.Context, gameCreator GameCreator, minTokenToJoinQueue int32) *TournamentQueueService {
	return &TournamentQueueService{
		ctx:                 ctx,
		minTokenToJoinQueue: minTokenToJoinQueue,
		coord:               newCoordinator(ctx, gameCreator),
	}
}

func (s *TournamentQueueService) Start() {
	s.coord.start()
}

func (s *TournamentQueueService) Stop() {
	s.coord.stop()
}

// HandleJoinTournament registers for the open tournament on the given schedule (same token lock as PVP queue).
func (s *TournamentQueueService) HandleJoinTournament(scheduleID uint, req *proto.PlayerAddress) error {
	var address types.PlayerAddress
	address.FromProto(req)
	temp := strings.ToLower(strings.TrimSpace(address.TemporaryAddress))

	sched, err := db.TournamentGetSchedule(scheduleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("tournament: schedule not found")
		}
		return err
	}
	if !sched.Enabled {
		return errors.New("tournament: schedule disabled")
	}
	if dao.TournamentBracketCapacity(sched.BracketExponent) == 0 {
		return errors.New("tournament: invalid bracket size")
	}

	open, err := db.TournamentGetOpenByScheduleID(scheduleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("tournament: no open registration for this schedule")
		}
		return err
	}

	if existing, err := db.TournamentGetEntryByPlayer(open.ID, address.Id, temp); err == nil && existing != nil {
		if existing.Status == dao.TournamentEntryStatusQueued {
			return errors.New("tournament: already registered")
		}
		return errors.New("tournament: already in this tournament")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if err := db.LockUserToken(s.ctx, address.Id, temp, s.minTokenToJoinQueue); err != nil {
		return err
	}

	return db.Get().Transaction(func(tx *gorm.DB) error {
		seq, err := db.TournamentNextJoinSequence(tx, open.ID)
		if err != nil {
			return err
		}
		e := &dao.TournamentEntry{
			TournamentID:     open.ID,
			JoinSequence:     seq,
			PlayerId:         address.Id,
			TemporaryAddress: temp,
			Status:           dao.TournamentEntryStatusQueued,
			SeedPosition:     0,
			EliminatedRound:  0,
		}
		return db.TournamentCreateEntry(tx, e)
	})
}

// HandleExitTournament removes a queued player and unlocks tokens.
func (s *TournamentQueueService) HandleExitTournament(scheduleID uint, req *proto.PlayerAddress) error {
	var address types.PlayerAddress
	address.FromProto(req)
	temp := strings.ToLower(strings.TrimSpace(address.TemporaryAddress))

	open, err := db.TournamentGetOpenByScheduleID(scheduleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	entry, err := db.TournamentGetEntryByPlayer(open.ID, address.Id, temp)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if entry.Status != dao.TournamentEntryStatusQueued {
		return errors.New("tournament: cannot leave after bracket is locked")
	}
	if err := db.Get().Delete(entry).Error; err != nil {
		return err
	}
	return db.UnlockUserToken(s.ctx, address.Id, temp)
}

// GameResultSettlementHook advances the bracket when a tournament match ends. Invoke after queue battle settlement.
func (s *TournamentQueueService) GameResultSettlementHook(event *types.GameCompletedEvent) error {
	if event == nil || event.GameInfo == nil || event.GameInfo.Type != types.GameTypeTournament {
		return nil
	}
	full, err := db.LoadGameByGameID(event.GameInfo.ID)
	if err != nil {
		return err
	}
	if full.Type != types.GameTypeTournament {
		return nil
	}
	return s.coord.onGameCompleted(full)
}
