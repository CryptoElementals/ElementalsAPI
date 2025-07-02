package cache

import (
	"testing"
	"time"

	"github.com/CryptoElementals/common/redis"
	"github.com/stretchr/testify/require"
)

func TestMemCache(t *testing.T) {
	c := NewMemCache()
	testCache(t, c)
}

func TestRedisCache(t *testing.T) {
	err := redis.Init(&redis.Config{
		Address:  "10.9.23.165:6379",
		Password: "qiaoyunb",
		Size:     10,
	})
	if err != nil {
		panic(err)
	}
	c, err := NewRedisCache()
	require.NoError(t, err)
	testCache(t, c)
}

func testCache(t *testing.T, c Cache) {
	key := "foo"
	val := "bar"
	expire := 5
	// expire in 5s
	err := c.Set(key, val, expire)
	require.NoError(t, err)

	// get success
	resp, err := c.Get(key)
	require.NoError(t, err)
	require.Equal(t, val, resp)

	// exist success
	ex, err := c.Exist(key)
	require.NoError(t, err)
	require.True(t, ex)

	// sleep and reset expire
	time.Sleep(time.Duration(3) * time.Second)
	err = c.Set(key, val, expire)
	require.NoError(t, err)

	// sleep until exceeding the first expire time
	time.Sleep(time.Duration(3) * time.Second)
	// get success
	resp, err = c.Get(key)
	require.NoError(t, err)
	require.Equal(t, val, resp)

	// exist success
	ex, err = c.Exist(key)
	require.NoError(t, err)
	require.True(t, ex)

	// sleep until exceeding the second expire time
	time.Sleep(time.Duration(3) * time.Second)

	// get failed
	resp, err = c.Get(key)
	require.Equal(t, ErrNotFound, err)
	require.Empty(t, resp)

	// exist failed
	ex, err = c.Exist(key)
	require.Equal(t, ErrNotFound, err)
	require.False(t, ex)

	// set key val
	err = c.Set(key, val, expire)
	require.NoError(t, err)
	// delet key val
	err = c.Delete(key)
	require.NoError(t, err)
	// get failed
	resp, err = c.Get(key)
	require.Equal(t, ErrNotFound, err)
	require.Empty(t, resp)

	// exist failed
	ex, err = c.Exist(key)
	require.Equal(t, ErrNotFound, err)
	require.False(t, ex)
}
