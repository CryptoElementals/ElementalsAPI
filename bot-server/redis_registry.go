package botserver

import (
	"fmt"

	"github.com/CryptoElementals/common/lobby_server/bot_manager"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/room_server/worker/types"
)

type redisBotRegistry struct {
	store *bot_manager.RedisStore
}

func newRedisBotRegistry(namespace string) (*redisBotRegistry, error) {
	if redis.GetGlobalPool() == nil {
		return nil, fmt.Errorf("redis global pool is not initialized")
	}
	store, err := bot_manager.NewRedisStore(namespace)
	if err != nil {
		return nil, err
	}
	return &redisBotRegistry{store: store}, nil
}

func (r *redisBotRegistry) UpsertAlive(nowMs int64, addrs []types.PlayerAddress) error {
	return r.store.UpsertAliveBots(nowMs, addrs...)
}

func (r *redisBotRegistry) Heartbeat(nowMs int64, addrs []types.PlayerAddress) error {
	return r.store.HeartbeatBots(nowMs, addrs...)
}

func (r *redisBotRegistry) MarkStopping(addrs []types.PlayerAddress) error {
	return r.store.MarkBotsStopping(addrs...)
}
