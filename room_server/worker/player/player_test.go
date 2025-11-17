package player

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	tt "github.com/CryptoElementals/common/room_server/worker/testing"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	pub "github.com/CryptoElementals/common/rpc/server"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

var testWorkerManager *worker.WorkerManager
var testPubsubServer *pub.PubSub
var pubsubPort = 30011

func TestMain(m *testing.M) {
	testWorkerManager = worker.NewWorkerManager(context.Background())
	testPubsubServer = pub.NewPubSub()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", pubsubPort))
	if err != nil {
		panic(err)
	}
	svr := grpc.NewServer()
	proto.RegisterPubSubServiceServer(svr, testPubsubServer)
	go func() {
		if err := svr.Serve(lis); err != nil {
			log.Fatalf("server start failed: %v", err)
		}
	}()

	m.Run()
}

func TestPlayerJoinExitQueue(t *testing.T) {
	mockGameInfoGetter := tt.NewMockGameInfoGetter(gomock.NewController(t))
	mockQueueInfoGetter := tt.NewMockQueuer(gomock.NewController(t))
	mockQueueHandler := tt.NewMockEventHandler(gomock.NewController(t))
	testWorkerManager.SpwanWorker(context.Background(), types.QUEUE_MANAGER_ID, types.WORKER_TYPE_QUEUE, mockQueueHandler)

	testService := NewService(context.Background(), testPubsubServer, testWorkerManager, mockGameInfoGetter, mockQueueInfoGetter)
	testPubsubServer.SetPlayerManager(testService)
	player1 := types.PlayerAddress{
		Id:               1,
		TemporaryAddress: "temp1",
	}
	player2 := types.PlayerAddress{
		Id:               2,
		TemporaryAddress: "temp2",
	}
	mockQueueHandler.EXPECT().Handle(gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(ctx context.Context, event *types.Event) error {
			switch evt := event.Data.(type) {
			case *types.JoinQueueEvent:
				if event.Sender == player1.String() {
					require.Equal(t, player1, evt.PlayerAddress)
				} else if event.Sender == player2.String() {
					require.Equal(t, player2, evt.PlayerAddress)
				} else {
					t.Error("unexpected event sender")
				}
			case *types.ExitQueueEvent:
				if event.Sender == player1.String() {
					require.Equal(t, player1, evt.PlayerAddress)
				} else if event.Sender == player2.String() {
					require.Equal(t, player2, evt.PlayerAddress)
				} else {
					t.Error("unexpected event sender")
				}
			default:
				t.Error("unexpected event type")
			}
			return nil
		})
	require.NoError(t, testService.AddPlayer(player1))
	require.NoError(t, testService.AddPlayer(player2))
	player1Struct := testService.players[player1]
	player2Struct := testService.players[player2]
	require.NotNil(t, player1Struct)
	require.NotNil(t, player2Struct)
	require.Equal(t, player1, player1Struct.address)
	require.Equal(t, player2, player2Struct.address)
	require.NoError(t, testService.JoinQueue(player1))
	require.NoError(t, testService.JoinQueue(player2))
	time.Sleep(1 * time.Millisecond)
	require.Equal(t, proto.PlayerStatus_PLAYER_IN_QUEUE, player1Struct.status)
	require.Equal(t, proto.PlayerStatus_PLAYER_IN_QUEUE, player2Struct.status)

	require.NoError(t, testService.ExitQueue(player1))
	require.NoError(t, testService.ExitQueue(player2))
	time.Sleep(1 * time.Millisecond)
	require.Equal(t, proto.PlayerStatus_PLAYER_UNKNOWN, player1Struct.status)
	require.Equal(t, proto.PlayerStatus_PLAYER_UNKNOWN, player2Struct.status)
	testService.RemovePlayer(player1)
	testService.RemovePlayer(player2)
	require.Nil(t, testService.players[player1])
	require.Nil(t, testService.players[player2])
}

