package game

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/room_server/worker"
	tt "github.com/CryptoElementals/common/room_server/worker/testing"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

var testWorkerManager *worker.WorkerManager

func TestMain(m *testing.M) {
	time.Local = time.UTC
	testWorkerManager = worker.NewWorkerManager(context.Background())
	// 运行测试
	m.Run()
}

func setupMemDb(t *testing.T) {
	err := db.Init(&db.Config{Development: true})
	require.NoError(t, err)
	err = db.MigrateMemDb()
	require.NoError(t, err)
}

func TestGameManagerNewGameAndRecover(t *testing.T) {
	setupMemDb(t)
	gameManager := NewGameManager(context.Background(), testWorkerManager)

	playerAddress1 := types.PlayerAddress{
		WalletAddress:    "1",
		TemporaryAddress: "1",
	}

	playerAddress2 := types.PlayerAddress{
		WalletAddress:    "2",
		TemporaryAddress: "2",
	}
	// no games now
	require.Len(t, gameManager.gamesMap, 0)
	testWorkerManager.SendEvent(types.GAME_MANAGER_ID, &types.Event{
		Sender: types.QUEUE_MANAGER_ID,
		Data: &types.GameMatchedEvent{
			Players: []types.PlayerAddress{playerAddress1, playerAddress2},
		},
	})
	time.Sleep(100 * time.Millisecond)
	// one game now
	require.Len(t, gameManager.gamesMap, 1)
	// tow players
	require.Len(t, gameManager.playerToGameMap, 2)
	var createdGame *Game
	// close game worker and clear game map
	for _, g := range gameManager.gamesMap {
		createdGame = g
		testWorkerManager.CloseWorker(g.workerID())
	}
	clear(gameManager.gamesMap)
	clear(gameManager.playerToGameMap)

	// recover
	gameManager.recoverGames()
	require.Len(t, gameManager.gamesMap, 1)
	var recoveredGame *Game
	for _, g := range gameManager.gamesMap {
		recoveredGame = g
	}
	require.EqualValues(t, createdGame, recoveredGame)
}

