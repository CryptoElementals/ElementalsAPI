package queue

import (
	"context"
	"testing"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	tt "github.com/CryptoElementals/common/room_server/worker/testing"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var globalTestWorkerManager *worker.WorkerManager

type testEventHandler struct {
	evtChan chan *types.Event
	tt      *testing.T
}

func (h *testEventHandler) Handle(ctx context.Context, event *types.Event) error {
	h.evtChan <- event
	h.tt.Log(*event)
	return nil
}

var globalTestQueueService *Service

// noopEventPublisher satisfies EventPublisher for tests (non-nil publisher required).
type noopEventPublisher struct{}

func (noopEventPublisher) Publish(ctx context.Context, req *proto.PublishRequest) (*proto.PublishResponse, error) {
	return &proto.PublishResponse{Success: true}, nil
}

func TestMain(m *testing.M) {
	if err := db.Init(&db.Config{Development: true}); err != nil {
		panic(err)
	}
	if err := db.MigrateMemDb(); err != nil {
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
	globalTestWorkerManager = worker.NewWorkerManager(context.Background())

	m.Run()
}

func TestJoinExitQueue(t *testing.T) {
	gameCreator := tt.NewMockGameCreator(gomock.NewController(t))
	globalTestQueueService = NewService(context.Background(), globalTestWorkerManager, noopEventPublisher{}, cache.NewMemCache(), gameCreator, 0, 0, 0, 0, "")
	require.NoError(t, globalTestQueueService.Start())
	// send join queue event
	player1 := types.PlayerAddress{
		Id:               1,
		TemporaryAddress: "temporary1",
	}
	err := globalTestQueueService.HandleJoinQueueEvent(player1.ToProto())
	require.NoError(t, err)
	require.True(t, globalTestQueueService.IsPlayerInQueue(player1))

	// send exit queue event
	err = globalTestQueueService.HandleExitQueueEvent(player1.ToProto())
	require.NoError(t, err)
	require.False(t, globalTestQueueService.IsPlayerInQueue(player1))
}

func TestGameMatched(t *testing.T) {
	gameCreator := tt.NewMockGameCreator(gomock.NewController(t))
	gameCreator.EXPECT().HandleGameMatchedEvent(gomock.Any()).Return(uint(1), nil).Times(1)
	globalTestQueueService = NewService(context.Background(), globalTestWorkerManager, noopEventPublisher{}, cache.NewMemCache(), gameCreator, 0, 0, 0, 0, "")
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

	err := globalTestQueueService.HandleJoinQueueEvent(player1.ToProto())
	require.NoError(t, err)
	require.True(t, globalTestQueueService.IsPlayerInQueue(player1))

	err = globalTestQueueService.HandleJoinQueueEvent(player1DuplicatedWallet.ToProto())
	require.NoError(t, err)
	require.True(t, globalTestQueueService.IsPlayerInQueue(player1))
	require.True(t, globalTestQueueService.IsPlayerInQueue(player1DuplicatedWallet))

	// send join queue event
	err = globalTestQueueService.HandleJoinQueueEvent(player2.ToProto())
	require.NoError(t, err)
}
