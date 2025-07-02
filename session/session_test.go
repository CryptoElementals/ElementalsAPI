package session

import (
	"testing"

	"github.com/CryptoElementals/common/redis"
	"github.com/stretchr/testify/require"
)

func TestRedisSession(t *testing.T) {
	err := redis.Init(&redis.Config{
		Address:  "10.9.23.165:6379",
		Password: "qiaoyunb",
		Size:     10,
	})
	require.NoError(t, err)
	pool, err := redis.GetRedigoPool()
	require.NoError(t, err)
	_, err = New(pool)
	require.NoError(t, err)

}
