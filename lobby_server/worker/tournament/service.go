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
func NewTournamentQueueService(ctx context.Context, gameCreator GameCreator, entryFee int32, intervalSeconds uint32, beforeStartSeconds uint32) *TournamentQueueService {
	return &TournamentQueueService{
		ctx:   ctx,
		coord: newCoordinator(ctx, gameCreator, entryFee, intervalSeconds, beforeStartSeconds),
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

	open, err := db.TournamentGetRegistrationOpenByTournamentIDBeforeDeadline(TournamentID, time.Now())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("Exceed the registration deadline for tournament %s", TournamentID)
		}
		return err
	}

	if existing, err := db.TournamentGetParticipantByPlayer(open.TournamentID, address.Id, temp); err == nil && existing != nil {
		return fmt.Errorf("Already joined the tournament %s", TournamentID)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if err := db.LockUserToken(s.ctx, address.Id, temp, open.EntryFee, TournamentID); err != nil {
		return err
	}

	p := &dao.TournamentParticipant{
		TournamentID: open.TournamentID,
		PlayerID:     address.Id,
		TempAddress:  temp,
		Status:       dao.TournamentParticipantStatusQueued,
	}
	return db.Get().Create(p).Error
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
