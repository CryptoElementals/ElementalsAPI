package queue

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/CryptoElementals/common/bot_manager"
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/stretchr/testify/require"
)

const botRedisNamespace = "lobby:v1"

type noopQueueGameCreator struct{}

func (noopQueueGameCreator) CreatePVPGameAndRun(_ []types.PlayerAddress, _ int64) (int64, error) {
	return 0, nil
}

func reserveBotForTest(t *testing.T, store *bot_manager.RedisStore, addr types.PlayerAddress) {
	t.Helper()
	now := time.Now().UnixMilli()
	require.NoError(t, store.UpsertAliveBots(now, addr))
	popped, err := store.PopFreshIdleBotForMatch(now, 60_000, proto.GameType_PVP)
	require.NoError(t, err)
	require.NotNil(t, popped)
	require.Equal(t, addr.String(), popped.String())
	gameTypeStr, err := redis.HGet(botRedisNamespace+":bots:ingame:hash", addr.String())
	require.NoError(t, err)
	require.Equal(t, strconv.Itoa(int(proto.GameType_PVP)), gameTypeStr)
}

func botInRedisSet(t *testing.T, setSuffix string, addr types.PlayerAddress) bool {
	t.Helper()
	ok, err := redis.SIsMember(botRedisNamespace+setSuffix, addr.String())
	require.NoError(t, err)
	return ok
}

func botInRedisInGameHash(t *testing.T, addr types.PlayerAddress) bool {
	t.Helper()
	ok, err := redis.HExists(botRedisNamespace+":bots:ingame:hash", addr.String())
	require.NoError(t, err)
	return ok
}

func newTestQueueWithBotStore(t *testing.T) (*Queue, *bot_manager.RedisStore) {
	t.Helper()
	testMiniRedis.FlushAll()
	botStore, err := bot_manager.NewRedisStore("")
	require.NoError(t, err)
	q, err := NewQueue(context.Background(), noopEventPublisher{}, botStore, noopQueueGameCreator{}, 60, 0, 0, 0, 0, 1000, "")
	require.NoError(t, err)
	return q, botStore
}

func TestAbortPendingMatch_ReleasesInGameBot(t *testing.T) {
	q, botStore := newTestQueueWithBotStore(t)

	bot := types.PlayerAddress{Id: 901, TemporaryAddress: "0xbot901"}
	human := types.PlayerAddress{Id: 902, TemporaryAddress: "0xhuman902"}
	reserveBotForTest(t, botStore, bot)
	require.True(t, botInRedisInGameHash(t, bot))

	require.NoError(t, db.Get().Create(&dao.UserToken{PlayerId: human.Id, TokenAmount: 10000}).Error)
	require.NoError(t, db.LockUserToken(context.Background(), human.Id, human.TemporaryAddress, 1000, ""))

	gm := &dao.GameMatch{
		Player1ID:          bot.Id,
		Player1TempAddress: bot.TemporaryAddress,
		Player2ID:          human.Id,
		Player2TempAddress: human.TemporaryAddress,
		Status:             dao.GameMatchStatusPending,
		GameType:           uint(proto.GameType_PVP),
	}
	require.NoError(t, db.Get().Create(gm).Error)

	require.NoError(t, q.abortPendingMatch(gm.ID, false, false))

	require.False(t, botInRedisInGameHash(t, bot))
	require.True(t, botInRedisSet(t, ":bots:idle:set", bot))

	require.NoError(t, db.Get().First(&gm, "id = ?", gm.ID).Error)
	require.Equal(t, dao.GameMatchStatusCancelled, gm.Status)
}

func TestBotNotifyFailureCleanup_ReleasesInGameBot(t *testing.T) {
	q, botStore := newTestQueueWithBotStore(t)

	bot := types.PlayerAddress{Id: 903, TemporaryAddress: "0xbot903"}
	human := types.PlayerAddress{Id: 904, TemporaryAddress: "0xhuman904"}
	reserveBotForTest(t, botStore, bot)
	require.True(t, botInRedisInGameHash(t, bot))

	require.NoError(t, db.Get().Create(&dao.UserToken{PlayerId: human.Id, TokenAmount: 10000}).Error)
	require.NoError(t, db.LockUserToken(context.Background(), human.Id, human.TemporaryAddress, 1000, ""))

	gm := &dao.GameMatch{
		Player1ID:          bot.Id,
		Player1TempAddress: bot.TemporaryAddress,
		Player2ID:          human.Id,
		Player2TempAddress: human.TemporaryAddress,
		Status:             dao.GameMatchStatusPending,
		GameType:           uint(proto.GameType_PVP),
	}
	require.NoError(t, db.Get().Create(gm).Error)

	err := q.postMatchNotifyPending(&dao.GameMatch{ID: gm.ID, Status: dao.GameMatchStatusCancelled})
	require.Error(t, err)

	_ = q.releaseInGameBot(bot)
	_ = q.abortPendingMatch(gm.ID, false, false)

	require.False(t, botInRedisInGameHash(t, bot))
	require.True(t, botInRedisSet(t, ":bots:idle:set", bot))
}
