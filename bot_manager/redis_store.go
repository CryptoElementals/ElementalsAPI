package bot_manager

import (
	"context"
	"errors"
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"gorm.io/gorm"
)

const defaultNamespace = "lobby:v1"

type RedisStore struct {
	pool                       redis.RedisPool
	idleKey                    string
	inGameKey                  string
	allKey                     string
	lastSeenKey                string
	tokenInsufficientKey       string
	tokenInsufficientThreshold int64
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
		pool:                 pool,
		idleKey:              namespace + ":bots:idle:set",
		inGameKey:            namespace + ":bots:ingame:set",
		allKey:               namespace + ":bots:all:set",
		lastSeenKey:          namespace + ":bots:last_seen:zset",
		tokenInsufficientKey: namespace + ":bots:token_insufficient:set",
	}, nil
}

// SetTokenInsufficientThreshold sets the minimum token balance required to return a released
// bot to the idle matchmaking set. When <= 0, released bots are not routed to the token-insufficient set.
func (s *RedisStore) SetTokenInsufficientThreshold(threshold int64) {
	if s == nil {
		return
	}
	s.tokenInsufficientThreshold = threshold
}

var upsertAliveScript = redis.NewScript(5, `
-- KEYS[1] idle set
-- KEYS[2] ingame set
-- KEYS[3] all set
-- KEYS[4] last_seen zset
-- KEYS[5] token_insufficient set
-- ARGV[1] now ms
-- ARGV[2...] bot player keys
local now = tonumber(ARGV[1])
if not now then
	return 0
end
for i = 2, #ARGV do
	local p = ARGV[i]
	redis.call("SADD", KEYS[3], p)
	redis.call("ZADD", KEYS[4], now, p)
	-- Only promote to idle when not reserved for a match or token-insufficient quarantine.
	if redis.call("SISMEMBER", KEYS[2], p) == 0 and redis.call("SISMEMBER", KEYS[5], p) == 0 then
		redis.call("SADD", KEYS[1], p)
	end
end
return math.max(0, #ARGV - 1)
`)

var heartbeatScript = redis.NewScript(1, `
-- KEYS[1] last_seen zset
-- ARGV[1] now ms
-- ARGV[2...] bot player keys
local now = tonumber(ARGV[1])
if not now then
	return 0
end
for i = 2, #ARGV do
	local p = ARGV[i]
	redis.call("ZADD", KEYS[1], now, p)
end
return math.max(0, #ARGV - 1)
`)

var popFreshIdleForMatchScript = redis.NewScript(3, `
-- KEYS[1] idle set
-- KEYS[2] ingame set
-- KEYS[3] last_seen zset
-- ARGV[1] cutoff ms
local cutoff = tonumber(ARGV[1])
if not cutoff then
	return ""
end
while true do
	local bot = redis.call("SPOP", KEYS[1])
	if not bot then
		return ""
	end
	local score = redis.call("ZSCORE", KEYS[3], bot)
	if score and tonumber(score) and tonumber(score) >= cutoff then
		redis.call("SADD", KEYS[2], bot)
		return bot
	end
end
`)

var releaseInGameBotScript = redis.NewScript(3, `
-- KEYS[1] idle set
-- KEYS[2] ingame set
-- KEYS[3] token_insufficient set
-- Freshness is enforced when popping from idle (see popFreshIdleForMatchScript).
-- ARGV[1] bot player key
-- ARGV[2] token threshold (0 = disabled)
-- ARGV[3] current token amount
local p = ARGV[1]
if redis.call("SISMEMBER", KEYS[2], p) == 0 then
	return 0
end
local thresh = tonumber(ARGV[2])
local tokens = tonumber(ARGV[3])
redis.call("SREM", KEYS[2], p)
if thresh and thresh > 0 and tokens and tokens < thresh then
	redis.call("SADD", KEYS[3], p)
else
	redis.call("SADD", KEYS[1], p)
end
return 1
`)

func (s *RedisStore) UpsertAliveBots(nowMs int64, addrs ...types.PlayerAddress) error {
	if len(addrs) == 0 {
		return nil
	}
	args := []interface{}{s.idleKey, s.inGameKey, s.allKey, s.lastSeenKey, s.tokenInsufficientKey, nowMs}
	for _, addr := range addrs {
		args = append(args, toPlayerKey(addr))
	}
	_, err := redis.ScriptDo(s.pool, upsertAliveScript, args...)
	return err
}

func (s *RedisStore) HeartbeatBots(nowMs int64, addrs ...types.PlayerAddress) error {
	if len(addrs) == 0 {
		return nil
	}
	args := []interface{}{s.lastSeenKey, nowMs}
	for _, addr := range addrs {
		args = append(args, toPlayerKey(addr))
	}
	_, err := redis.ScriptDo(s.pool, heartbeatScript, args...)
	return err
}

func (s *RedisStore) MarkBotsStopping(addrs ...types.PlayerAddress) error {
	if len(addrs) == 0 {
		return nil
	}
	const shutdownScore int64 = 1
	return s.HeartbeatBots(shutdownScore, addrs...)
}

func (s *RedisStore) PopFreshIdleBotForMatch(nowMs int64, freshnessMs int64) (*types.PlayerAddress, error) {
	cutoff := nowMs - freshnessMs
	key, err := redis.ScriptString(s.pool, popFreshIdleForMatchScript, s.idleKey, s.inGameKey, s.lastSeenKey, cutoff)
	if err != nil {
		return nil, err
	}
	if key == "" {
		return nil, nil
	}
	var out types.PlayerAddress
	if err := out.Parse(key); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *RedisStore) ReleaseInGameBot(ctx context.Context, addr types.PlayerAddress) (bool, error) {
	playerTokens := int64(0)
	ut, err := db.GetPlayerToken(ctx, addr.Id)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, err
		}
	} else {
		playerTokens = int64(ut.TokenAmount)
	}
	ok, err := redis.ScriptInt(s.pool, releaseInGameBotScript, s.idleKey, s.inGameKey, s.tokenInsufficientKey, toPlayerKey(addr), s.tokenInsufficientThreshold, playerTokens)
	if err != nil {
		return false, err
	}
	return ok == 1, nil
}

func (s *RedisStore) IsBot(addr types.PlayerAddress) (bool, error) {
	return redis.SIsMember(s.allKey, toPlayerKey(addr))
}

func toPlayerKey(a types.PlayerAddress) string {
	return types.NewPlayerAddress(a.Id, a.TemporaryAddress).String()
}
