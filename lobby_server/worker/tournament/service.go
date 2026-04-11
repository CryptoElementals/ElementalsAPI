package tournament

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"gorm.io/gorm"
)

// TournamentQueueService runs tournament schedules, registration, bracket lock, and match progression.
type TournamentQueueService struct {
	ctx   context.Context
	coord *coordinator
}

// NewTournamentQueueService constructs the tournament worker. Call Start after DB is ready.
func NewTournamentQueueService(ctx context.Context, gameCreator GameCreator, entryFee int32, minPlayersRequired uint32, intervalSeconds uint32, beforeStartSeconds uint32) *TournamentQueueService {
	return &TournamentQueueService{
		ctx:   ctx,
		coord: newCoordinator(ctx, gameCreator, entryFee, minPlayersRequired, intervalSeconds, beforeStartSeconds),
	}
}

func (s *TournamentQueueService) Start() {
	log.Debugw("tournament tournament queue service start")
	s.coord.start()
}

func (s *TournamentQueueService) Stop() {
	s.coord.stop()
}

// HandleJoinTournament registers for the open tournament on the given schedule (same token lock as PVP queue).
func (s *TournamentQueueService) HandleJoinTournamentEvent(TournamentID string, req *proto.PlayerAddress) error {
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

	t, err := db.TournamentGetByTournamentID(TournamentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("TournamentID %s invalid, no tournament found", TournamentID)
		}
		return fmt.Errorf("load tournament %s: %w", TournamentID, err)
	}

	if t.Status != dao.TournamentStatusRegistrationOpen {
		return fmt.Errorf("tournament %s is not open for registration (status: %s)", TournamentID, t.Status)
	}

	if existing, err := db.TournamentGetParticipantByPlayer(t.TournamentID, address.Id, temp); err == nil && existing != nil {
		return fmt.Errorf("Already joined the tournament %s", TournamentID)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if !now.Before(t.RegistrationDeadline) {
		return fmt.Errorf(
			"registration deadline has passed for tournament %s (deadline: %s, now: %s)",
			TournamentID,
			t.RegistrationDeadline.UTC().Format(time.RFC3339),
			now.UTC().Format(time.RFC3339),
		)
	}

	if err := db.LockUserToken(s.ctx, address.Id, temp, t.EntryFee, TournamentID); err != nil {
		return err
	}

	p := &dao.TournamentParticipant{
		TournamentID: t.TournamentID,
		PlayerID:     address.Id,
		TempAddress:  temp,
		Status:       dao.TournamentParticipantStatusQueued,
	}
	return db.Get().Create(p).Error
}

// GameResultSettlementHook advances the bracket when a tournament match ends. Invoke after queue battle settlement.
func (s *TournamentQueueService) GameResultSettlementHook(event *types.GameCompletedEvent) error {
	if event == nil {
		log.Errorw("tournament: game result settlement hook: event is nil")
		return nil
	}
	if event.GameInfo == nil {
		log.Errorw("tournament: game result settlement hook: game info is nil", "game_id", event.GameInfo.ID, "game_info", event.GameInfo)
		return nil
	}
	if event.GameInfo.Type != types.GameTypeTournament {
		log.Errorw("tournament: game result settlement hook: invalid game type", "game_id", event.GameInfo.ID, "game_type", event.GameInfo.Type)
		return nil
	}

	log.Debugw("tournament: game result settlement hook", "game_id", event.GameInfo.ID)
	return s.coord.onGameCompleted(event.GameInfo)
}
