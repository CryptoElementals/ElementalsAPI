package redis

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func TestRegisterPoolAndPoolOperator(t *testing.T) {
	s := miniredis.RunT(t)
	t.Cleanup(s.Close)

	extraPool := &redis.Pool{
		MaxIdle:     2,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", s.Addr())
		},
	}
	_, err := extraPool.Get().Do("PING")
	require.NoError(t, err)

	require.NoError(t, registerPool("manualExtra", extraPool))
	op, err := Pool("manualExtra")
	require.NoError(t, err)
	require.NotNil(t, op)

	require.NoError(t, op.Set("namedpoolkey", "v1", 0))
	v, err := op.Get("namedpoolkey")
	require.NoError(t, err)
	require.Equal(t, "v1", v)
}

func TestRegisterPoolReservedDefaultName(t *testing.T) {
	s := miniredis.RunT(t)
	t.Cleanup(s.Close)
	p := &redis.Pool{
		MaxIdle: 1,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", s.Addr())
		},
	}
	_, err := p.Get().Do("PING")
	require.NoError(t, err)

	err = registerPool(defaultPoolName, p)
	require.Error(t, err)
}

func TestRegisterPoolDuplicate(t *testing.T) {
	s := miniredis.RunT(t)
	t.Cleanup(s.Close)
	p := &redis.Pool{
		MaxIdle: 1,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", s.Addr())
		},
	}
	_, err := p.Get().Do("PING")
	require.NoError(t, err)

	require.NoError(t, registerPool("dup", p))
	err = registerPool("dup", p)
	require.Error(t, err)
}

func TestPoolUnknownName(t *testing.T) {
	_, err := Pool("no_such_pool_xyz")
	require.Error(t, err)
}

func TestGetPoolConfigUnknownName(t *testing.T) {
	_, err := GetPoolConfig("no_such_named_config_xyz")
	require.Error(t, err)
}

func TestGetPoolConfigDefaultMatchesGetConfig(t *testing.T) {
	d, err := GetPoolConfig("default")
	require.NoError(t, err)
	require.NotNil(t, d)
	require.Equal(t, GetConfig(), d)
	empty, err := GetPoolConfig("")
	require.NoError(t, err)
	require.Equal(t, d, empty)
}

func TestInitDuplicateNamedConfigRejected(t *testing.T) {
	s := miniredis.RunT(t)
	t.Cleanup(s.Close)
	base := &Config{Address: s.Addr(), Size: 2}
	a := &ConfigWithName{Name: "poolA", Cfg: base}
	err := Init(base, a, &ConfigWithName{Name: "poolA", Cfg: base})
	require.Error(t, err)
}

func TestInitReservedConfigNameRejected(t *testing.T) {
	s := miniredis.RunT(t)
	t.Cleanup(s.Close)
	base := &Config{Address: s.Addr(), Size: 2}
	err := Init(base, &ConfigWithName{Name: defaultPoolName, Cfg: base})
	require.Error(t, err)
}
