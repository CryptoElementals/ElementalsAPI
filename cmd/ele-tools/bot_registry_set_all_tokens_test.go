package main

import (
	"testing"
	"time"

	"github.com/CryptoElementals/common/redis"
	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

func TestSetAllTokensRegistryBehavior(t *testing.T) {
	s := miniredis.RunT(t)
	require.NoError(t, redis.Init(&redis.Config{Address: s.Addr(), Size: 4}))

	const ns = "lobby:v1"
	keys := botRegistryKeys(ns)
	nowMs := time.Now().UnixMilli()

	t.Run("token_insufficient promoted to idle", func(t *testing.T) {
		botKey := "100_0xbot100"
		s.SAdd(ns+":bots:all:set", botKey)
		s.SAdd(ns+":bots:token_insufficient:set", botKey)

		inTokenInsufficient, err := redis.SIsMember(keys.tokenInsufficientKey, botKey)
		require.NoError(t, err)
		require.True(t, inTokenInsufficient)

		err = promoteBotRegistryToIdle(keys, botKey, nowMs)
		require.NoError(t, err)

		ok, err := redis.SIsMember(keys.idleKey, botKey)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = redis.SIsMember(keys.tokenInsufficientKey, botKey)
		require.NoError(t, err)
		require.False(t, ok)
	})

	t.Run("ingame bot unchanged when not promoted", func(t *testing.T) {
		botKey := "200_0xbot200"
		s.SAdd(ns+":bots:all:set", botKey)
		s.HSet(ns+":bots:ingame:hash", botKey, "1")

		inTokenInsufficient, err := redis.SIsMember(keys.tokenInsufficientKey, botKey)
		require.NoError(t, err)
		require.False(t, inTokenInsufficient)

		exists, err := redis.HExists(keys.inGameKey, botKey)
		require.NoError(t, err)
		require.True(t, exists)

		ok, err := redis.SIsMember(keys.idleKey, botKey)
		require.NoError(t, err)
		require.False(t, ok)
	})
}
