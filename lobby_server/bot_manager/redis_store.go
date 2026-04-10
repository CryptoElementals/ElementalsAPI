package bot_manager

import (
	"fmt"

	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

const defaultNamespace = "lobby:v1"

type RedisStore struct {
	pool      redis.RedisPool
	namespace string
	idleKey   string
	inGameKey string
	allKey    string
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
		pool:      pool,
		namespace: namespace,
		idleKey:   namespace + ":bots:idle:set",
		inGameKey: namespace + ":bots:ingame:set",
		allKey:    namespace + ":bots:all:set",
	}, nil
}

var registerScript = redis.NewScript(3, `
-- KEYS[1] idle set
-- KEYS[2] ingame set
-- KEYS[3] all set
-- ARGV[*] bot player keys
for i = 1, #ARGV do
	local p = ARGV[i]
	redis.call("SADD", KEYS[1], p)
	redis.call("SREM", KEYS[2], p)
	redis.call("SADD", KEYS[3], p)
end
return #ARGV
`)

var unregisterScript = redis.NewScript(3, `
-- KEYS[1] idle set
-- KEYS[2] ingame set
-- KEYS[3] all set
-- ARGV[*] bot player keys
for i = 1, #ARGV do
	local p = ARGV[i]
	redis.call("SREM", KEYS[1], p)
	redis.call("SREM", KEYS[2], p)
	redis.call("SREM", KEYS[3], p)
end
return #ARGV
`)

var popIdleForMatchScript = redis.NewScript(2, `
-- KEYS[1] idle set
-- KEYS[2] ingame set
local bot = redis.call("SPOP", KEYS[1])
if not bot then
	return ""
end
redis.call("SADD", KEYS[2], bot)
return bot
`)

var releaseInGameBotScript = redis.NewScript(2, `
-- KEYS[1] idle set
-- KEYS[2] ingame set
-- ARGV[1] bot player key
local p = ARGV[1]
if redis.call("SISMEMBER", KEYS[2], p) == 0 then
	return 0
end
redis.call("SREM", KEYS[2], p)
redis.call("SADD", KEYS[1], p)
return 1
`)

func (s *RedisStore) RegisterBots(addrs ...types.PlayerAddress) error {
	if len(addrs) == 0 {
		return nil
	}
	args := []interface{}{s.idleKey, s.inGameKey, s.allKey}
	for _, addr := range addrs {
		args = append(args, toPlayerKey(addr))
	}
	_, err := redis.ScriptDo(s.pool, registerScript, args...)
	return err
}

func (s *RedisStore) UnregisterBots(addrs ...types.PlayerAddress) error {
	if len(addrs) == 0 {
		return nil
	}
	args := []interface{}{s.idleKey, s.inGameKey, s.allKey}
	for _, addr := range addrs {
		args = append(args, toPlayerKey(addr))
	}
	_, err := redis.ScriptDo(s.pool, unregisterScript, args...)
	return err
}

func (s *RedisStore) PopIdleBotForMatch() (*types.PlayerAddress, error) {
	key, err := redis.ScriptString(s.pool, popIdleForMatchScript, s.idleKey, s.inGameKey)
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

func (s *RedisStore) ReleaseInGameBot(addr types.PlayerAddress) (bool, error) {
	ok, err := redis.ScriptInt(s.pool, releaseInGameBotScript, s.idleKey, s.inGameKey, toPlayerKey(addr))
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
