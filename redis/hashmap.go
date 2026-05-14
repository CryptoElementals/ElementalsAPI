package redis

import (
	"strconv"

	"github.com/gomodule/redigo/redis"
)

const (
	HSET_COMMAND    = "HSET"
	HMSET_COMMAND   = "HMSET"
	HGET_COMMAND    = "HGET"
	HMGET_COMMAND   = "HMGET"
	HGETALL_COMMAND = "HGETALL"
	HDEL_COMMAND    = "HDEL"
	HEXISTS_COMMAND = "HEXISTS"
	HLEN_COMMAND    = "HLEN"
	HKEYS_COMMAND   = "HKEYS"
	HVALS_COMMAND   = "HVALS"
	HINCRBY_COMMAND = "HINCRBY"
)

func HSet(key string, field string, value interface{}) (int, error) {
	return mustDefault().HSet(key, field, value)
}

func HMSet(key string, values map[string]interface{}) error {
	return mustDefault().HMSet(key, values)
}

func HGet(key string, field string) (string, error) {
	return mustDefault().HGet(key, field)
}

// HGetInt64 parses a hash field as base-10 int64. If the field is missing, ok is false and err is nil.
func HGetInt64(key, field string) (value int64, ok bool, err error) {
	s, err := mustDefault().HGet(key, field)
	if err == redis.ErrNil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return v, true, nil
}

func HMGet(key string, fields ...string) ([]interface{}, error) {
	return mustDefault().HMGet(key, fields...)
}

func HGetAll(key string) (map[string]string, error) {
	return mustDefault().HGetAll(key)
}

func HDel(key string, fields ...string) (int, error) {
	return mustDefault().HDel(key, fields...)
}

func HExists(key string, field string) (bool, error) {
	return mustDefault().HExists(key, field)
}

func HLen(key string) (int, error) {
	return mustDefault().HLen(key)
}

func HKeys(key string) ([]string, error) {
	return mustDefault().HKeys(key)
}

func HVals(key string) ([]string, error) {
	return mustDefault().HVals(key)
}

func HIncrBy(key string, field string, increment int64) (int64, error) {
	return mustDefault().HIncrBy(key, field, increment)
}
