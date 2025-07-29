package queue

import (
	"context"
	"testing"
	"time"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/room_server/worker"
	tt "github.com/CryptoElementals/common/room_server/worker/testing"
	"github.com/CryptoElementals/common/room_server/worker/types"
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

func TestMain(m *testing.M) {
	globalTestWorkerManager = worker.NewWorkerManager(context.Background())

	m.Run()
}

func TestJoinExitQueue(t *testing.T) {
	gameCreator := tt.NewMockGameCreator(gomock.NewController(t))
	globalTestQueueService = NewService(context.Background(), globalTestWorkerManager, cache.NewMemCache(), gameCreator, 0)
	require.NoError(t, globalTestQueueService.Start())
	// send join queue event
	player1 := types.PlayerAddress{
		WalletAddress:    "wallet1",
		TemporaryAddress: "temporary1",
	}
	evt := types.NewEvent(player1.String(), &types.JoinQueueEvent{
		PlayerAddress: player1,
	})
	globalTestWorkerManager.SendEvent(types.QUEUE_MANAGER_ID, evt)
	time.Sleep(1 * time.Millisecond)
	require.True(t, globalTestQueueService.IsPlayerInQueue(player1))

	// send exit queue event
	evt = types.NewEvent(player1.String(), &types.ExitQueueEvent{
		PlayerAddress: player1,
	})
	globalTestWorkerManager.SendEvent(types.QUEUE_MANAGER_ID, evt)
	time.Sleep(1 * time.Millisecond)
	require.False(t, globalTestQueueService.IsPlayerInQueue(player1))
}

func TestGameMatched(t *testing.T) {
	gameCreator := tt.NewMockGameCreator(gomock.NewController(t))
	globalTestQueueService = NewService(context.Background(), globalTestWorkerManager, cache.NewMemCache(), gameCreator, 0)
	require.NoError(t, globalTestQueueService.Start())
	// send join queue event
	player1 := types.PlayerAddress{
		WalletAddress:    "wallet1",
		TemporaryAddress: "temporary1",
	}
	player1DuplicatedWallet := types.PlayerAddress{
		WalletAddress:    "wallet1",
		TemporaryAddress: "temporary3",
	}
	player2 := types.PlayerAddress{
		WalletAddress:    "wallet2",
		TemporaryAddress: "temporary2",
	}

	evt := types.NewEvent(player1.String(), &types.JoinQueueEvent{
		PlayerAddress: player1,
	})
	globalTestWorkerManager.SendEvent(types.QUEUE_MANAGER_ID, evt)
	time.Sleep(1 * time.Millisecond)
	require.True(t, globalTestQueueService.IsPlayerInQueue(player1))

	evt = types.NewEvent(player1DuplicatedWallet.String(), &types.JoinQueueEvent{
		PlayerAddress: player1DuplicatedWallet,
	})
	globalTestWorkerManager.SendEvent(types.QUEUE_MANAGER_ID, evt)
	time.Sleep(1 * time.Millisecond)
	require.True(t, globalTestQueueService.IsPlayerInQueue(player1))
	require.True(t, globalTestQueueService.IsPlayerInQueue(player1DuplicatedWallet))

	// send join queue event
	evt = types.NewEvent(player2.String(), &types.JoinQueueEvent{
		PlayerAddress: player2,
	})
	globalTestWorkerManager.SendEvent(types.QUEUE_MANAGER_ID, evt)
	time.Sleep(1 * time.Millisecond)
}
