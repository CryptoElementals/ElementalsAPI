package tournament

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/CryptoElementals/common/bot_manager"
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

const botRedisNamespace = "lobby:v1"

var (
	tournamentMiniRedis *miniredis.Miniredis
	tournamentRedisOnce sync.Once
)

func setupBotRedis(t *testing.T) *bot_manager.RedisStore {
	t.Helper()
	tournamentRedisOnce.Do(func() {
		var err error
		tournamentMiniRedis, err = miniredis.Run()
		require.NoError(t, err)
		require.NoError(t, redis.Init(&redis.Config{Address: tournamentMiniRedis.Addr(), Size: 4}))
	})
	tournamentMiniRedis.FlushAll()
	store, err := bot_manager.NewRedisStore("")
	require.NoError(t, err)
	return store
}

func reserveBot(t *testing.T, store *bot_manager.RedisStore, addr types.PlayerAddress) {
	t.Helper()
	now := time.Now().UnixMilli()
	require.NoError(t, store.UpsertAliveBots(now, addr))
	popped, err := store.PopFreshIdleBotForMatch(now, 60_000)
	require.NoError(t, err)
	require.NotNil(t, popped)
	require.Equal(t, addr.String(), popped.String())
}

func botInRedisSet(t *testing.T, setSuffix string, addr types.PlayerAddress) bool {
	t.Helper()
	ok, err := redis.SIsMember(botRedisNamespace+setSuffix, addr.String())
	require.NoError(t, err)
	return ok
}

type failTournamentGameCreator struct{}

func (f *failTournamentGameCreator) CreateTournamentGameAndRun(_ []types.PlayerAddress, _ int64, _ int64, _ int64) (int64, error) {
	return 0, fmt.Errorf("create tournament game failed")
}

func TestOnGameCompleted_AbortedGame_ReleasesBots(t *testing.T) {
	setupSQLite(t)
	botStore := setupBotRedis(t)

	bot := types.PlayerAddress{Id: 8001, TemporaryAddress: "0xbot8001"}
	human := types.PlayerAddress{Id: 8002, TemporaryAddress: "0xhuman8002"}
	reserveBot(t, botStore, bot)
	require.True(t, botInRedisSet(t, ":bots:ingame:set", bot))

	gameID := int64(88001)
	match := &dao.TournamentMatch{
		TournamentID:       "99001",
		RoundNo:            1,
		MatchNo:            1,
		Player1ID:          bot.Id,
		Player1TempAddress: bot.TemporaryAddress,
		Player2ID:          human.Id,
		Player2TempAddress: human.TemporaryAddress,
		GameID:             &gameID,
		Status:             dao.TournamentMatchStatusPlaying,
	}
	require.NoError(t, db.Get().Create(match).Error)

	gr := &dao.GameResult{
		GameID:         gameID,
		GameType:       proto.GameType_TOURNAMENT,
		GameResultType: proto.GameResultType_GAME_ABORTED,
	}
	require.NoError(t, db.Get().Create(gr).Error)

	coord := newCoordinator(context.Background(), noopPublisher{}, noopPublisher{}, botStore, &noopGameCreator{}, 1000, 2, 3600, 180, 180, 15, 10)
	require.NoError(t, coord.onGameCompleted(gameID))

	require.False(t, botInRedisSet(t, ":bots:ingame:set", bot))
	require.True(t, botInRedisSet(t, ":bots:idle:set", bot))
}

func TestStartGamesForNewMatches_CreateFailure_ReleasesBots(t *testing.T) {
	setupSQLite(t)
	botStore := setupBotRedis(t)

	bot := types.PlayerAddress{Id: 8101, TemporaryAddress: "0xbot8101"}
	human := types.PlayerAddress{Id: 8102, TemporaryAddress: "0xhuman8102"}
	reserveBot(t, botStore, bot)
	require.True(t, botInRedisSet(t, ":bots:ingame:set", bot))

	match := &dao.TournamentMatch{
		TournamentID:       "99002",
		RoundNo:            1,
		MatchNo:            1,
		Player1ID:          bot.Id,
		Player1TempAddress: bot.TemporaryAddress,
		Player2ID:          human.Id,
		Player2TempAddress: human.TemporaryAddress,
		Status:             dao.TournamentMatchStatusMatched,
	}
	require.NoError(t, db.Get().Create(match).Error)

	coord := newCoordinator(context.Background(), noopPublisher{}, noopPublisher{}, botStore, &failTournamentGameCreator{}, 1000, 2, 3600, 180, 180, 15, 10)
	require.False(t, coord.startGamesForNewMatches([]uint{match.ID}))

	require.False(t, botInRedisSet(t, ":bots:ingame:set", bot))
	require.True(t, botInRedisSet(t, ":bots:idle:set", bot))
}
