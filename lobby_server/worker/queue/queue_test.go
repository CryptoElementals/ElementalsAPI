package queue

import (
	"context"
	"testing"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/redis"
	tt "github.com/CryptoElementals/common/room_server/worker/testing"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/snowflake"
	"github.com/alicebob/miniredis/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var globalTestQueueService *Service

var testMiniRedis *miniredis.Miniredis

// noopEventPublisher satisfies EventPublisher for tests (non-nil publisher required).
type noopEventPublisher struct{}

func (noopEventPublisher) Publish(_ context.Context, _ *proto.Event) (*proto.PublishResponse, error) {
	return &proto.PublishResponse{Success: true}, nil
}

func (noopEventPublisher) Topic() string { return "test-topic" }

func TestMain(m *testing.M) {
	if err := snowflake.Init(1); err != nil {
		panic(err)
	}
	if err := db.Init(&db.Config{Development: true}); err != nil {
		panic(err)
	}
	if err := db.MigrateMemDb(); err != nil {
		panic(err)
	}
	var err error
	testMiniRedis, err = miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer testMiniRedis.Close()
	if err := redis.Init(&redis.Config{Address: testMiniRedis.Addr(), Size: 4}); err != nil {
		panic(err)
	}
	for _, p := range []dao.UserProfile{
		{PlayerID: 1, Name: "queue_test_p1", Address: "addr1"},
		{PlayerID: 2, Name: "queue_test_p2", Address: "addr2"},
	} {
		if err := db.Get().Save(&p).Error; err != nil {
			panic(err)
		}
	}
	m.Run()
}

func TestJoinExitQueue(t *testing.T) {
	testMiniRedis.FlushAll()
	gameCreator := tt.NewMockGameCreator(gomock.NewController(t))
	var err error
	globalTestQueueService, err = NewService(context.Background(), noopEventPublisher{}, nil, gameCreator, 0, 60, 0, 0, 0, 0, "")
	require.NoError(t, err)
	require.NoError(t, globalTestQueueService.Start())
	// send join queue event
	player1 := types.PlayerAddress{
		Id:               1,
		TemporaryAddress: "temporary1",
	}
	err = globalTestQueueService.HandleJoinQueueEvent(player1.ToProto())
	require.NoError(t, err)
	require.True(t, globalTestQueueService.IsPlayerInQueue(player1))

	// send exit queue event
	err = globalTestQueueService.HandleExitQueueEvent(player1.ToProto())
	require.NoError(t, err)
	require.False(t, globalTestQueueService.IsPlayerInQueue(player1))
}

func TestGameMatched(t *testing.T) {
	testMiniRedis.FlushAll()
	ctrl := gomock.NewController(t)
	gameCreator := tt.NewMockGameCreator(ctrl)
	gameCreator.EXPECT().CreateGameAndRun(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1), nil).Times(1)
	var err error
	globalTestQueueService, err = NewService(context.Background(), noopEventPublisher{}, nil, gameCreator, 0, 60, 0, 0, 0, 0, "")
	require.NoError(t, err)
	require.NoError(t, globalTestQueueService.Start())
	// send join queue event
	player1 := types.PlayerAddress{
		Id:               1,
		TemporaryAddress: "temporary1",
	}
	player1DuplicatedWallet := types.PlayerAddress{
		Id:               1, // Same ID as player1
		TemporaryAddress: "temporary3",
	}
	player2 := types.PlayerAddress{
		Id:               2,
		TemporaryAddress: "temporary2",
	}

	err = globalTestQueueService.HandleJoinQueueEvent(player1.ToProto())
	require.NoError(t, err)
	require.True(t, globalTestQueueService.IsPlayerInQueue(player1))

	err = globalTestQueueService.HandleJoinQueueEvent(player1DuplicatedWallet.ToProto())
	require.NoError(t, err)
	require.True(t, globalTestQueueService.IsPlayerInQueue(player1))
	require.True(t, globalTestQueueService.IsPlayerInQueue(player1DuplicatedWallet))

	// Third join triggers a PVP match and inserts game_match (no game row until both confirm).
	err = globalTestQueueService.HandleJoinQueueEvent(player2.ToProto())
	require.NoError(t, err)

	var gm dao.GameMatch
	require.NoError(t, db.Get().Order("id desc").First(&gm).Error)
	require.Equal(t, dao.GameMatchStatusPending, gm.Status)

	pa := types.NewPlayerAddress(gm.Player1ID, gm.Player1TempAddress)
	pb := types.NewPlayerAddress(gm.Player2ID, gm.Player2TempAddress)

	reqA := &proto.ConfirmMatchRequest{
		PlayerAddress: pa.ToProto(),
		MatchId:       gm.ID,
	}
	require.NoError(t, globalTestQueueService.HandleConfirmMatch(reqA))

	reqB := &proto.ConfirmMatchRequest{
		PlayerAddress: pb.ToProto(),
		MatchId:       gm.ID,
	}
	require.NoError(t, globalTestQueueService.HandleConfirmMatch(reqB))

	require.NoError(t, db.Get().First(&gm, "id = ?", gm.ID).Error)
	require.Equal(t, dao.GameMatchStatusGameCreated, gm.Status)
	require.NotNil(t, gm.GameID)
	require.Equal(t, int64(1), *gm.GameID)
}
