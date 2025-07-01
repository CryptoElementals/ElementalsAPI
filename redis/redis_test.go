package redis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	Init(&RedisConfig{
		Address:  "address",
		Password: "password",
		Size:     10,
	})
	m.Run()
}

func TestRedisGetSet(t *testing.T) {
	key := "foo"
	val := "bar"
	expire := 5
	// expire in 5s
	err := SetWithExpire(key, val, expire)
	require.NoError(t, err)

	// get success
	resp, err := Get(key)
	require.NoError(t, err)
	require.Equal(t, val, resp)

	// exist success
	ex, err := Exist(key)
	require.NoError(t, err)
	require.True(t, ex)

	// sleep and reset expire
	time.Sleep(time.Duration(3) * time.Second)
	err = SetWithExpire(key, val, expire)
	require.NoError(t, err)

	// sleep until exceeding the first expire time
	time.Sleep(time.Duration(3) * time.Second)
	// get success
	resp, err = Get(key)
	require.NoError(t, err)
	require.Equal(t, val, resp)

	// exist success
	ex, err = Exist(key)
	require.NoError(t, err)
	require.True(t, ex)

	// sleep until exceeding the second expire time
	time.Sleep(time.Duration(3) * time.Second)

	// get failed
	resp, err = Get(key)
	require.Equal(t, ErrNotFound, err)
	require.Empty(t, resp)

	// exist failed
	ex, err = Exist(key)
	require.Equal(t, ErrNotFound, err)
	require.False(t, ex)

}
