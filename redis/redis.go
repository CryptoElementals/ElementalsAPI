package redis

import (
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/gomodule/redigo/redis"
)

type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	Size     int    `mapstructure:"size"`
}

type RedisConn = redis.Conn

type RedisPool interface {
	Get() RedisConn
}

const (
	PING_COMMAND   = "PING"
	AUTH_COMMAND   = "AUTH"
	GET_COMMAND    = "GET"
	SET_COMMAND    = "SET"
	EXPIRE_COMMAND = "EX"
	EXISTS_COMMAND = "EXISTS"
)

var ErrNotFound = redis.ErrNil

var globalPool RedisPool

func Init(cfg *RedisConfig) {
	pool := &redis.Pool{
		MaxIdle:     cfg.Size,
		IdleTimeout: 240 * time.Second,
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do(PING_COMMAND)
			return err
		},
		Dial: func() (redis.Conn, error) {
			return dial("tcp", cfg.Address, cfg.Password)
		},
	}
	globalPool = pool
}

func SetGlobalPool(pool RedisPool) {
	globalPool = pool
}

func dial(network, address, password string) (redis.Conn, error) {
	c, err := redis.Dial(network, address)
	if err != nil {
		return nil, err
	}
	if password != "" {
		if _, err := c.Do(AUTH_COMMAND, password); err != nil {
			c.Close()
			return nil, err
		}
	}
	return c, err
}

func Get(key string) (string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	res, err := redis.String(conn.Do(GET_COMMAND, key))
	if err == redis.ErrNil {
		return res, ErrNotFound
	}
	return res, err
}

// expire by seconds
func SetWithExpire(key string, val string, expire int) error {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	_, err := conn.Do(SET_COMMAND, key, val, EXPIRE_COMMAND, expire)
	return err
}

func Exist(key string) (bool, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	exist, err := redis.Bool(conn.Do(EXISTS_COMMAND, key))
	if err == redis.ErrNil {
		return false, ErrNotFound
	}
	if err != nil {
		return false, err
	}
	if !exist {
		return false, ErrNotFound
	}
	return true, nil
}
