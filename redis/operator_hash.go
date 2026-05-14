package redis

import (
	"github.com/gomodule/redigo/redis"
)

func (o *redisOperator) HSet(key string, field string, value interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(HSET_COMMAND, key, field, value))
}

func (o *redisOperator) HMSet(key string, values map[string]interface{}) error {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(values)*2+1)
	args = append(args, key)
	for field, value := range values {
		args = append(args, field, value)
	}
	_, err := conn.Do(HMSET_COMMAND, args...)
	return err
}

func (o *redisOperator) HGet(key string, field string) (string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.String(conn.Do(HGET_COMMAND, key, field))
}

func (o *redisOperator) HMGet(key string, fields ...string) ([]interface{}, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(fields)+1)
	args = append(args, key)
	for _, field := range fields {
		args = append(args, field)
	}
	return redis.Values(conn.Do(HMGET_COMMAND, args...))
}

func (o *redisOperator) HGetAll(key string) (map[string]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.StringMap(conn.Do(HGETALL_COMMAND, key))
}

func (o *redisOperator) HDel(key string, fields ...string) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(fields)+1)
	args = append(args, key)
	for _, field := range fields {
		args = append(args, field)
	}
	return redis.Int(conn.Do(HDEL_COMMAND, args...))
}

func (o *redisOperator) HExists(key string, field string) (bool, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Bool(conn.Do(HEXISTS_COMMAND, key, field))
}

func (o *redisOperator) HLen(key string) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(HLEN_COMMAND, key))
}

func (o *redisOperator) HKeys(key string) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Strings(conn.Do(HKEYS_COMMAND, key))
}

func (o *redisOperator) HVals(key string) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Strings(conn.Do(HVALS_COMMAND, key))
}

func (o *redisOperator) HIncrBy(key string, field string, increment int64) (int64, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int64(conn.Do(HINCRBY_COMMAND, key, field, increment))
}