func TestPlayerEventHandler(t *testing.T) {
	mockGameInfoGetter := tt.NewMockGameInfoGetter(gomock.NewController(t))
	mockQueueInfoGetter := tt.NewMockQueuer(gomock.NewController(t))

	testService := NewService(context.Background(), testPubsubServer, testWorkerManager, mockGameInfoGetter, mockQueueInfoGetter)
	testPubsubServer.SetPlayerManager(testService)
	player1 := types.PlayerAddress{
		Id:               1,
		TemporaryAddress: "temp1",
	}
	player2 := types.PlayerAddress{
		Id:               2,
		TemporaryAddress: "temp2",
	}
	conn, err := client.DailGrpcEndpoint(fmt.Sprintf("localhost:%d", pubsubPort))
	require.NoError(t, err)
	defer conn.Close()
	pubsubClient := client.NewPubSubClient(conn)
	player1Chan := make(chan *proto.Event, 100)
	player2Chan := make(chan *proto.Event, 100)
	errChan := make(chan error, 1)
	errChan2 := make(chan error, 1)
	go func() {
		select {
		case err := <-errChan:
			if err == nil {
				return
			}
			t.Errorf("unexpected error: %v", err)
		case err := <-errChan2:
			if err == nil {
				return
			}
			t.Errorf("unexpected error: %v", err)
		}
	}()
	require.NoError(t, pubsubClient.Subscribe(player1.String(), player1.String(), player1Chan, errChan))
	require.NoError(t, pubsubClient.Subscribe(player2.String(), player2.String(), player2Chan, errChan2))
	time.Sleep(1 * time.Millisecond)
	player1Struct := testService.players[player1]
	require.NotNil(t, player1Struct)
	player2Struct := testService.players[player2]
	require.NotNil(t, player2Struct)

	require.NoError(t, testService.JoinQueue(player1))
	gameID := 1
	testWorkerManager.SendEvent(player1.String(), types.NewEvent(types.GAME_MANAGER_ID, &types.GameCreatedEvent{
		GameID: uint(gameID),
		Players: []types.PlayerAddress{
			player1,
			player2,
		},
	}))
	evt := <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_TYPE_MATCHED,
	}, evt)
	require.Equal(t, player1Struct.status, proto.PlayerStatus_PLAYER_IN_GAME)

	testWorkerManager.SendEvent(player1.String(), types.NewEvent(types.GAME_MANAGER_ID, &types.GameReadyEvent{
		GameID: uint(gameID),
	}))
	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_TYPE_GAME_CREATED,
	}, evt)

	// send partial ready
	testWorkerManager.SendEvent(player1.String(), types.NewEvent(types.GAME_MANAGER_ID, &types.RoundPartialReadyEvent{
		GameID:       uint(gameID),
		RoundNumber:  0,
		ReadyAddress: player1,
	}))
	select {
	case evt = <-player1Chan:
		t.Fatalf("unexpected event: %v", evt)
	default:
	}

	testWorkerManager.SendEvent(player1.String(), types.NewEvent(types.GAME_MANAGER_ID, &types.RoundPartialReadyEvent{
		GameID:       uint(gameID),
		RoundNumber:  0,
		ReadyAddress: player2,
	}))
	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_TYPE_PART_CONFIRMED,
	}, evt)

	testWorkerManager.SendEvent(player1.String(), types.NewEvent(types.GAME_MANAGER_ID, &types.RoundReadyEvent{
		GameID:      uint(gameID),
		RoundNumber: 0,
	}))
	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_TYPE_ROUND_READY,
	}, evt)

	testWorkerManager.SendEvent(player1.String(), types.NewEvent(types.GAME_MANAGER_ID, &types.CommitmentsOnChainEvent{
		GameID:      uint(gameID),
		RoundNumber: 0,
	}))
	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_TYPE_COMMITMENTS_ON_CHAIN,
	}, evt)

	testWorkerManager.SendEvent(player1.String(), types.NewEvent(types.GAME_MANAGER_ID, &types.RoundCompletedEvent{
		GameID: uint(gameID),
		RoundInfo: &dao.Round{
			GameID:      uint(gameID),
			RoundNumber: 0,
			Status:      proto.RoundStatus_ROUND_COMPLETED,
		},
	}))

	// will receive two events in a row
	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_TYPE_CARDS_ON_CHAIN,
	}, evt)

	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_TYPE_ROUND_COMPLETE,
	}, evt)

	// send game end
	testWorkerManager.SendEvent(player1.String(), types.NewEvent(types.GAME_MANAGER_ID, &types.GameCompletedEvent{
		GameID: uint(gameID),
		GameInfo: &dao.Game{
			BaseModel: dao.BaseModel{
				ID: uint(gameID),
			},
			Status: proto.GameStatus_GAME_END,
		},
	}))

	// will receive three events in a row
	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_TYPE_CARDS_ON_CHAIN,
	}, evt)

	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_TYPE_ROUND_COMPLETE,
	}, evt)
	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_TYPE_GAME_COMPLETE,
	}, evt)

}
