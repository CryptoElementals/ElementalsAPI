package redis

import (
	"github.com/CryptoElementals/common/log"
	"github.com/gomodule/redigo/redis"
)

const (
	XADD_COMMAND                  = "XADD"
	XDEL_COMMAND                  = "XDEL"
	XLEN_COMMAND                  = "XLEN"
	XRANGE_COMMAND                = "XRANGE"
	XREVRANGE_COMMAND             = "XREVRANGE"
	XREAD_COMMAND                 = "XREAD"
	XTRIM_COMMAND                 = "XTRIM"
	XGROUP_COMMAND                = "XGROUP"
	XREADGROUP_COMMAND            = "XREADGROUP"
	XACK_COMMAND                  = "XACK"
	XPENDING_COMMAND              = "XPENDING"
	XCLAIM_COMMAND                = "XCLAIM"
	XAUTOCLAIM_COMMAND            = "XAUTOCLAIM"
	XINFO_COMMAND                 = "XINFO"
	XGROUP_CREATE_SUBCOMMAND      = "CREATE"
	XGROUP_DESTROY_SUBCOMMAND     = "DESTROY"
	XGROUP_DELCONSUMER_SUBCOMMAND = "DELCONSUMER"
	MKSTREAM_OPTION               = "MKSTREAM"
	STREAMS_OPTION                = "STREAMS"
	BLOCK_OPTION                  = "BLOCK"
	COUNT_OPTION                  = "COUNT"
	MINID_OPTION                  = "MINID"
	MAXLEN_OPTION                 = "MAXLEN"
	APPROX_OPTION                 = "~"
)

func XAdd(stream string, id string, fields map[string]interface{}) (string, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := make([]interface{}, 0, len(fields)*2+2)
	args = append(args, stream, id)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return redis.String(conn.Do(XADD_COMMAND, args...))
}

func XDel(stream string, ids ...string) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, stream)
	for _, id := range ids {
		args = append(args, id)
	}
	return redis.Int(conn.Do(XDEL_COMMAND, args...))
}

func XLen(stream string) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int(conn.Do(XLEN_COMMAND, stream))
}

func XRange(stream string, start string, end string, count int) ([]interface{}, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := []interface{}{stream, start, end}
	if count > 0 {
		args = append(args, COUNT_OPTION, count)
	}
	return redis.Values(conn.Do(XRANGE_COMMAND, args...))
}

func XRevRange(stream string, end string, start string, count int) ([]interface{}, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := []interface{}{stream, end, start}
	if count > 0 {
		args = append(args, COUNT_OPTION, count)
	}
	return redis.Values(conn.Do(XREVRANGE_COMMAND, args...))
}

func XRead(stream string, startID string, count int, blockMs int) ([]interface{}, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

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

func XTrimMaxLen(stream string, maxLen int, approximate bool) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := []interface{}{stream, MAXLEN_OPTION}
	if approximate {
		args = append(args, APPROX_OPTION)
	}
	args = append(args, maxLen)
	return redis.Int(conn.Do(XTRIM_COMMAND, args...))
}

func XTrimMinID(stream string, minID string) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int(conn.Do(XTRIM_COMMAND, stream, MINID_OPTION, minID))
}

func XGroupCreate(stream string, group string, startID string, mkstream bool) error {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := []interface{}{XGROUP_CREATE_SUBCOMMAND, stream, group, startID}
	if mkstream {
		args = append(args, MKSTREAM_OPTION)
	}
	_, err := conn.Do(XGROUP_COMMAND, args...)
	return err
}

func XGroupDestroy(stream string, group string) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int(conn.Do(XGROUP_COMMAND, XGROUP_DESTROY_SUBCOMMAND, stream, group))
}

func XGroupDelConsumer(stream string, group string, consumer string) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Int(conn.Do(XGROUP_COMMAND, XGROUP_DELCONSUMER_SUBCOMMAND, stream, group, consumer))
}

func XReadGroup(group string, consumer string, stream string, id string, count int, blockMs int) ([]interface{}, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

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

func XAck(stream string, group string, ids ...string) (int, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := make([]interface{}, 0, len(ids)+2)
	args = append(args, stream, group)
	for _, id := range ids {
		args = append(args, id)
	}
	return redis.Int(conn.Do(XACK_COMMAND, args...))
}

func XPending(stream string, group string) ([]interface{}, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Values(conn.Do(XPENDING_COMMAND, stream, group))
}

func XClaim(stream string, group string, consumer string, minIdleTimeMs int, ids ...string) ([]interface{}, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := make([]interface{}, 0, len(ids)+4)
	args = append(args, stream, group, consumer, minIdleTimeMs)
	for _, id := range ids {
		args = append(args, id)
	}
	return redis.Values(conn.Do(XCLAIM_COMMAND, args...))
}

// XAutoClaim runs XAUTOCLAIM key group consumer min-idle-time start [COUNT count].
// Returns the raw Redis array: [next-start, entries...] (entries in XRANGE shape).
func XAutoClaim(stream string, group string, consumer string, minIdleTimeMs int, start string, count int) ([]interface{}, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()

	args := []interface{}{stream, group, consumer, minIdleTimeMs, start}
	if count > 0 {
		args = append(args, COUNT_OPTION, count)
	}
	return redis.Values(conn.Do(XAUTOCLAIM_COMMAND, args...))
}

func XInfoStream(stream string) ([]interface{}, error) {
	conn := globalPool.Get()
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redis.Values(conn.Do(XINFO_COMMAND, "STREAM", stream))
}
