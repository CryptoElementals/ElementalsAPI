package player_info

import (
	"context"
	"fmt"
	"strconv"

	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

const defaultNamespace = "lobby:v1"

type RedisStore struct {
	pool       redis.RedisPool
	namespace  string
	queueZSet  string
	pendingMap string
	inGameSet  string
}

func (s *RedisStore) IsInQueue(_ context.Context, player types.PlayerAddress) (bool, error) {
	return redis.ZScoreMemberExists(s.queueZSet, key(player))
}

// QueueJoinedAtMs returns the queue ZSET score (join time in milliseconds) when the player is in the queue.
func (s *RedisStore) QueueJoinedAtMs(_ context.Context, player types.PlayerAddress) (ms int64, ok bool, err error) {
	return redis.ZScoreInt64IfMember(s.queueZSet, key(player))
}

func (s *RedisStore) ListQueuedPlayers(_ context.Context) ([]types.PlayerAddress, error) {
	keys, err := redis.ZRange(s.queueZSet, 0, -1)
	if err != nil {
		return nil, err
	}
	out := make([]types.PlayerAddress, 0, len(keys))
	for _, k := range keys {
		var p types.PlayerAddress
		if err := p.Parse(k); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func NewRedisStore(namespace string) (*RedisStore, error) {
	pool := redis.GetGlobalPool()
	if pool == nil {
		return nil, fmt.Errorf("redis global pool is not initialized")
	}
	if namespace == "" {
		namespace = defaultNamespace
	}
	return &RedisStore{
		pool:       pool,
		namespace:  namespace,
		queueZSet:  namespace + ":queue:zset",
		pendingMap: namespace + ":pending:hash",
		inGameSet:  namespace + ":ingame:set",
	}, nil
}

var queueAddScript = redis.NewScript(3, `
-- KEYS[1] queue zset
-- KEYS[2] pending hash
-- KEYS[3] ingame set
-- ARGV[1] player key
-- ARGV[2] now(ms)
local p = ARGV[1]
local now = tonumber(ARGV[2])
if redis.call("HEXISTS", KEYS[2], p) == 1 then
	return 0
end
if redis.call("SISMEMBER", KEYS[3], p) == 1 then
	return 0
end
redis.call("ZADD", KEYS[1], now, p)
return 1
`)

var queueRemoveScript = redis.NewScript(1, `
-- KEYS[1] queue zset
-- ARGV[1] player key
return redis.call("ZREM", KEYS[1], ARGV[1])
`)

var setPendingPairScript = redis.NewScript(3, `
-- KEYS[1] queue zset
-- KEYS[2] pending hash
-- KEYS[3] ingame set
-- ARGV[1] player1 key
-- ARGV[2] player2 key
-- ARGV[3] match id
local p1 = ARGV[1]
local p2 = ARGV[2]
local mid = ARGV[3]
if redis.call("HEXISTS", KEYS[2], p1) == 1 or redis.call("HEXISTS", KEYS[2], p2) == 1 then
	return 0
end
if redis.call("SISMEMBER", KEYS[3], p1) == 1 or redis.call("SISMEMBER", KEYS[3], p2) == 1 then
	return 0
end
redis.call("ZREM", KEYS[1], p1, p2)
redis.call("HSET", KEYS[2], p1, mid, p2, mid)
return 1
`)

var joinQueueOrMatchScript = redis.NewScript(1, `
-- KEYS[1] queue zset
-- ARGV[1] self key
-- ARGV[2] self id
-- ARGV[3] self temp address lower
-- ARGV[4] now(ms)
local self = ARGV[1]
local selfID = ARGV[2]
local selfTemp = ARGV[3]
local now = tonumber(ARGV[4])
local candidates = redis.call("ZRANGE", KEYS[1], 0, 99)
for i = 1, #candidates do
	local c = candidates[i]
	local cid, ctemp = string.match(c, "^(%d+)_(.+)$")
	-- Match only if neither player id nor temp address matches joiner (same as legacy in-memory queue).
	if c ~= self and cid and ctemp and cid ~= selfID and ctemp ~= selfTemp then
		local removed = redis.call("ZREM", KEYS[1], c)
		if removed == 1 then
			return { "MATCH", c }
		end
	end
end
redis.call("ZADD", KEYS[1], now, self)
return { "QUEUED", "" }
`)

var firstWaitingPlayerBeforeScript = redis.NewScript(1, `
-- KEYS[1] queue zset
-- ARGV[1] cutoff(ms)
local cutoff = tonumber(ARGV[1])
local out = redis.call("ZRANGEBYSCORE", KEYS[1], "-inf", cutoff, "LIMIT", 0, 1)
if #out == 0 then
	return ""
end
return out[1]
`)

var cancelPendingPairScript = redis.NewScript(1, `
-- KEYS[1] pending hash
-- ARGV[1] match id
-- ARGV[2] player1 key
-- ARGV[3] player2 key
local m1 = redis.call("HGET", KEYS[1], ARGV[2])
local m2 = redis.call("HGET", KEYS[1], ARGV[3])
if m1 ~= ARGV[1] or m2 ~= ARGV[1] then
	return 0
end
redis.call("HDEL", KEYS[1], ARGV[2], ARGV[3])
return 1
`)

var finalizePairScript = redis.NewScript(2, `
-- KEYS[1] pending hash
-- KEYS[2] ingame set
-- ARGV[1] match id
-- ARGV[2] player1 key
-- ARGV[3] player2 key
local m1 = redis.call("HGET", KEYS[1], ARGV[2])
local m2 = redis.call("HGET", KEYS[1], ARGV[3])
if m1 ~= ARGV[1] or m2 ~= ARGV[1] then
	return 0
end
redis.call("HDEL", KEYS[1], ARGV[2], ARGV[3])
redis.call("SADD", KEYS[2], ARGV[2], ARGV[3])
return 1
`)

var markOutOfGameScript = redis.NewScript(1, `
-- KEYS[1] ingame set
-- ARGV[*] player keys
if #ARGV == 0 then
	return 0
end
return redis.call("SREM", KEYS[1], unpack(ARGV))
`)

func (s *RedisStore) AddQueue(_ context.Context, player types.PlayerAddress, nowMs int64) (bool, error) {
	out, err := redis.ScriptInt(s.pool, queueAddScript, s.queueZSet, s.pendingMap, s.inGameSet, key(player), strconv.FormatInt(nowMs, 10))
	if err != nil {
		return false, err
	}
	return out == 1, nil
}

func (s *RedisStore) RemoveQueue(_ context.Context, player types.PlayerAddress) error {
	_, err := redis.ScriptDo(s.pool, queueRemoveScript, s.queueZSet, key(player))
	return err
}

func (s *RedisStore) SetPendingPair(_ context.Context, matchID int64, p1, p2 types.PlayerAddress) (bool, error) {
	out, err := redis.ScriptInt(s.pool, setPendingPairScript, s.queueZSet, s.pendingMap, s.inGameSet, key(p1), key(p2), strconv.FormatInt(matchID, 10))
	if err != nil {
		return false, err
	}
	return out == 1, nil
}

func (s *RedisStore) CancelPendingPair(_ context.Context, matchID int64, p1, p2 types.PlayerAddress) (bool, error) {
	out, err := redis.ScriptInt(s.pool, cancelPendingPairScript, s.pendingMap, strconv.FormatInt(matchID, 10), key(p1), key(p2))
	if err != nil {
		return false, err
	}
	return out == 1, nil
}

func (s *RedisStore) FinalizeConfirmedPair(_ context.Context, matchID int64, p1, p2 types.PlayerAddress) (bool, error) {
	out, err := redis.ScriptInt(s.pool, finalizePairScript, s.pendingMap, s.inGameSet, strconv.FormatInt(matchID, 10), key(p1), key(p2))
	if err != nil {
		return false, err
	}
	return out == 1, nil
}

func (s *RedisStore) MarkPlayersOutOfGame(_ context.Context, players ...types.PlayerAddress) error {
	args := []interface{}{s.inGameSet}
	for _, p := range players {
		args = append(args, key(p))
	}
	_, err := redis.ScriptDo(s.pool, markOutOfGameScript, args...)
	return err
}

func (s *RedisStore) IsInGame(_ context.Context, player types.PlayerAddress) (bool, error) {
	return redis.SIsMember(s.inGameSet, key(player))
}

func (s *RedisStore) PendingMatchID(_ context.Context, player types.PlayerAddress) (int64, bool, error) {
	return redis.HGetInt64(s.pendingMap, key(player))
}

func (s *RedisStore) JoinQueueOrGetMatchCandidate(_ context.Context, player types.PlayerAddress, nowMs int64) (*types.PlayerAddress, bool, error) {
	reply, err := redis.ScriptValues(s.pool, joinQueueOrMatchScript, s.queueZSet, key(player), strconv.FormatInt(player.Id, 10), player.TemporaryAddress, strconv.FormatInt(nowMs, 10))
	if err != nil {
		return nil, false, err
	}
	var mode string
	var candidate string
	if _, err := redis.ScanReply(reply, &mode, &candidate); err != nil {
		return nil, false, err
	}
	if mode == "QUEUED" {
		return nil, true, nil
	}
	if mode != "MATCH" || candidate == "" {
		return nil, false, fmt.Errorf("unexpected join mode: %s", mode)
	}
	var addr types.PlayerAddress
	if err := addr.Parse(candidate); err != nil {
		return nil, false, err
	}
	return &addr, false, nil
}

func (s *RedisStore) FirstWaitingPlayerBefore(_ context.Context, cutoffMs int64) (*types.PlayerAddress, error) {
	k, err := redis.ScriptString(s.pool, firstWaitingPlayerBeforeScript, s.queueZSet, strconv.FormatInt(cutoffMs, 10))
	if err != nil {
		return nil, err
	}
	if k == "" {
		return nil, nil
	}
	var addr types.PlayerAddress
	if err := addr.Parse(k); err != nil {
		return nil, err
	}
	return &addr, nil
}

func key(a types.PlayerAddress) string {
	return types.NewPlayerAddress(a.Id, a.TemporaryAddress).String()
}
