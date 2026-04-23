package tournament

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/bot_manager"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/pubsub"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TournamentQueueService runs tournament schedules, registration, bracket lock, and match progression.
type TournamentQueueService struct {
	ctx   context.Context
	coord *coordinator
}

// NewTournamentQueueService constructs the tournament worker. Call Start after DB is ready.
func NewTournamentQueueService(ctx context.Context, publisher pubsub.Publisher, botStore *bot_manager.RedisStore, gameCreator GameCreator, entryFee int32, minPlayersRequired uint32, intervalSeconds uint32, beforeStartSeconds uint32, botFillWindowSeconds uint32, botFillIntervalSeconds uint32, botFreshnessSec int64) *TournamentQueueService {
	svc := &TournamentQueueService{
		ctx:   ctx,
		coord: newCoordinator(ctx, publisher, botStore, gameCreator, entryFee, minPlayersRequired, intervalSeconds, beforeStartSeconds, botFillWindowSeconds, botFillIntervalSeconds, botFreshnessSec),
	}
	svc.coord.joinTournamentFunc = func(tournamentID string, req *proto.PlayerAddress) error {
		return svc.handleJoinTournamentEvent(tournamentID, req, true)
	}
	return svc
}

func (s *TournamentQueueService) Start() {
	log.Debugw("tournament tournament queue service start")
	s.coord.start()
}

func (s *TournamentQueueService) Stop() {
	s.coord.stop()
}

func (s *TournamentQueueService) SetTournamentSchedulingEnabled(enabled bool) {
	if s == nil || s.coord == nil {
		return
	}
	s.coord.setTournamentCreationEnabled(enabled)
}

func (s *TournamentQueueService) IsTournamentSchedulingEnabled() bool {
	if s == nil || s.coord == nil {
		return false
	}
	return s.coord.isTournamentCreationEnabled()
}

// HandleJoinTournament registers for the open tournament on the given schedule (same token lock as PVP queue).
func (s *TournamentQueueService) HandleJoinTournamentEvent(TournamentID string, req *proto.PlayerAddress) error {
	return s.handleJoinTournamentEvent(TournamentID, req, false)
}

func (s *TournamentQueueService) handleJoinTournamentEvent(TournamentID string, req *proto.PlayerAddress, allowAfterDeadline bool) error {
	var address types.PlayerAddress
	address.FromProto(req)
	temp := strings.ToLower(strings.TrimSpace(address.TemporaryAddress))

	if address.Id == 0 {
		return fmt.Errorf("invalid player id: 0")
	}
	if _, err := db.GetUserProfileByPlayerIDInt(address.Id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("player not found for player_id %d", address.Id)
		}
		return fmt.Errorf("load user profile for player_id %d: %w", address.Id, err)
	}

	now := time.Now().UTC()

	if err := db.Get().Transaction(func(tx *gorm.DB) error {
		var lockedT dao.Tournament
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tournament_id = ?", TournamentID).
			First(&lockedT).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("TournamentID %s invalid, no tournament found", TournamentID)
			}
			return err
		}
		if lockedT.Status != dao.TournamentStatusRegistrationOpen {
			return fmt.Errorf("tournament %s is not open for registration (status: %s)", TournamentID, lockedT.Status)
		}
		if !allowAfterDeadline && !now.Before(lockedT.RegistrationDeadline) {
			return fmt.Errorf(
				"registration deadline has passed for tournament %s (deadline: %s, now: %s)",
				TournamentID,
				lockedT.RegistrationDeadline.UTC().Format(time.RFC3339),
				now.UTC().Format(time.RFC3339),
			)
		}
		if existing, err := db.TournamentGetParticipantByPlayerTx(tx, lockedT.TournamentID, address.Id, temp); err == nil && existing != nil {
			return fmt.Errorf("Already joined the tournament %s", TournamentID)
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := db.DeductUserTokenForTournamentEntryTx(tx, address.Id, lockedT.EntryFee); err != nil {
			return err
		}
		p := &dao.TournamentParticipant{
			TournamentID: lockedT.TournamentID,
			PlayerID:     address.Id,
			TempAddress:  temp,
			Status:       dao.TournamentParticipantStatusQueued,
		}
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		return db.RecordTournamentEntryLedgerTx(
			tx,
			lockedT.TournamentID,
			address.Id,
			temp,
			lockedT.EntryFee,
			dao.TournamentEntryLedgerDirectionEntryDeduct,
			"join",
		)
	}); err != nil {
		return err
	}
	if err := s.coord.publishTournamentRosterUpdate(TournamentID); err != nil {
		log.Errorw("tournament: publish roster update failed", "tournament_id", TournamentID, "player_id", address.Id, "temp_address", temp, "err", err)
		return err
	}
	log.Debugw("tournament: publish roster update success", "tournament_id", TournamentID, "player_id", address.Id, "temp_address", temp)
	return nil
}

// GameResultSettlementHook applies tournament match rewards (tournament_rewards + wallet) and advances the bracket when a tournament match ends.
// Uses game_id + tournament_matches + game_results only (no full games row load).
func (s *TournamentQueueService) GameResultSettlementHook(event *types.GameCompletedEvent) error {
	if event == nil {
		log.Errorw("tournament: game result settlement hook: event is nil")
		return nil
	}
	if event.GameID == 0 {
		return nil
	}
	if _, err := db.TournamentGetMatchByGameID(event.GameID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	log.Debugw("tournament: game result settlement hook", "game_id", event.GameID)
	return s.coord.onGameCompleted(event.GameID)
}
