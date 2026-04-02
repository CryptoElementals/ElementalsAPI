package turnament

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Uses the global sqlite :memory: DB from db.Init(Development: true) — do not use t.Parallel().

func setupSQLite(t *testing.T) {
	t.Helper()
	require.NoError(t, db.Init(&db.Config{Development: true}))
	require.NoError(t, db.MigrateMemDb())
}

func seedGameArgs(t *testing.T) *dao.GameArgs {
	t.Helper()
	ga := &dao.GameArgs{
		MaxNormalRounds:                       3,
		MaxExtraRounds:                        0,
		MaxTurnsPerNormalRound:                3,
		MaxTurnsPerExtraRound:                 1,
		InitialHP:                             3000,
		InitialMultiplier:                     1,
		ConfirmationTimeout:                   60,
		CommitmentSubmissionTimeout:           60,
		CardSubmissionTimeout:                 60,
		GameContinueTimeout:                   120,
		ConfirmationTimeoutRedundancy:         10,
		CommitmentSubmissionTimeoutRedundancy: 10,
		CardSubmissionTimeoutRedundancy:       10,
		GameContinueTimeoutRedundancy:         10,
	}
	require.NoError(t, db.Get().Create(ga).Error)
	return ga
}

func seedProfileAndToken(t *testing.T, playerID int64, uniqueName, tempAddr string, tokenAmount int32) {
	t.Helper()
	prof := dao.UserProfile{
		PlayerID: playerID,
		Name:     uniqueName,
		Address:  "0x" + uniqueName,
	}
	require.NoError(t, db.Get().Create(&prof).Error)
	ut := dao.UserToken{
		PlayerId:    playerID,
		TokenAmount: tokenAmount,
	}
	require.NoError(t, db.Get().Create(&ut).Error)
}

func seedSchedule(t *testing.T, firstStart time.Time) *dao.TournamentSchedule {
	t.Helper()
	s := &dao.TournamentSchedule{
		Name:              "test-sched",
		BracketExponent:   dao.TournamentBracketExpMin,
		IntervalSeconds:   3600,
		FirstStartAt:      firstStart,
		Enabled:           true,
	}
	require.NoError(t, db.Get().Create(s).Error)
	return s
}

func openTournament(t *testing.T, scheduleID uint, scheduledStart time.Time) *dao.Tournament {
	t.Helper()
	capacity := dao.TournamentBracketCapacity(dao.TournamentBracketExpMin)
	tour := &dao.Tournament{
		TournamentScheduleID: scheduleID,
		InstanceIndex:        0,
		ScheduledStartAt:     scheduledStart,
		BracketExponent:      dao.TournamentBracketExpMin,
		MaxParticipants:      capacity,
		BracketSlots:         0,
		Status:               dao.TournamentStatusOpen,
	}
	require.NoError(t, db.Get().Create(tour).Error)
	return tour
}

type mockGameCreator struct {
	nextGameID uint
	onMatch    func([]types.PlayerAddress) (uint, error)
}

func (m *mockGameCreator) CreateGameAndRun(players []types.PlayerAddress, _ uint, _ int64) (uint, error) {
	if m.onMatch != nil {
		return m.onMatch(players)
	}
	m.nextGameID++
	return m.nextGameID, nil
}

func TestEnsureOpenTournament_CreatesFirstInstance(t *testing.T) {
	setupSQLite(t)
	sched := seedSchedule(t, time.Now().Add(time.Hour))
	tc := newCoordinator(context.Background(), &mockGameCreator{})
	require.NoError(t, tc.ensureOpenTournament(sched))

	open, err := db.TournamentGetOpenByScheduleID(sched.ID)
	require.NoError(t, err)
	require.Equal(t, dao.TournamentStatusOpen, open.Status)
	require.Equal(t, uint32(0), open.InstanceIndex)
	require.Equal(t, sched.BracketExponent, open.BracketExponent)
}

func TestEnsureOpenTournament_IdempotentWhenOpenExists(t *testing.T) {
	setupSQLite(t)
	sched := seedSchedule(t, time.Now().Add(time.Hour))
	tc := newCoordinator(context.Background(), &mockGameCreator{})
	require.NoError(t, tc.ensureOpenTournament(sched))
	require.NoError(t, tc.ensureOpenTournament(sched))

	var n int64
	require.NoError(t, db.Get().Model(&dao.Tournament{}).Where("tournament_schedule_id = ?", sched.ID).Count(&n).Error)
	require.EqualValues(t, 1, n)
}

