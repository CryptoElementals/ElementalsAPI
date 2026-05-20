package tournament

import (
	"context"
	"testing"
	"time"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
)

type noopGameCreator struct{}
type noopPublisher struct{}

func (n *noopGameCreator) CreateGameAndRun(_ []types.PlayerAddress, _ proto.GameType, _ int64) (int64, error) {
	return 1, nil
}

func (noopPublisher) Publish(_ context.Context, _ *proto.Event) (*proto.PublishResponse, error) {
	return &proto.PublishResponse{Success: true}, nil
}

func (noopPublisher) Topic() string { return "test-topic" }

func setupSQLite(t *testing.T) {
	t.Helper()
	require.NoError(t, db.Init(&db.Config{Development: true}))
	require.NoError(t, db.MigrateMemDb())
}

func seedUserToken(t *testing.T, playerID int64, tokenAmount int32) {
	t.Helper()
	ut := &dao.UserToken{
		PlayerId:    playerID,
		TokenAmount: tokenAmount,
	}
	require.NoError(t, db.Get().Create(ut).Error)
}

func seedTournament(t *testing.T, tournamentID string, deadline time.Time) {
	t.Helper()
	tour := &dao.Tournament{
		TournamentID:         tournamentID,
		Status:               dao.TournamentStatusRegistrationOpen,
		ScheduledStartAt:     time.Now().Add(time.Hour),
		RegistrationDeadline: deadline,
		EntryFee:             1000,
	}
	require.NoError(t, db.Get().Create(tour).Error)
}

func TestHandleJoinTournamentEvent_Success(t *testing.T) {
	setupSQLite(t)
	seedUserToken(t, 5001, 5000)
	seedTournament(t, "tour-success", time.Now().Add(10*time.Minute))

	svc := NewTournamentQueueService(context.Background(), noopPublisher{}, noopPublisher{}, nil, &noopGameCreator{}, 1000, 2, 3600, 180, 180, 15, 10)
	req := &proto.PlayerAddress{Id: 5001, TemporaryAddress: " 0xABCDEF "}
	require.NoError(t, svc.HandleJoinTournamentEvent("tour-success", req))

	p, err := db.TournamentGetParticipantByPlayer("tour-success", 5001, "0xabcdef")
	require.NoError(t, err)
	require.Equal(t, dao.TournamentParticipantStatusQueued, p.Status)
	require.Equal(t, "0xabcdef", p.TempAddress)

	var ut dao.UserToken
	require.NoError(t, db.Get().Where("player_id = ?", int64(5001)).First(&ut).Error)
	require.EqualValues(t, 4000, ut.TokenAmount)

	var ledger dao.TournamentEntryLedger
	require.NoError(t, db.Get().
		Where("tournament_id = ? AND player_id = ? AND reason = ?", "tour-success", int64(5001), "join").
		First(&ledger).Error)
	require.Equal(t, dao.TournamentEntryLedgerDirectionEntryDeduct, ledger.Direction)
	require.EqualValues(t, 1000, ledger.Amount)
}

func TestHandleJoinTournamentEvent_DuplicateJoinRejected(t *testing.T) {
	setupSQLite(t)
	seedUserToken(t, 5002, 5000)
	seedTournament(t, "tour-dup", time.Now().Add(10*time.Minute))

	svc := NewTournamentQueueService(context.Background(), noopPublisher{}, noopPublisher{}, nil, &noopGameCreator{}, 1000, 2, 3600, 180, 180, 15, 10)
	req := &proto.PlayerAddress{Id: 5002, TemporaryAddress: "0xdup"}
	require.NoError(t, svc.HandleJoinTournamentEvent("tour-dup", req))

	err := svc.HandleJoinTournamentEvent("tour-dup", req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Already joined the tournament")

	var cnt int64
	require.NoError(t, db.Get().Model(&dao.TournamentParticipant{}).
		Where("tournament_id = ? AND player_id = ?", "tour-dup", int64(5002)).
		Count(&cnt).Error)
	require.EqualValues(t, 1, cnt)
}

func TestHandleJoinTournamentEvent_RejectWhenDeadlineExceeded(t *testing.T) {
	setupSQLite(t)
	seedUserToken(t, 5003, 5000)
	seedTournament(t, "tour-expired", time.Now().Add(-1*time.Minute))

	svc := NewTournamentQueueService(context.Background(), noopPublisher{}, noopPublisher{}, nil, &noopGameCreator{}, 1000, 2, 3600, 180, 180, 15, 10)
	err := svc.HandleJoinTournamentEvent("tour-expired", &proto.PlayerAddress{Id: 5003, TemporaryAddress: "0xlate"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "registration deadline has passed")

	var cnt int64
	require.NoError(t, db.Get().Model(&dao.TournamentParticipant{}).
		Where("tournament_id = ? AND player_id = ?", "tour-expired", int64(5003)).
		Count(&cnt).Error)
	require.EqualValues(t, 0, cnt)
}

func TestHandleJoinTournamentEvent_RejectWhenTokenNotEnough(t *testing.T) {
	setupSQLite(t)
	seedUserToken(t, 5004, 500)
	seedTournament(t, "tour-no-token", time.Now().Add(10*time.Minute))

	svc := NewTournamentQueueService(context.Background(), noopPublisher{}, noopPublisher{}, nil, &noopGameCreator{}, 1000, 2, 3600, 180, 180, 15, 10)
	err := svc.HandleJoinTournamentEvent("tour-no-token", &proto.PlayerAddress{Id: 5004, TemporaryAddress: "0xpoor"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "user token amount is not enough")

	var cnt int64
	require.NoError(t, db.Get().Model(&dao.TournamentParticipant{}).
		Where("tournament_id = ? AND player_id = ?", "tour-no-token", int64(5004)).
		Count(&cnt).Error)
	require.EqualValues(t, 0, cnt)
}
