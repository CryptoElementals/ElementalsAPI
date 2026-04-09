package redis

import (
	"github.com/CryptoElementals/common/log"
	redigo "github.com/gomodule/redigo/redis"
)

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
