package redis

import (
	"github.com/CryptoElementals/common/log"
	"github.com/gomodule/redigo/redis"
)

const (
	LPUSH_COMMAND  = "LPUSH"
	RPUSH_COMMAND  = "RPUSH"
	LPOP_COMMAND   = "LPOP"
	RPOP_COMMAND   = "RPOP"
	LLEN_COMMAND   = "LLEN"
	LRANGE_COMMAND = "LRANGE"
	LREM_COMMAND   = "LREM"
	LSET_COMMAND   = "LSET"
	LINDEX_COMMAND = "LINDEX"
	LTRIM_COMMAND  = "LTRIM"
)

func LPush(key string, values ...interface{}) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := make([]interface{}, 0, len(values)+1)
	args = append(args, key)
	args = append(args, values...)
	return redis.Int(conn.Do(LPUSH_COMMAND, args...))
}

func RPush(key string, values ...interface{}) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := make([]interface{}, 0, len(values)+1)
	args = append(args, key)
	args = append(args, values...)
	return redis.Int(conn.Do(RPUSH_COMMAND, args...))
}

func LPop(key string) (string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.String(conn.Do(LPOP_COMMAND, key))
}

func RPop(key string) (string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.String(conn.Do(RPOP_COMMAND, key))
}

func LLen(key string) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int(conn.Do(LLEN_COMMAND, key))
}

func LRange(key string, start int, stop int) ([]string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Strings(conn.Do(LRANGE_COMMAND, key, start, stop))
}

func LRem(key string, count int, value interface{}) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int(conn.Do(LREM_COMMAND, key, count, value))
}

func LSet(key string, index int, value interface{}) error {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	_, err := conn.Do(LSET_COMMAND, key, index, value)
	return err
}

func LIndex(key string, index int) (string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.String(conn.Do(LINDEX_COMMAND, key, index))
}

func LTrim(key string, start int, stop int) error {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	_, err := conn.Do(LTRIM_COMMAND, key, start, stop)
	return err
}