func TestLockTournament_EmptyQueue_Cancelled(t *testing.T) {
	setupSQLite(t)
	sched := seedSchedule(t, time.Now().Add(time.Hour))
	tour := openTournament(t, sched.ID, time.Now().Add(-time.Minute))
	tc := newCoordinator(context.Background(), &mockGameCreator{})
	require.NoError(t, tc.lockTournament(tour))

	var reloaded dao.Tournament
	require.NoError(t, db.Get().First(&reloaded, tour.ID).Error)
	require.Equal(t, dao.TournamentStatusCancelled, reloaded.Status)
	require.NotNil(t, reloaded.LockedAt)
}

func TestLockTournament_SinglePlayer_CompletedWinner(t *testing.T) {
	setupSQLite(t)
	sched := seedSchedule(t, time.Now().Add(time.Hour))
	tour := openTournament(t, sched.ID, time.Now().Add(-time.Minute))
	require.NoError(t, db.Get().Create(&dao.TournamentEntry{
		TournamentID:     tour.ID,
		JoinSequence:     1,
		PlayerId:         9001,
		TemporaryAddress: "0xsolo",
		Status:           dao.TournamentEntryStatusQueued,
	}).Error)

	tc := newCoordinator(context.Background(), &mockGameCreator{})
	require.NoError(t, tc.lockTournament(tour))

	var reloaded dao.Tournament
	require.NoError(t, db.Get().First(&reloaded, tour.ID).Error)
	require.Equal(t, dao.TournamentStatusCompleted, reloaded.Status)
	require.EqualValues(t, 1, reloaded.BracketSlots)

	var ent dao.TournamentEntry
	require.NoError(t, db.Get().Where("tournament_id = ?", tour.ID).First(&ent).Error)
	require.Equal(t, dao.TournamentEntryStatusWinner, ent.Status)
}

func TestLockTournament_TwoPlayers_CreatesOneMatchAndBracketSlots(t *testing.T) {
	setupSQLite(t)
	sched := seedSchedule(t, time.Now().Add(time.Hour))
	tour := openTournament(t, sched.ID, time.Now().Add(-time.Minute))
	for i, pid := range []int64{9101, 9102} {
		require.NoError(t, db.Get().Create(&dao.TournamentEntry{
			TournamentID:     tour.ID,
			JoinSequence:     int64(i + 1),
			PlayerId:         pid,
			TemporaryAddress: fmt.Sprintf("0xp%d", pid),
			Status:           dao.TournamentEntryStatusQueued,
		}).Error)
	}

	tc := newCoordinator(context.Background(), &mockGameCreator{})
	require.NoError(t, tc.lockTournament(tour))

	var reloaded dao.Tournament
	require.NoError(t, db.Get().First(&reloaded, tour.ID).Error)
	require.Equal(t, dao.TournamentStatusInProgress, reloaded.Status)
	require.EqualValues(t, 2, reloaded.BracketSlots)

	matches, err := db.TournamentListMatchesForRound(db.Get(), tour.ID, 1)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.NotNil(t, matches[0].EntryAID)
	require.NotNil(t, matches[0].EntryBID)
}

func TestPickTournamentWinner_Tie_PrefersSmallerJoinSequence(t *testing.T) {
	ea := &dao.TournamentEntry{JoinSequence: 1, PlayerId: 1, TemporaryAddress: "0xa"}
	ea.ID = 10
	eb := &dao.TournamentEntry{JoinSequence: 5, PlayerId: 2, TemporaryAddress: "0xb"}
	eb.ID = 20
	m := &dao.TournamentMatch{EntryA: ea, EntryB: eb}
	g := &dao.Game{
		GameResult: &dao.GameResult{GameResultType: proto.GameResultType_GAME_TIE},
	}
	w, err := pickTournamentWinner(m, g)
	require.NoError(t, err)
	require.EqualValues(t, 10, w)
}

func TestPickTournamentWinner_Normal_WinnerFromResult(t *testing.T) {
	ea := &dao.TournamentEntry{JoinSequence: 9, PlayerId: 100, TemporaryAddress: "0xaa"}
	ea.ID = 10
	eb := &dao.TournamentEntry{JoinSequence: 1, PlayerId: 200, TemporaryAddress: "0xbb"}
	eb.ID = 20
	m := &dao.TournamentMatch{EntryA: ea, EntryB: eb}
	g := &dao.Game{
		GameResult: &dao.GameResult{
			GameResultType:         proto.GameResultType_GAME_NORMAL,
			WinnerPlayerId:         200,
			WinnerTemporaryAddress: "0xbb",
		},
	}
	w, err := pickTournamentWinner(m, g)
	require.NoError(t, err)
	require.EqualValues(t, 20, w)
}

