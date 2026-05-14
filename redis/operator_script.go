package redis

import (
	redigo "github.com/gomodule/redigo/redis"
)

func (o *redisOperator) ScriptLoad(script string) (string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redigo.String(conn.Do(SCRIPT_LOAD_COMMAND, "LOAD", script))
}

func (o *redisOperator) EvalSha(sha string, keys []string, args ...interface{}) (interface{}, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	cmdArgs := make([]interface{}, 0, 2+len(keys)+len(args))
	cmdArgs = append(cmdArgs, sha, len(keys))
	for _, k := range keys {
		cmdArgs = append(cmdArgs, k)
	}
	cmdArgs = append(cmdArgs, args...)
	return conn.Do(EVALSHA_COMMAND, cmdArgs...)
}

func (o *redisOperator) Eval(script string, keys []string, args ...interface{}) (interface{}, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	cmdArgs := make([]interface{}, 0, 2+len(keys)+len(args))
	cmdArgs = append(cmdArgs, script, len(keys))
	for _, k := range keys {
		cmdArgs = append(cmdArgs, k)
	}
	cmdArgs = append(cmdArgs, args...)
	return conn.Do(EVAL_COMMAND, cmdArgs...)
}

// ScriptDo runs a compiled script using this operator's pool.
func (o *redisOperator) ScriptDo(s *Script, keysAndArgs ...interface{}) (interface{}, error) {
	return ScriptDo(o.pool, s, keysAndArgs...)
}

// ScriptInt runs the script and parses the reply as an integer.
func (o *redisOperator) ScriptInt(s *Script, keysAndArgs ...interface{}) (int, error) {
	return ScriptInt(o.pool, s, keysAndArgs...)
}

// ScriptString runs the script and parses the reply as a string.
func (o *redisOperator) ScriptString(s *Script, keysAndArgs ...interface{}) (string, error) {
	return ScriptString(o.pool, s, keysAndArgs...)
}

// ScriptValues runs the script and parses the reply as a slice of values.
func (o *redisOperator) ScriptValues(s *Script, keysAndArgs ...interface{}) ([]interface{}, error) {
	return ScriptValues(o.pool, s, keysAndArgs...)
}
