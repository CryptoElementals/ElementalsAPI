package redis

import (
	redigo "github.com/gomodule/redigo/redis"
)

func (o *RedisOperator) SAdd(key string, members ...interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(members)+1)
	args = append(args, key)
	args = append(args, members...)
	return redigo.Int(conn.Do(SADD_COMMAND, args...))
}

func (o *RedisOperator) SRem(key string, members ...interface{}) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(members)+1)
	args = append(args, key)
	args = append(args, members...)
	return redigo.Int(conn.Do(SREM_COMMAND, args...))
}

func (o *RedisOperator) SMove(source, destination string, member interface{}) (bool, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redigo.Bool(conn.Do(SMOVE_COMMAND, source, destination, member))
}

func (o *RedisOperator) SMembers(key string) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redigo.Strings(conn.Do(SMEMBERS_COMMAND, key))
}

func (o *RedisOperator) SIsMember(key string, member interface{}) (bool, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redigo.Bool(conn.Do(SISMEMBER_CMD, key, member))
}

func (o *RedisOperator) SCard(key string) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redigo.Int(conn.Do(SCARD_COMMAND, key))
}

func (o *RedisOperator) SPop(key string) (string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redigo.String(conn.Do(SPOP_COMMAND, key))
}
