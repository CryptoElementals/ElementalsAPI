package redis

import (
	"github.com/gomodule/redigo/redis"
)

func (o *RedisOperator) XAdd(stream string, id string, fields map[string]interface{}) (string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(fields)*2+2)
	args = append(args, stream, id)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return redis.String(conn.Do(XADD_COMMAND, args...))
}

func (o *RedisOperator) XDel(stream string, ids ...string) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, stream)
	for _, id := range ids {
		args = append(args, id)
	}
	return redis.Int(conn.Do(XDEL_COMMAND, args...))
}

func (o *RedisOperator) XLen(stream string) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(XLEN_COMMAND, stream))
}

func (o *RedisOperator) XRange(stream string, start string, end string, count int) ([]interface{}, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := []interface{}{stream, start, end}
	if count > 0 {
		args = append(args, COUNT_OPTION, count)
	}
	return redis.Values(conn.Do(XRANGE_COMMAND, args...))
}

func (o *RedisOperator) XRevRange(stream string, end string, start string, count int) ([]interface{}, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := []interface{}{stream, end, start}
	if count > 0 {
		args = append(args, COUNT_OPTION, count)
	}
	return redis.Values(conn.Do(XREVRANGE_COMMAND, args...))
}

func (o *RedisOperator) XRead(stream string, startID string, count int, blockMs int) ([]interface{}, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, 8)
	if blockMs > 0 {
		args = append(args, BLOCK_OPTION, blockMs)
	}
	if count > 0 {
		args = append(args, COUNT_OPTION, count)
	}
	args = append(args, STREAMS_OPTION, stream, startID)
	return redis.Values(conn.Do(XREAD_COMMAND, args...))
}

func (o *RedisOperator) XTrimMaxLen(stream string, maxLen int, approximate bool) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := []interface{}{stream, MAXLEN_OPTION}
	if approximate {
		args = append(args, APPROX_OPTION)
	}
	args = append(args, maxLen)
	return redis.Int(conn.Do(XTRIM_COMMAND, args...))
}

func (o *RedisOperator) XTrimMinID(stream string, minID string) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(XTRIM_COMMAND, stream, MINID_OPTION, minID))
}

func (o *RedisOperator) XGroupCreate(stream string, group string, startID string, mkstream bool) error {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := []interface{}{XGROUP_CREATE_SUBCOMMAND, stream, group, startID}
	if mkstream {
		args = append(args, MKSTREAM_OPTION)
	}
	_, err := conn.Do(XGROUP_COMMAND, args...)
	return err
}

func (o *RedisOperator) XGroupDestroy(stream string, group string) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(XGROUP_COMMAND, XGROUP_DESTROY_SUBCOMMAND, stream, group))
}

func (o *RedisOperator) XGroupDelConsumer(stream string, group string, consumer string) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Int(conn.Do(XGROUP_COMMAND, XGROUP_DELCONSUMER_SUBCOMMAND, stream, group, consumer))
}

func (o *RedisOperator) XReadGroup(group string, consumer string, stream string, id string, count int, blockMs int) ([]interface{}, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := []interface{}{"GROUP", group, consumer}
	if blockMs > 0 {
		args = append(args, BLOCK_OPTION, blockMs)
	}
	if count > 0 {
		args = append(args, COUNT_OPTION, count)
	}
	args = append(args, STREAMS_OPTION, stream, id)
	return redis.Values(conn.Do(XREADGROUP_COMMAND, args...))
}

func (o *RedisOperator) XAck(stream string, group string, ids ...string) (int, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(ids)+2)
	args = append(args, stream, group)
	for _, id := range ids {
		args = append(args, id)
	}
	return redis.Int(conn.Do(XACK_COMMAND, args...))
}

func (o *RedisOperator) XPending(stream string, group string) ([]interface{}, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Values(conn.Do(XPENDING_COMMAND, stream, group))
}

func (o *RedisOperator) XClaim(stream string, group string, consumer string, minIdleTimeMs int, ids ...string) ([]interface{}, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := make([]interface{}, 0, len(ids)+4)
	args = append(args, stream, group, consumer, minIdleTimeMs)
	for _, id := range ids {
		args = append(args, id)
	}
	return redis.Values(conn.Do(XCLAIM_COMMAND, args...))
}

func (o *RedisOperator) XAutoClaim(stream string, group string, consumer string, minIdleTimeMs int, start string, count int) ([]interface{}, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	args := []interface{}{stream, group, consumer, minIdleTimeMs, start}
	if count > 0 {
		args = append(args, COUNT_OPTION, count)
	}
	return redis.Values(conn.Do(XAUTOCLAIM_COMMAND, args...))
}

func (o *RedisOperator) XInfoStream(stream string) ([]interface{}, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Values(conn.Do(XINFO_COMMAND, "STREAM", stream))
}
