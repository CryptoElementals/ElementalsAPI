package redis

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	err := Init(&Config{
		Address:  "10.9.23.165:6379",
		Password: "qiaoyunb",
		Size:     10,
	})
	if err != nil {
		panic(err)
	}
	m.Run()
}

func TestRedisGetSet(t *testing.T) {
	key := "foo"
	val := "bar"
	expire := 5
	// expire in 5s
	err := Set(key, val, expire)
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
	err = Set(key, val, expire)
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
	require.Equal(t, redis.ErrNil, err)
	require.Empty(t, resp)

	// exist failed
	ex, err = Exist(key)
	require.NoError(t, err)
	require.False(t, ex)

	// set key val
	err = Set(key, val, expire)
	require.NoError(t, err)
	// delet key val
	err = Delete(key)
	require.NoError(t, err)
	// get failed
	resp, err = Get(key)
	require.Equal(t, redis.ErrNil, err)
	require.Empty(t, resp)

	// exist failed
	ex, err = Exist(key)
	require.NoError(t, err)
	require.False(t, ex)
}

// TestZInitRegistersNamedPoolsFromConfigs runs last (name prefix "TestZ") so other tests still
// use the Redis from TestMain. It verifies Init builds and registers named pools from cfgs.
func TestZInitRegistersNamedPoolsFromConfigs(t *testing.T) {
	sDefault := miniredis.RunT(t)
	sExtra := miniredis.RunT(t)
	t.Cleanup(sDefault.Close)
	t.Cleanup(sExtra.Close)

	base := &Config{Address: sDefault.Addr(), Size: 2}
	extraCfg := &Config{Address: sExtra.Addr(), Size: 2}
	require.NoError(t, Init(base, &ConfigWithName{Name: "namedFromInit", Cfg: extraCfg}))

	op, err := Pool("namedFromInit")
	require.NoError(t, err)
	require.NoError(t, op.Set("ik", "iv", 0))
	v, err := op.Get("ik")
	require.NoError(t, err)
	require.Equal(t, "iv", v)

	_, err = Get("ik")
	require.Equal(t, redis.ErrNil, err)

	gotCfg, err := GetPoolConfig("namedFromInit")
	require.NoError(t, err)
	require.Equal(t, extraCfg.Address, gotCfg.Address)
}
