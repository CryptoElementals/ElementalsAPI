package redis

import (
	"github.com/CryptoElementals/common/log"
	"github.com/gomodule/redigo/redis"
)

const (
	ZADD_COMMAND             = "ZADD"
	ZREM_COMMAND             = "ZREM"
	ZSCORE_COMMAND           = "ZSCORE"
	ZCARD_COMMAND            = "ZCARD"
	ZCOUNT_COMMAND           = "ZCOUNT"
	ZRANGE_COMMAND           = "ZRANGE"
	ZREVRANGE_COMMAND        = "ZREVRANGE"
	ZRANGEBYSCORE_COMMAND    = "ZRANGEBYSCORE"
	ZREVRANGEBYSCORE_COMMAND = "ZREVRANGEBYSCORE"
	ZRANK_COMMAND            = "ZRANK"
	ZREVRANK_COMMAND         = "ZREVRANK"
	ZINCRBY_COMMAND          = "ZINCRBY"
)

func ZAdd(key string, score float64, member interface{}) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int(conn.Do(ZADD_COMMAND, key, score, member))
}

func ZRem(key string, members ...interface{}) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := make([]interface{}, 0, len(members)+1)
	args = append(args, key)
	args = append(args, members...)
	return redis.Int(conn.Do(ZREM_COMMAND, args...))
}

func ZScore(key string, member interface{}) (float64, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Float64(conn.Do(ZSCORE_COMMAND, key, member))
}

// ZScoreMemberExists reports whether member is present in the sorted set (ZSCORE is not nil).
func ZScoreMemberExists(key string, member interface{}) (bool, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	v, err := conn.Do(ZSCORE_COMMAND, key, member)
	if err != nil {
		return false, err
	}
	if v == nil {
		return false, nil
	}
	return true, nil
}

func ZCard(key string) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int(conn.Do(ZCARD_COMMAND, key))
}

func ZCount(key string, min interface{}, max interface{}) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int(conn.Do(ZCOUNT_COMMAND, key, min, max))
}

func ZRange(key string, start int, stop int) ([]string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Strings(conn.Do(ZRANGE_COMMAND, key, start, stop))
}

func ZRevRange(key string, start int, stop int) ([]string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Strings(conn.Do(ZREVRANGE_COMMAND, key, start, stop))
}

func ZRangeByScore(key string, min interface{}, max interface{}, offset int, count int) ([]string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Strings(conn.Do(ZRANGEBYSCORE_COMMAND, key, min, max, "LIMIT", offset, count))
}

func ZRevRangeByScore(key string, max interface{}, min interface{}, offset int, count int) ([]string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Strings(conn.Do(ZREVRANGEBYSCORE_COMMAND, key, max, min, "LIMIT", offset, count))
}

func ZRank(key string, member interface{}) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int(conn.Do(ZRANK_COMMAND, key, member))
}

func ZRevRank(key string, member interface{}) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int(conn.Do(ZREVRANK_COMMAND, key, member))
}

func ZIncrBy(key string, increment float64, member interface{}) (float64, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Float64(conn.Do(ZINCRBY_COMMAND, key, increment, member))
}
