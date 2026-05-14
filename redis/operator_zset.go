package redis

import (
	"github.com/gomodule/redigo/redis"
)

func (o *redisOperator) ZAdd(key string, score float64, member interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(ZADD_COMMAND, key, score, member))
}

func (o *redisOperator) ZRem(key string, members ...interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(members)+1)
	args = append(args, key)
	args = append(args, members...)
	return redis.Int(conn.Do(ZREM_COMMAND, args...))
}

func (o *redisOperator) ZScore(key string, member interface{}) (float64, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Float64(conn.Do(ZSCORE_COMMAND, key, member))
}

func (o *redisOperator) zScoreRaw(key string, member interface{}) (interface{}, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return conn.Do(ZSCORE_COMMAND, key, member)
}

func (o *redisOperator) ZScoreMemberExists(key string, member interface{}) (bool, error) {
	v, err := o.zScoreRaw(key, member)
	if err != nil {
		return false, err
	}
	if v == nil {
		return false, nil
	}
	return true, nil
}

func (o *redisOperator) ZScoreInt64IfMember(key string, member interface{}) (score int64, ok bool, err error) {
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

func (o *redisOperator) ZCard(key string) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(ZCARD_COMMAND, key))
}

func (o *redisOperator) ZCount(key string, min interface{}, max interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(ZCOUNT_COMMAND, key, min, max))
}

func (o *redisOperator) ZRange(key string, start int, stop int) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Strings(conn.Do(ZRANGE_COMMAND, key, start, stop))
}

func (o *redisOperator) ZRevRange(key string, start int, stop int) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Strings(conn.Do(ZREVRANGE_COMMAND, key, start, stop))
}

func (o *redisOperator) ZRangeByScore(key string, min interface{}, max interface{}, offset int, count int) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Strings(conn.Do(ZRANGEBYSCORE_COMMAND, key, min, max, "LIMIT", offset, count))
}

func (o *redisOperator) ZRevRangeByScore(key string, max interface{}, min interface{}, offset int, count int) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Strings(conn.Do(ZREVRANGEBYSCORE_COMMAND, key, max, min, "LIMIT", offset, count))
}

func (o *redisOperator) ZRank(key string, member interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(ZRANK_COMMAND, key, member))
}

func (o *redisOperator) ZRevRank(key string, member interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(ZREVRANK_COMMAND, key, member))
}

func (o *redisOperator) ZIncrBy(key string, increment float64, member interface{}) (float64, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Float64(conn.Do(ZINCRBY_COMMAND, key, increment, member))
}
