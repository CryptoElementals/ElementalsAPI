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

var ErrNotFound = redis.ErrNil

var globalPool *redis.Pool

func Init(cfg *RedisConfig) *redis.Pool {
	pool := &redis.Pool{
		MaxIdle:     cfg.Size,
		IdleTimeout: 240 * time.Second,
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		Dial: func() (redis.Conn, error) {
			return dial("tcp", cfg.Address, cfg.Address)
		},
	}
	globalPool = pool
	return pool
}

func dial(network, address, password string) (redis.Conn, error) {
	c, err := redis.Dial(network, address)
	if err != nil {
		return nil, err
	}
	if password != "" {
		if _, err := c.Do("AUTH", password); err != nil {
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
	res, err := redis.String(conn.Do("GET", key))
	if err == redis.ErrNil {
		return res, ErrNotFound
	}
	return res, err
}

func Set(key string, val string) (any, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return conn.Do("SET", key, val)
}

// expire by seconds
func SetExpire(key string, val string, expire int) (any, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return conn.Do("SET", key, val, "EX", expire)
}

func Exist(key string) (bool, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Bool(conn.Do("EXIST", key))
}
