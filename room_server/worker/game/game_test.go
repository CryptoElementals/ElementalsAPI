package game

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker"
	tt "github.com/CryptoElementals/common/room_server/worker/testing"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func newTestGameArgsRow(t *testing.T, initialHP int64) *dao.GameArgs {
	t.Helper()
	ga := &dao.GameArgs{
		MaxNormalRounds:                       3,
		MaxExtraRounds:                        0,
		MaxTurnsPerNormalRound:                3,
		MaxTurnsPerExtraRound:                 1,
		InitialHP:                             initialHP,
		BaseStake:                             1000,
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
	dao.MustValidateGameArgs(ga)
	require.NotZero(t, ga.ID)
	return ga
}

var testWorkerManager *worker.WorkerManager

// gmEventHandler bridges WorkerManager events to GameManager (same wiring production would use for GAME_MANAGER_ID).
type gmEventHandler struct {
	gm *GameManager
}

func (h *gmEventHandler) Handle(ctx context.Context, event *types.Event) error {
	switch d := event.Data.(type) {
	case *types.GameMatchedEvent:
		_, err := h.gm.CreateGameAndRun(d.Players, d.GameType, 0)
		return err
	default:
		return nil
	}
}

func registerGameManagerWorker(gm *GameManager) {
	testWorkerManager.SpawnWorker(context.Background(), types.GAME_MANAGER_ID, types.WORKER_TYPE_GAME_MANAGER, &gmEventHandler{gm: gm})
}

// NopPublisher implements Publisher with a no-op Publish. Use in tests when a non-nil Publisher is required.
type NopPublisher struct{}

func (NopPublisher) Publish(_ context.Context, _ *proto.PublishRequest) (*proto.PublishResponse, error) {
	return &proto.PublishResponse{Success: true}, nil
}

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

func prepareCards(t *testing.T) {
	t.Helper()
	cards := []dao.Card{
		{CardID: 1, ElementType: "Metal", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "Kylin", Description: "Kylin clad in armor, representing strength and protection"},
		{CardID: 2, ElementType: "Wood", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "Forest Spirit", Description: "Forest Spirit controlling the cycle of life and death"},
		{CardID: 3, ElementType: "Water", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "Siren", Description: "Siren, half-human half-beast, possessing enchanting charm"},
		{CardID: 4, ElementType: "Fire", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "Phoenix", Description: "Phoenix with flames and rebirth, symbolizing eternal life"},
		{CardID: 5, ElementType: "Earth", Level: "normal", LifeForce: 500, Attack: 1000, Defense: 500, Name: "World Turtle", Description: "World Turtle, steady and powerful with immense strength"},
	}
	require.NoError(t, db.Get().Save(&cards).Error)
}

func setupGameTest(ctx context.Context, expectedRoundNumber int, t *testing.T) {
	playerAddress1 := types.PlayerAddress{
		Id:               1,
		TemporaryAddress: "1",
	}

	playerAddress2 := types.PlayerAddress{
		Id:               2,
		TemporaryAddress: "2",
	}
	mockPlayer1 := tt.NewMockEventHandler(gomock.NewController(t))
	mockPlayer2 := tt.NewMockEventHandler(gomock.NewController(t))
	mockChain := tt.NewMockEventHandler(gomock.NewController(t))
	testWorkerManager.SpawnWorker(context.Background(), playerAddress1.String(), types.WORKER_TYPE_PLAYER, mockPlayer1)
	testWorkerManager.SpawnWorker(context.Background(), playerAddress2.String(), types.WORKER_TYPE_PLAYER, mockPlayer2)
	testWorkerManager.SpawnWorker(context.Background(), types.CHAIN_MANAGER_ID, types.WORKER_TYPE_CHAIN, mockChain)
	gameCreatedEvtMatcher := tt.NewEventTypeMatcher(&types.GameCreatedEvent{})
	mockChain.EXPECT().Handle(gomock.Any(), tt.NewEventTypeMatcher(&types.RequireGameCreationEvent{})).Times(1).DoAndReturn(func(ctx context.Context, event *types.Event) error {
		evt := event.Data.(*types.RequireGameCreationEvent)
		gid := evt.GameID
		wid := fmt.Sprint(gid)
		contractEvt := types.NewEvent(types.CHAIN_MANAGER_ID, &proto.TxGameCreated{})
		testWorkerManager.SendEvent(wid, contractEvt)
		_ = gid
		_ = wid
		return nil
	})
	mockPlayer1.EXPECT().Handle(gomock.Any(), gameCreatedEvtMatcher).Times(1).DoAndReturn(func(ctx context.Context, event *types.Event) error {
		evt := event.Data.(*types.GameCreatedEvent)
		gid := evt.GameID
		wid := fmt.Sprint(gid)
		testWorkerManager.SendEvent(wid, types.NewEvent(playerAddress1.String(), &proto.ConfirmBattleRequest{
			GameID:        gid,
			RoundNumber:   1,
			TurnNumber:    1,
			PlayerAddress: playerAddress1.ToProto(),
		}))
		return nil
	})
	mockPlayer2.EXPECT().Handle(gomock.Any(), gameCreatedEvtMatcher).Times(1).DoAndReturn(func(ctx context.Context, event *types.Event) error {
		evt := event.Data.(*types.GameCreatedEvent)
		gid := evt.GameID
		wid := fmt.Sprint(gid)
		testWorkerManager.SendEvent(wid, types.NewEvent(playerAddress2.String(), &proto.ConfirmBattleRequest{
			GameID:        gid,
			RoundNumber:   1,
			TurnNumber:    1,
			PlayerAddress: playerAddress2.ToProto(),
		}))
		return nil
	})

	waitGameEnd := make(chan struct{}, 2)
	mockPlayer1.EXPECT().Handle(gomock.Any(), tt.NewEventTypeMatcher(&types.GameCompletedEvent{})).Times(1).DoAndReturn(func(ctx context.Context, event *types.Event) error {
		evt := event.Data.(*types.GameCompletedEvent)
		gid := evt.GameID
		require.Equal(t, gid, evt.GameID)
		require.Equal(t, proto.GameStatus_GAME_END, evt.GameInfo.Status)
		waitGameEnd <- struct{}{}
		return nil
	})
	mockPlayer2.EXPECT().Handle(gomock.Any(), tt.NewEventTypeMatcher(&types.GameCompletedEvent{})).Times(1).DoAndReturn(func(ctx context.Context, event *types.Event) error {
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
	require.Eventually(t, func() bool {
		g, err := db.GetActiveGameByPlayer(playerAddress1.Id, playerAddress1.TemporaryAddress)
		return err == nil && g != nil && g.ID > 0
	}, 3*time.Second, 20*time.Millisecond, "persisted game after match")
	gi, err := db.GetActiveGameByPlayer(playerAddress1.Id, playerAddress1.TemporaryAddress)
	require.NoError(t, err)
	require.NotNil(t, gi.GameArgs)
	created := &types.GameCreatedEvent{
		GameID:              gi.ID,
		Players:             []types.PlayerAddress{playerAddress1, playerAddress2},
		ConfirmationTimeout: gi.GameArgs.ConfirmationTimeout,
	}
	for _, p := range []types.PlayerAddress{playerAddress1, playerAddress2} {
		testWorkerManager.SendEvent(p.String(), &types.Event{Sender: types.GAME_MANAGER_ID, Data: created})
	}
	for i := 0; i < 2; i++ {
		select {
		case <-ctx.Done():
			t.Error("game timeout")
		case <-waitGameEnd:
		}
	}
}

func TestGameManagerNewGameAndRecover(t *testing.T) {
	setupMemDb(t)
	ga := newTestGameArgsRow(t, 3000)
	contractClient := tt.NewMockContractClient(gomock.NewController(t))
	gameManager := NewGameManager(context.Background(), testWorkerManager, NopPublisher{}, ga, contractClient, 0, 0)
	registerGameManagerWorker(gameManager)
	require.NoError(t, gameManager.Start())
	playerAddress1 := types.PlayerAddress{
		Id:               1,
		TemporaryAddress: "1",
	}

	playerAddress2 := types.PlayerAddress{
		Id:               2,
		TemporaryAddress: "2",
	}

	testWorkerManager.SendEvent(types.GAME_MANAGER_ID, &types.Event{
		Sender: types.QUEUE_MANAGER_ID,
		Data: &types.GameMatchedEvent{
			Players: []types.PlayerAddress{playerAddress1, playerAddress2},
		},
	})

	require.Eventually(t, func() bool {
		_, err := db.GetActiveGameByPlayer(playerAddress1.Id, playerAddress1.TemporaryAddress)
		return err == nil
	}, 3*time.Second, 20*time.Millisecond, "game row after match")
	// Stateless manager: DB is the source of truth. Ensure the game was created and runtime state can be rebuilt.
	gameInfo, err := db.GetActiveGameByPlayer(playerAddress1.Id, playerAddress1.TemporaryAddress)
	require.NoError(t, err)
	require.NotNil(t, gameInfo)

	currentRound := buildRuntimeState(gameInfo)
	require.NotNil(t, currentRound)
	require.NotNil(t, currentRound.game)
	require.NotEmpty(t, currentRound.game.Turns)
	require.NotNil(t, currentRound.getCurrentTurn())
	require.GreaterOrEqual(t, currentRound.turnNumber, uint32(1))
	require.LessOrEqual(t, currentRound.turnNumber, uint32(3))

	// Phase 3: full-graph save must not change observable shape vs granular writes.
	full, err := db.LoadGameByGameID(gameInfo.ID)
	require.NoError(t, err)
	snap := db.CaptureGamePersistenceSnapshot(full)
	require.NoError(t, db.SaveFullGameGraph(full))
	reloaded, err := db.LoadGameByGameID(gameInfo.ID)
	require.NoError(t, err)
	require.Equal(t, snap, db.CaptureGamePersistenceSnapshot(reloaded))
}

func TestGameStateMachine(t *testing.T) {
	t.Skip("TODO: rework harness — GameCreated is delivered via Publisher, not player workers; DbRoundToRoundResult IsGameOver uses game end state")
	setupMemDb(t)
	prepareCards(t)
	contractClient := tt.NewMockContractClient(gomock.NewController(t))
	compareRound := func(svc *Service, roundNumber int, isLast bool) {
		rr, gr, err := svc.LoadBattleInfoFromDB(1, uint32(roundNumber))
		require.NoError(t, err)
		require.Equal(t, isLast, rr.IsGameOver)
		if isLast {
			require.NotNil(t, gr)
		}
		rrDb, grDb, err := svc.LoadBattleInfoFromDB(1, uint32(roundNumber))
		require.NoError(t, err)

		require.EqualExportedValues(t, rr, rrDb)
		require.EqualExportedValues(t, gr, grDb)
	}
	t.Run("1 rounds", func(t *testing.T) {
		ga := newTestGameArgsRow(t, 1000)
		svc := NewService(context.Background(), testWorkerManager, NopPublisher{}, ga, contractClient, 0, 0)
		registerGameManagerWorker(svc.gameManager)
		require.NoError(t, svc.Start())
		ctx, cancel := context.WithTimeout(context.Background(), 3000*time.Millisecond)
		defer cancel()
		setupGameTest(ctx, 1, t)
		compareRound(svc, 1, true)
	})
	t.Run("2 rounds", func(t *testing.T) {
		ga := newTestGameArgsRow(t, 3000)
		svc := NewService(context.Background(), testWorkerManager, NopPublisher{}, ga, contractClient, 0, 0)
		registerGameManagerWorker(svc.gameManager)
		require.NoError(t, svc.Start())
		ctx, cancel := context.WithTimeout(context.Background(), 3000*time.Millisecond)
		defer cancel()
		setupGameTest(ctx, 2, t)
		compareRound(svc, 1, false)
		compareRound(svc, 2, true)
	})
	t.Run("3 rounds", func(t *testing.T) {
		ga := newTestGameArgsRow(t, 10000)
		svc := NewService(context.Background(), testWorkerManager, NopPublisher{}, ga, contractClient, 0, 0)
		registerGameManagerWorker(svc.gameManager)
		require.NoError(t, svc.Start())
		ctx, cancel := context.WithTimeout(context.Background(), 3000*time.Millisecond)
		defer cancel()
		setupGameTest(ctx, 3, t)
		compareRound(svc, 1, false)
		compareRound(svc, 2, false)
		compareRound(svc, 3, true)
	})
}