func TestGameStateMachine(t *testing.T) {
	setupMemDb(t)
	gameManager := NewGameManager(context.Background(), testWorkerManager)
	playerAddress1 := types.PlayerAddress{
		WalletAddress:    "1",
		TemporaryAddress: "1",
	}

	playerAddress2 := types.PlayerAddress{
		WalletAddress:    "2",
		TemporaryAddress: "2",
	}

	mockPlayer1 := tt.NewMockEventHandler(gomock.NewController(t))
	mockPlayer2 := tt.NewMockEventHandler(gomock.NewController(t))
	mockChain := tt.NewMockEventHandler(gomock.NewController(t))
	testWorkerManager.SpwanWorker(context.Background(), playerAddress1.String(), types.WORKER_TYPE_PLAYER, mockPlayer1)
	testWorkerManager.SpwanWorker(context.Background(), playerAddress2.String(), types.WORKER_TYPE_PLAYER, mockPlayer2)
	testWorkerManager.SpwanWorker(context.Background(), types.CHAIN_MANAGER_ID, types.WORKER_TYPE_CHAIN, mockChain)
	gameCreatedEvtMatcher := tt.NewEventTypeMatcher(&types.GameCreatedEvent{})
	mockChain.EXPECT().Handle(gomock.Any(), gomock.Any(), tt.NewEventTypeMatcher(&types.RequireContractCreationEvent{})).Times(1).DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		evt := event.Data.(*types.RequireContractCreationEvent)
		gid := evt.GameID
		wid := fmt.Sprint(gid)
		contractEvt := types.NewEvent(types.CHAIN_MANAGER_ID, &types.RoomContractCreated{
			GameID:              gid,
			RoomContractAddress: "0x123",
		})
		testWorkerManager.SendEvent(wid, contractEvt)
		return nil
	})
	mockChain.EXPECT().Handle(gomock.Any(), gomock.Any(), tt.NewEventTypeMatcher(&types.RequireSetupNewRoundEvent{})).Times(2).DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		evt := event.Data.(*types.RequireSetupNewRoundEvent)
		gid := evt.GameID
		wid := fmt.Sprint(gid)
		newRoundEvt := event.Data.(*types.RequireSetupNewRoundEvent)
		setupEvt := types.NewEvent(types.CHAIN_MANAGER_ID, &types.NewRoundSetupComplete{
			GameID:      gid,
			RoundNumber: newRoundEvt.RoundNumber,
		})
		testWorkerManager.SendEvent(wid, setupEvt)
		return nil
	})
	mockPlayer1.EXPECT().Handle(gomock.Any(), gomock.Any(), gameCreatedEvtMatcher).Times(1).DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		evt := event.Data.(*types.GameCreatedEvent)
		gid := evt.GameID
		wid := fmt.Sprint(gid)
		testWorkerManager.SendEvent(wid, types.NewEvent(playerAddress1.String(), &types.PlayerReadyEvent{
			GameId:        gid,
			RoundNumber:   1,
			PlayerAddress: playerAddress1,
		}))
		return nil
	})
	mockPlayer2.EXPECT().Handle(gomock.Any(), gomock.Any(), gameCreatedEvtMatcher).Times(1).DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		evt := event.Data.(*types.GameCreatedEvent)
		gid := evt.GameID
		wid := fmt.Sprint(gid)
		testWorkerManager.SendEvent(wid, types.NewEvent(playerAddress2.String(), &types.PlayerReadyEvent{
			GameId:        gid,
			RoundNumber:   1,
			PlayerAddress: playerAddress2,
		}))
		return nil
	})

	roundCompleteEvtMatcher := tt.NewEventTypeMatcher(&types.RoundCompletedEvent{})
	mockPlayer1.EXPECT().Handle(gomock.Any(), gomock.Any(), roundCompleteEvtMatcher).Times(2).DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		evt := event.Data.(*types.RoundCompletedEvent)
		gid := evt.GameID
		wid := fmt.Sprint(gid)
		testWorkerManager.SendEvent(wid, types.NewEvent(playerAddress1.String(), &types.PlayerReadyEvent{
			GameId:        gid,
			RoundNumber:   evt.RoundInfo.RoundNumber + 1,
			PlayerAddress: playerAddress1,
		}))
		return nil
	})
	mockPlayer2.EXPECT().Handle(gomock.Any(), gomock.Any(), roundCompleteEvtMatcher).Times(2).DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		evt := event.Data.(*types.RoundCompletedEvent)
		gid := evt.GameID
		wid := fmt.Sprint(gid)
		testWorkerManager.SendEvent(wid, types.NewEvent(playerAddress2.String(), &types.PlayerReadyEvent{
			GameId:        gid,
			RoundNumber:   evt.RoundInfo.RoundNumber + 1,
			PlayerAddress: playerAddress2,
		}))
		return nil
	})

	gameReadyEvtMatcher := tt.NewEventTypeMatcher(&types.GameReadyEvent{})
	roundReadyEvtMatcher := tt.NewEventTypeMatcher(&types.RoundReadyEvent{})
	mockPlayer1.EXPECT().Handle(gomock.Any(), gomock.Any(), gameReadyEvtMatcher).Times(1).Return(nil)
	mockPlayer2.EXPECT().Handle(gomock.Any(), gomock.Any(), gameReadyEvtMatcher).Times(1).Return(nil)
	mockPlayer1.EXPECT().Handle(gomock.Any(), gomock.Any(), roundReadyEvtMatcher).Times(3).DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		evt := event.Data.(*types.RoundReadyEvent)
		gid := evt.GameID
		wid := fmt.Sprint(gid)
		testWorkerManager.SendEvent(wid, types.NewEvent(types.CHAIN_MANAGER_ID, &types.PlayerCommitmentOnChain{
			GameID:      evt.GameID,
			Address:     playerAddress1,
			RoundNumber: evt.RoundNumber,
			Commitment:  []byte("commitment1"),
		}))
		return nil
	})
	mockPlayer2.EXPECT().Handle(gomock.Any(), gomock.Any(), roundReadyEvtMatcher).Times(3).DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		evt := event.Data.(*types.RoundReadyEvent)
		gid := evt.GameID
		wid := fmt.Sprint(gid)
		testWorkerManager.SendEvent(wid, types.NewEvent(types.CHAIN_MANAGER_ID, &types.PlayerCommitmentOnChain{
			GameID:      evt.GameID,
			Address:     playerAddress2,
			RoundNumber: evt.RoundNumber,
			Commitment:  []byte("commitment2"),
		}))
		return nil
	})
	mockPlayer1.EXPECT().Handle(gomock.Any(), gomock.Any(), tt.NewEventTypeMatcher(&types.CommitmentsOnChainEvent{})).Times(3).DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		evt := event.Data.(*types.CommitmentsOnChainEvent)
		gid := evt.GameID
		wid := fmt.Sprint(gid)
		testWorkerManager.SendEvent(wid, types.NewEvent(types.CHAIN_MANAGER_ID, &types.PlayerCardsOnChain{
			GameID:      evt.GameID,
			Address:     playerAddress1,
			RoundNumber: evt.RoundNumber,
			Cards:       []uint{1, 2, 3},
			Salt:        []byte("salt1"),
		}))
		return nil
	})
	mockPlayer2.EXPECT().Handle(gomock.Any(), gomock.Any(), tt.NewEventTypeMatcher(&types.CommitmentsOnChainEvent{})).Times(3).DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		evt := event.Data.(*types.CommitmentsOnChainEvent)
		gid := evt.GameID
		wid := fmt.Sprint(gid)
		testWorkerManager.SendEvent(wid, types.NewEvent(types.CHAIN_MANAGER_ID, &types.PlayerCardsOnChain{
			GameID:      evt.GameID,
			Address:     playerAddress2,
			RoundNumber: evt.RoundNumber,
			Cards:       []uint{4, 5, 6},
			Salt:        []byte("salt2"),
		}))
		return nil
	})

	waitGameEnd := make(chan struct{}, 2)
	mockPlayer1.EXPECT().Handle(gomock.Any(), gomock.Any(), tt.NewEventTypeMatcher(&types.GameCompletedEvent{})).Times(1).DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		evt := event.Data.(*types.GameCompletedEvent)
		gid := evt.GameID
		require.Equal(t, gid, evt.GameID)
		require.Equal(t, proto.GameStatus_GAME_END, evt.GameInfo.Status)
		waitGameEnd <- struct{}{}
		return nil
	})
	mockPlayer2.EXPECT().Handle(gomock.Any(), gomock.Any(), tt.NewEventTypeMatcher(&types.GameCompletedEvent{})).Times(1).DoAndReturn(func(ctx context.Context, sender worker.EventSender, event *types.Event) error {
		evt := event.Data.(*types.GameCompletedEvent)
		gid := evt.GameID
		require.Equal(t, gid, evt.GameID)
		require.Equal(t, proto.GameStatus_GAME_END, evt.GameInfo.Status)
		waitGameEnd <- struct{}{}
		return nil
	})
	testWorkerManager.SendEvent(types.GAME_MANAGER_ID, &types.Event{
		Sender: types.QUEUE_MANAGER_ID,
		Data: &types.GameMatchedEvent{
			Players: []types.PlayerAddress{playerAddress1, playerAddress2},
		},
	})
	time.Sleep(10 * time.Millisecond)
	require.Len(t, gameManager.gamesMap, 1)
	for i := 0; i < 2; i++ {
		<-waitGameEnd
	}

}
