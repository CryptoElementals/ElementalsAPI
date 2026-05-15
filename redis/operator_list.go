package redis

import (
	"github.com/gomodule/redigo/redis"
)

func (o *RedisOperator) LPush(key string, values ...interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(values)+1)
	args = append(args, key)
	args = append(args, values...)
	return redis.Int(conn.Do(LPUSH_COMMAND, args...))
}

func (o *RedisOperator) RPush(key string, values ...interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(values)+1)
	args = append(args, key)
	args = append(args, values...)
	return redis.Int(conn.Do(RPUSH_COMMAND, args...))
}

func (o *RedisOperator) LPop(key string) (string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.String(conn.Do(LPOP_COMMAND, key))
}

func (o *RedisOperator) RPop(key string) (string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.String(conn.Do(RPOP_COMMAND, key))
}

func (o *RedisOperator) LLen(key string) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(LLEN_COMMAND, key))
}

func (o *RedisOperator) LRange(key string, start int, stop int) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Strings(conn.Do(LRANGE_COMMAND, key, start, stop))
}

func (o *RedisOperator) LRem(key string, count int, value interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(LREM_COMMAND, key, count, value))
}

func (o *RedisOperator) LSet(key string, index int, value interface{}) error {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	_, err := conn.Do(LSET_COMMAND, key, index, value)
	return err
}

func (o *RedisOperator) LIndex(key string, index int) (string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.String(conn.Do(LINDEX_COMMAND, key, index))
}

func (o *RedisOperator) LTrim(key string, start int, stop int) error {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	_, err := conn.Do(LTRIM_COMMAND, key, start, stop)
	return err
}