func TestPickTournamentWinner_FallbackSequenceWhenWinnerMismatch(t *testing.T) {
	ea := &dao.TournamentEntry{JoinSequence: 2, PlayerId: 1, TemporaryAddress: "0xa"}
	ea.ID = 10
	eb := &dao.TournamentEntry{JoinSequence: 7, PlayerId: 2, TemporaryAddress: "0xb"}
	eb.ID = 20
	m := &dao.TournamentMatch{EntryA: ea, EntryB: eb}
	g := &dao.Game{
		GameResult: &dao.GameResult{
			GameResultType:         proto.GameResultType_GAME_NORMAL,
			WinnerPlayerId:         999,
			WinnerTemporaryAddress: "0xunknown",
		},
	}
	w, err := pickTournamentWinner(m, g)
	require.NoError(t, err)
	require.EqualValues(t, 10, w)
}

func TestTournamentQueueService_JoinExit_JoinSequence(t *testing.T) {
	setupSQLite(t)
	_ = seedGameArgs(t)
	sched := seedSchedule(t, time.Now().Add(time.Hour))
	tour := openTournament(t, sched.ID, time.Now().Add(time.Minute))
	_ = tour

	seedProfileAndToken(t, 5001, "tn5001", "0xt5001", 100_000)
	svc := NewTournamentQueueService(context.Background(), &mockGameCreator{}, 1000)

	p1 := &proto.PlayerAddress{Id: 5001, TemporaryAddress: "0xt5001"}
	require.NoError(t, svc.HandleJoinTournament(sched.ID, p1))
	require.Error(t, svc.HandleJoinTournament(sched.ID, p1))

	var e1 dao.TournamentEntry
	require.NoError(t, db.Get().Where("tournament_id = ? AND player_id = ?", tour.ID, int64(5001)).First(&e1).Error)
	require.EqualValues(t, 1, e1.JoinSequence)

	require.NoError(t, svc.HandleExitTournament(sched.ID, p1))
	err := db.Get().Where("tournament_id = ? AND player_id = ?", tour.ID, int64(5001)).First(&dao.TournamentEntry{}).Error
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestTournamentQueueService_Join_RejectsDisabledSchedule(t *testing.T) {
	setupSQLite(t)
	sched := seedSchedule(t, time.Now().Add(time.Hour))
	require.NoError(t, db.Get().Model(&dao.TournamentSchedule{}).Where("id = ?", sched.ID).Update("enabled", false).Error)
	openTournament(t, sched.ID, time.Now().Add(time.Minute))

	seedProfileAndToken(t, 5002, "tn5002", "0xt5002", 100_000)
	svc := NewTournamentQueueService(context.Background(), &mockGameCreator{}, 1000)
	err := svc.HandleJoinTournament(sched.ID, &proto.PlayerAddress{Id: 5002, TemporaryAddress: "0xt5002"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "disabled")
}

func TestTournamentQueueService_GameResultSettlementHook_IgnoresNonTournament(t *testing.T) {
	setupSQLite(t)
	svc := NewTournamentQueueService(context.Background(), &mockGameCreator{}, 1000)
	g := &dao.Game{Type: types.GameTypePVP}
	g.ID = 1
	require.NoError(t, svc.GameResultSettlementHook(&types.GameCompletedEvent{GameID: 1, GameInfo: g}))
}

func TestTournamentQueueService_GameResultSettlementHook_NilSafe(t *testing.T) {
	setupSQLite(t)
	svc := NewTournamentQueueService(context.Background(), &mockGameCreator{}, 1000)
	require.NoError(t, svc.GameResultSettlementHook(nil))
	require.NoError(t, svc.GameResultSettlementHook(&types.GameCompletedEvent{}))
}

func TestLockTournament_OverflowKicksExcessPlayers(t *testing.T) {
	setupSQLite(t)
	sched := seedSchedule(t, time.Now().Add(time.Hour))
	tour := openTournament(t, sched.ID, time.Now().Add(-time.Minute))
	// MaxParticipants = 64; enqueue 65
	for i := 0; i < 65; i++ {
		require.NoError(t, db.Get().Create(&dao.TournamentEntry{
			TournamentID:     tour.ID,
			JoinSequence:     int64(i + 1),
			PlayerId:         int64(10000 + i),
			TemporaryAddress: fmt.Sprintf("0xo%d", i),
			Status:           dao.TournamentEntryStatusQueued,
		}).Error)
	}

	tc := newCoordinator(context.Background(), &mockGameCreator{})
	require.NoError(t, tc.lockTournament(tour))

	var inBracket, kicked int64
	require.NoError(t, db.Get().Model(&dao.TournamentEntry{}).
		Where("tournament_id = ? AND status = ?", tour.ID, dao.TournamentEntryStatusInBracket).
		Count(&inBracket).Error)
	require.NoError(t, db.Get().Model(&dao.TournamentEntry{}).
		Where("tournament_id = ? AND status = ?", tour.ID, dao.TournamentEntryStatusKickedOverflow).
		Count(&kicked).Error)
	require.EqualValues(t, 64, inBracket)
	require.EqualValues(t, 1, kicked)
}

func TestOnGameCompleted_FinalRound_MarksWinnerAndCompletes(t *testing.T) {
	setupSQLite(t)
	ga := seedGameArgs(t)
	sched := seedSchedule(t, time.Now().Add(time.Hour))
	tour := &dao.Tournament{
		TournamentScheduleID: sched.ID,
		InstanceIndex:        0,
		ScheduledStartAt:     time.Now().Add(-time.Hour),
		BracketExponent:      dao.TournamentBracketExpMin,
		MaxParticipants:      64,
		BracketSlots:         2,
		Status:               dao.TournamentStatusInProgress,
	}
	require.NoError(t, db.Get().Create(tour).Error)

	e1 := &dao.TournamentEntry{
		TournamentID: tour.ID, JoinSequence: 3, PlayerId: 301, TemporaryAddress: "0xe1",
		Status: dao.TournamentEntryStatusInBracket, SeedPosition: 0,
	}
	e2 := &dao.TournamentEntry{
		TournamentID: tour.ID, JoinSequence: 8, PlayerId: 302, TemporaryAddress: "0xe2",
		Status: dao.TournamentEntryStatusInBracket, SeedPosition: 1,
	}
	require.NoError(t, db.Get().Create(e1).Error)
	require.NoError(t, db.Get().Create(e2).Error)

	g := &dao.Game{
		GameArgsID: ga.ID,
		Type:       types.GameTypeTournament,
		Status:     proto.GameStatus_GAME_END,
	}
	require.NoError(t, db.Get().Omit("Players", "Turns", "GameResult").Create(g).Error)
	require.NoError(t, db.Get().Create(&dao.GamePlayerInfo{GameID: g.ID, PlayerId: 301, TemporaryAddress: "0xe1"}).Error)
	require.NoError(t, db.Get().Create(&dao.GamePlayerInfo{GameID: g.ID, PlayerId: 302, TemporaryAddress: "0xe2"}).Error)
	gr := &dao.GameResult{
		GameID:                 g.ID,
		GameResultType:         proto.GameResultType_GAME_TIE,
		WinnerPlayerId:         0,
		WinnerTemporaryAddress: "",
	}
	require.NoError(t, db.Get().Create(gr).Error)
	g.GameResult = gr

	match := &dao.TournamentMatch{
		TournamentID: tour.ID,
		RoundNumber:  1,
		MatchIndex:   0,
		EntryAID:     &e1.ID,
		EntryBID:     &e2.ID,
		GameID:       &g.ID,
	}
	require.NoError(t, db.Get().Create(match).Error)

	tc := newCoordinator(context.Background(), &mockGameCreator{})
	require.NoError(t, tc.onGameCompleted(g))

	var reloaded dao.Tournament
	require.NoError(t, db.Get().First(&reloaded, tour.ID).Error)
	require.Equal(t, dao.TournamentStatusCompleted, reloaded.Status)

	var w dao.TournamentEntry
	require.NoError(t, db.Get().Where("tournament_id = ? AND status = ?", tour.ID, dao.TournamentEntryStatusWinner).First(&w).Error)
	require.EqualValues(t, e1.ID, w.ID)

	var m2 dao.TournamentMatch
	require.NoError(t, db.Get().First(&m2, match.ID).Error)
	require.NotNil(t, m2.WinnerEntryID)
	require.EqualValues(t, e1.ID, *m2.WinnerEntryID)
}

func TestMockGameCreator_StartMatchFails_NoPanic(t *testing.T) {
	setupSQLite(t)
	sched := seedSchedule(t, time.Now().Add(time.Hour))
	tour := openTournament(t, sched.ID, time.Now().Add(-time.Minute))
	for i, pid := range []int64{9201, 9202} {
		require.NoError(t, db.Get().Create(&dao.TournamentEntry{
			TournamentID:     tour.ID,
			JoinSequence:     int64(i + 1),
			PlayerId:         pid,
			TemporaryAddress: fmt.Sprintf("0xf%d", pid),
			Status:           dao.TournamentEntryStatusQueued,
		}).Error)
	}

	fail := &mockGameCreator{
		onMatch: func([]types.PlayerAddress) (uint, error) {
			return 0, errors.New("chain unavailable")
		},
	}
	tc := newCoordinator(context.Background(), fail)
	require.NoError(t, tc.lockTournament(tour))

	var matches []dao.TournamentMatch
	require.NoError(t, db.Get().Where("tournament_id = ?", tour.ID).Find(&matches).Error)
	require.NotEmpty(t, matches)
	for _, m := range matches {
		require.Nil(t, m.GameID)
	}
}
