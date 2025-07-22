package player

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CryptoElementals/common/conversion"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	pub "github.com/CryptoElementals/common/rpc/server"
	"github.com/CryptoElementals/common/room_server/worker"
	tt "github.com/CryptoElementals/common/room_server/worker/testing"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var testWorkerManager *worker.WorkerManager
var testPubsubServer *pub.PubSubServer
var pubsubPort = 30011

func TestMain(m *testing.M) {
	testWorkerManager = worker.NewWorkerManager(context.Background())
	testPubsubServer = pub.NewPubSubServer()
	testPubsubServer.Run(pubsubPort)
	m.Run()
}

func TestPlayerJoinExitQueue(t *testing.T) {
	mockGameInfoGetter := tt.NewMockGameInfoGetter(gomock.NewController(t))
	mockQueueInfoGetter := tt.NewMockQueueInfoGetter(gomock.NewController(t))
	mockQueueHandler := tt.NewMockEventHandler(gomock.NewController(t))
	testWorkerManager.SpwanWorker(context.Background(), types.QUEUE_MANAGER_ID, types.WORKER_TYPE_QUEUE, mockQueueHandler)

	testService := NewService(context.Background(), testPubsubServer, testWorkerManager, mockGameInfoGetter, mockQueueInfoGetter)
	player1 := types.PlayerAddress{
		WalletAddress:    "player1",
		TemporaryAddress: "temp1",
	}
	player2 := types.PlayerAddress{
		WalletAddress:    "player2",
		TemporaryAddress: "temp2",
	}
	mockQueueHandler.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().
		DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
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
	require.Equal(t, proto.PlayerStatus_PLAYER_KNOWN, player1Struct.status)
	require.Equal(t, proto.PlayerStatus_PLAYER_KNOWN, player2Struct.status)
	testService.RemovePlayer(player1)
	testService.RemovePlayer(player2)
	require.Nil(t, testService.players[player1])
	require.Nil(t, testService.players[player2])
}

func TestPlayerEventHandler(t *testing.T) {
	mockGameInfoGetter := tt.NewMockGameInfoGetter(gomock.NewController(t))
	mockQueueInfoGetter := tt.NewMockQueueInfoGetter(gomock.NewController(t))

	testService := NewService(context.Background(), testPubsubServer, testWorkerManager, mockGameInfoGetter, mockQueueInfoGetter)
	player1 := types.PlayerAddress{
		WalletAddress:    "player1",
		TemporaryAddress: "temp1",
	}
	player2 := types.PlayerAddress{
		WalletAddress:    "player2",
		TemporaryAddress: "temp2",
	}

	require.NoError(t, testService.AddPlayer(player1))
	player1Struct := testService.players[player1]
	require.NotNil(t, player1Struct)
	pubsubClient, err := client.NewPubSubClient(fmt.Sprintf("localhost:%d", pubsubPort))
	require.NoError(t, err)
	defer pubsubClient.Close()
	player1Chan := make(chan *proto.Event, 100)
	errChan := make(chan error, 1)
	go func() {
		err := <-errChan
		if err == nil {
			return
		}
		t.Errorf("unexpected error: %v", err)
	}()
	require.NoError(t, pubsubClient.Subscribe(player1.String(), player1.String(), player1Chan, errChan))
	time.Sleep(1 * time.Millisecond)
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
		Type: proto.EventType_GAME_CREATED,
		Data: &proto.Event_GameCreated{
			GameCreated: &proto.GameCreated{
				GameId: uint32(gameID),
				Players: []*proto.PlayerAddress{
					{
						WalletAddress:    player1.WalletAddress,
						TemporaryAddress: player1.TemporaryAddress,
					},
					{
						WalletAddress:    player2.WalletAddress,
						TemporaryAddress: player2.TemporaryAddress,
					},
				},
			},
		},
	}, evt)
	require.Equal(t, player1Struct.status, proto.PlayerStatus_PLAYER_IN_GAME)

	testWorkerManager.SendEvent(player1.String(), types.NewEvent(types.GAME_MANAGER_ID, &types.GameReadyEvent{
		GameID:          uint(gameID),
		ContractAddress: "0x123",
	}))
	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_GAME_READY,
		Data: &proto.Event_GameReady{
			GameReady: &proto.GameReady{
				GameId:          uint32(gameID),
				ContractAddress: "0x123",
			},
		},
	}, evt)

	testWorkerManager.SendEvent(player1.String(), types.NewEvent(types.GAME_MANAGER_ID, &types.RoundReadyEvent{
		GameID:      uint(gameID),
		RoundNumber: 0,
	}))
	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_ROUND_READY,
		Data: &proto.Event_RoundReady{
			RoundReady: &proto.RoundReady{
				GameId:   uint32(gameID),
				RoundNum: 0,
			},
		},
	}, evt)

	testWorkerManager.SendEvent(player1.String(), types.NewEvent(types.GAME_MANAGER_ID, &types.CommitmentsOnChainEvent{
		GameID:      uint(gameID),
		RoundNumber: 0,
	}))
	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_COMMITMENTS_ON_CHAIN,
		Data: &proto.Event_CommitmentsOnChain{
			CommitmentsOnChain: &proto.CommitmentsOnChain{
				GameId:   uint32(gameID),
				RoundNum: 0,
			},
		},
	}, evt)

	testWorkerManager.SendEvent(player1.String(), types.NewEvent(types.GAME_MANAGER_ID, &types.RoundCompletedEvent{
		GameID: uint(gameID),
		RoundInfo: &dao.Round{
			GameID:      uint(gameID),
			RoundNumber: 0,
			Status:      proto.RoundStatus_ROUND_COMPLETED,
		},
	}))

	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_ROUND_COMPLETED,
		Data: &proto.Event_RoundCompleted{
			RoundCompleted: &proto.RoundCompleted{
				GameId: uint32(gameID),
				RoundInfo: conversion.DbGameRoundToProtoGameRound(&dao.Round{
					GameID:      uint(gameID),
					RoundNumber: 0,
					Status:      proto.RoundStatus_ROUND_COMPLETED,
				}),
			},
		},
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

	evt = <-player1Chan
	require.EqualExportedValues(t, &proto.Event{
		Type: proto.EventType_GAME_COMPLETED,
		Data: &proto.Event_GameInfo{
			GameInfo: &proto.GameInfo{
				GameId: uint32(gameID),
				Status: proto.GameStatus_GAME_END,
			},
		},
	}, evt)

}
