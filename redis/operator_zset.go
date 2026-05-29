package redis

import (
	"github.com/gomodule/redigo/redis"
)

func (o *RedisOperator) ZAdd(key string, score float64, member interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(ZADD_COMMAND, key, score, member))
}

func (o *RedisOperator) ZRem(key string, members ...interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(members)+1)
	args = append(args, key)
	args = append(args, members...)
	return redis.Int(conn.Do(ZREM_COMMAND, args...))
}

func (o *RedisOperator) ZScore(key string, member interface{}) (float64, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Float64(conn.Do(ZSCORE_COMMAND, key, member))
}

func (o *RedisOperator) zScoreRaw(key string, member interface{}) (interface{}, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return conn.Do(ZSCORE_COMMAND, key, member)
}

func (o *RedisOperator) ZScoreMemberExists(key string, member interface{}) (bool, error) {
	v, err := o.zScoreRaw(key, member)
	if err != nil {
		return false, err
	}
	if v == nil {
		return false, nil
	}
	return true, nil
}

func (o *RedisOperator) ZScoreInt64IfMember(key string, member interface{}) (score int64, ok bool, err error) {
	v, err := o.zScoreRaw(key, member)
	if err != nil {
		return 0, false, err
	}
	if v == nil {
		return 0, false, nil
	}
	f, err := redis.Float64(v, nil)
	if err != nil {
		return 0, false, err
	}
	return int64(f), true, nil
}

func (o *RedisOperator) ZCard(key string) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(ZCARD_COMMAND, key))
}

func (o *RedisOperator) ZCount(key string, min interface{}, max interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(ZCOUNT_COMMAND, key, min, max))
}

func (o *RedisOperator) ZRange(key string, start int, stop int) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Strings(conn.Do(ZRANGE_COMMAND, key, start, stop))
}

func (o *RedisOperator) ZRevRange(key string, start int, stop int) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Strings(conn.Do(ZREVRANGE_COMMAND, key, start, stop))
}

func (o *RedisOperator) ZRangeByScore(key string, min interface{}, max interface{}, offset int, count int) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Strings(conn.Do(ZRANGEBYSCORE_COMMAND, key, min, max, "LIMIT", offset, count))
}

func (o *RedisOperator) ZRevRangeByScore(key string, max interface{}, min interface{}, offset int, count int) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Strings(conn.Do(ZREVRANGEBYSCORE_COMMAND, key, max, min, "LIMIT", offset, count))
}

func (o *RedisOperator) ZRank(key string, member interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(ZRANK_COMMAND, key, member))
}

func (o *RedisOperator) ZRevRank(key string, member interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(ZREVRANK_COMMAND, key, member))
}

func (o *RedisOperator) ZIncrBy(key string, increment float64, member interface{}) (float64, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Float64(conn.Do(ZINCRBY_COMMAND, key, increment, member))
}
