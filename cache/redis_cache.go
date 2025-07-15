package cache

import (
	"github.com/CryptoElementals/common/redis"
	redigo "github.com/gomodule/redigo/redis"
)

type RedisCache struct{}

func NewRedisCache() (Cache, error) {
	err := redis.Ping()
	if err != nil {
		return nil, err
	}
	return &RedisCache{}, nil
}

func (*RedisCache) Get(key string) (string, error) {
	val, err := redis.Get(key)
	if err == redigo.ErrNil {
		return "", ErrNotFound
	}
	return val, err
}
func (*RedisCache) Set(key string, val string, expire int) error {
	return redis.Set(key, val, expire)
}
func (*RedisCache) Exist(key string) (bool, error) {
	exist, err := redis.Exist(key)
	if err == redigo.ErrNil {
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
func (*RedisCache) Delete(key string) error {
	return redis.Delete(key)
}

// List implements Cache.
func (r *RedisCache) List(prefix string) ([]string, error) {
	return redis.Scan(prefix)
}
