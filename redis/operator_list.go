package redis

import (
	"github.com/gomodule/redigo/redis"
)

func (o *redisOperator) LPush(key string, values ...interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(values)+1)
	args = append(args, key)
	args = append(args, values...)
	return redis.Int(conn.Do(LPUSH_COMMAND, args...))
}

func (o *redisOperator) RPush(key string, values ...interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(values)+1)
	args = append(args, key)
	args = append(args, values...)
	return redis.Int(conn.Do(RPUSH_COMMAND, args...))
}

func (o *redisOperator) LPop(key string) (string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.String(conn.Do(LPOP_COMMAND, key))
}

func (o *redisOperator) RPop(key string) (string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.String(conn.Do(RPOP_COMMAND, key))
}

func (o *redisOperator) LLen(key string) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(LLEN_COMMAND, key))
}

func (o *redisOperator) LRange(key string, start int, stop int) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Strings(conn.Do(LRANGE_COMMAND, key, start, stop))
}

func (o *redisOperator) LRem(key string, count int, value interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(LREM_COMMAND, key, count, value))
}

func (o *redisOperator) LSet(key string, index int, value interface{}) error {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	_, err := conn.Do(LSET_COMMAND, key, index, value)
	return err
}

func (o *redisOperator) LIndex(key string, index int) (string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.String(conn.Do(LINDEX_COMMAND, key, index))
}

func (o *redisOperator) LTrim(key string, start int, stop int) error {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	_, err := conn.Do(LTRIM_COMMAND, key, start, stop)
	return err
}
