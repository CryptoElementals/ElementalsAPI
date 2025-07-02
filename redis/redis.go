package redis

import (
	"errors"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/gomodule/redigo/redis"
)

type Config struct {
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
	DELETE_COMMAND = "DEL"
)

var globalPool RedisPool

func Init(cfg *Config) error {
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
	// do ping for quick redis error request
	_, err := pool.Get().Do(PING_COMMAND)
	if err != nil {
		return err
	}
	globalPool = pool
	return nil
}

func SetGlobalPool(pool RedisPool) {
	globalPool = pool
}

func GetGlobalPool() RedisPool {
	return globalPool
}

func GetRedigoPool() (*redis.Pool, error) {
	p, ok := globalPool.(*redis.Pool)
	if !ok {
		return nil, errors.New("pool is not a valid redisgo redis pool")
	}
	return p, nil
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
	return res, err
}

// expire by seconds
func Set(key string, val string, expire int) error {
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

func Delete(key string) error {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	_, err := conn.Do(DELETE_COMMAND, key)
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
	return redis.Bool(conn.Do(EXISTS_COMMAND, key))
}

func Ping() error {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	_, err := conn.Do(PING_COMMAND)
	if err != nil {
		return err
	}
	return nil
}
