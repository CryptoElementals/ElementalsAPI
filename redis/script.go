package redis

import (
	"errors"

	"github.com/CryptoElementals/common/log"
	redigo "github.com/gomodule/redigo/redis"
)

// Script is a Redis Lua script with a fixed KEYS arity (see redigo.Script).
type Script = redigo.Script

// NewScript compiles a Lua script; keyCount is the number of KEYS[] entries.
func NewScript(keyCount int, src string) *Script {
	return redigo.NewScript(keyCount, src)
}

// ScriptDo runs the script using one connection from pool.
func ScriptDo(pool RedisPool, s *Script, keysAndArgs ...interface{}) (interface{}, error) {
	if pool == nil {
		return nil, errors.New("redis pool is nil")
	}
	c := pool.Get()
	defer func() {
		_ = c.Close()
	}()
	return s.Do(c, keysAndArgs...)
}

// ScriptInt runs the script and parses the reply as an integer.
func ScriptInt(pool RedisPool, s *Script, keysAndArgs ...interface{}) (int, error) {
	v, err := ScriptDo(pool, s, keysAndArgs...)
	return redigo.Int(v, err)
}

// ScriptString runs the script and parses the reply as a string.
func ScriptString(pool RedisPool, s *Script, keysAndArgs ...interface{}) (string, error) {
	v, err := ScriptDo(pool, s, keysAndArgs...)
	return redigo.String(v, err)
}

// ScriptValues runs the script and parses the reply as a slice of values.
func ScriptValues(pool RedisPool, s *Script, keysAndArgs ...interface{}) ([]interface{}, error) {
	v, err := ScriptDo(pool, s, keysAndArgs...)
	return redigo.Values(v, err)
}

// ScanReply scans a Redis multi-bulk reply into destinations (see redigo.Scan).
func ScanReply(src []interface{}, dest ...interface{}) ([]interface{}, error) {
	return redigo.Scan(src, dest...)
}

const (
	SCRIPT_LOAD_COMMAND = "SCRIPT"
	EVALSHA_COMMAND     = "EVALSHA"
	EVAL_COMMAND        = "EVAL"
)

func ScriptLoad(script string) (string, error) {
	conn := globalPool.Get()
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redigo.String(conn.Do(SCRIPT_LOAD_COMMAND, "LOAD", script))
}

func EvalSha(sha string, keys []string, args ...interface{}) (interface{}, error) {
	conn := globalPool.Get()
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	cmdArgs := make([]interface{}, 0, 2+len(keys)+len(args))
	cmdArgs = append(cmdArgs, sha, len(keys))
	for _, k := range keys {
		cmdArgs = append(cmdArgs, k)
	}
	cmdArgs = append(cmdArgs, args...)
	return conn.Do(EVALSHA_COMMAND, cmdArgs...)
}

func Eval(script string, keys []string, args ...interface{}) (interface{}, error) {
	conn := globalPool.Get()
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	cmdArgs := make([]interface{}, 0, 2+len(keys)+len(args))
	cmdArgs = append(cmdArgs, script, len(keys))
	for _, k := range keys {
		cmdArgs = append(cmdArgs, k)
	}
	cmdArgs = append(cmdArgs, args...)
	return conn.Do(EVAL_COMMAND, cmdArgs...)
}
