package redis

import (
	"strconv"

	"github.com/CryptoElementals/common/log"
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
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int(conn.Do(HSET_COMMAND, key, field, value))
}

func HMSet(key string, values map[string]interface{}) error {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := make([]interface{}, 0, len(values)*2+1)
	args = append(args, key)
	for field, value := range values {
		args = append(args, field, value)
	}
	_, err := conn.Do(HMSET_COMMAND, args...)
	return err
}

func HGet(key string, field string) (string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.String(conn.Do(HGET_COMMAND, key, field))
}

// HGetInt64 parses a hash field as base-10 int64. If the field is missing, ok is false and err is nil.
func HGetInt64(key, field string) (value int64, ok bool, err error) {
	s, err := HGet(key, field)
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
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := make([]interface{}, 0, len(fields)+1)
	args = append(args, key)
	for _, field := range fields {
		args = append(args, field)
	}
	return redis.Values(conn.Do(HMGET_COMMAND, args...))
}

func HGetAll(key string) (map[string]string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.StringMap(conn.Do(HGETALL_COMMAND, key))
}

func HDel(key string, fields ...string) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := make([]interface{}, 0, len(fields)+1)
	args = append(args, key)
	for _, field := range fields {
		args = append(args, field)
	}
	return redis.Int(conn.Do(HDEL_COMMAND, args...))
}

func HExists(key string, field string) (bool, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Bool(conn.Do(HEXISTS_COMMAND, key, field))
}

func HLen(key string) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int(conn.Do(HLEN_COMMAND, key))
}

func HKeys(key string) ([]string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Strings(conn.Do(HKEYS_COMMAND, key))
}

func HVals(key string) ([]string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Strings(conn.Do(HVALS_COMMAND, key))
}

func HIncrBy(key string, field string, increment int64) (int64, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int64(conn.Do(HINCRBY_COMMAND, key, field, increment))
}
