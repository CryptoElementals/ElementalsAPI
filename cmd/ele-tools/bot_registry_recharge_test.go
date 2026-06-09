package main

import (
	"testing"
	"time"

	"github.com/CryptoElementals/common/redis"
	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

func TestFilterTokenInsufficientBots(t *testing.T) {
	candidates := []string{"1_0xaaa", "2_0xbbb", "3_0xccc"}

	t.Run("no filter no limit", func(t *testing.T) {
		got, skipped := filterTokenInsufficientBots(candidates, "", 0)
		require.Equal(t, candidates, got)
		require.Equal(t, 0, skipped)
	})

	t.Run("limit", func(t *testing.T) {
		got, skipped := filterTokenInsufficientBots(candidates, "", 2)
		require.Equal(t, []string{"1_0xaaa", "2_0xbbb"}, got)
		require.Equal(t, 0, skipped)
	})

	t.Run("player key found", func(t *testing.T) {
		got, skipped := filterTokenInsufficientBots(candidates, "2_0xbbb", 0)
		require.Equal(t, []string{"2_0xbbb"}, got)
		require.Equal(t, 0, skipped)
	})

	t.Run("player key not found", func(t *testing.T) {
		got, skipped := filterTokenInsufficientBots(candidates, "9_0xzzz", 0)
		require.Nil(t, got)
		require.Equal(t, 1, skipped)
	})

	t.Run("player key with limit", func(t *testing.T) {
		got, skipped := filterTokenInsufficientBots(candidates, "2_0xbbb", 1)
		require.Equal(t, []string{"2_0xbbb"}, got)
		require.Equal(t, 0, skipped)
	})
}

func TestPromoteBotRegistryToIdle(t *testing.T) {
	s := miniredis.RunT(t)
	require.NoError(t, redis.Init(&redis.Config{Address: s.Addr(), Size: 4}))

	const ns = "lobby:v1"
	keys := botRegistryKeys(ns)
	botKey := "100_0xbot100"
	nowMs := time.Now().UnixMilli()

	s.SAdd(ns+":bots:all:set", botKey)
	s.SAdd(ns+":bots:token_insufficient:set", botKey)
	s.HSet(ns+":bots:ingame:hash", botKey, "1")

	err := promoteBotRegistryToIdle(keys, botKey, nowMs)
	require.NoError(t, err)

	ok, err := redis.SIsMember(keys.idleKey, botKey)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = redis.SIsMember(keys.tokenInsufficientKey, botKey)
	require.NoError(t, err)
	require.False(t, ok)

	exists, err := redis.HExists(keys.inGameKey, botKey)
	require.NoError(t, err)
	require.False(t, exists)

	score, err := redis.ZScore(keys.lastSeenKey, botKey)
	require.NoError(t, err)
	require.Equal(t, float64(nowMs), score)
}
