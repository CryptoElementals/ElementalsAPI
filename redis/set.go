package redis

import (
	"github.com/CryptoElementals/common/log"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	SADD_COMMAND     = "SADD"
	SREM_COMMAND     = "SREM"
	SMOVE_COMMAND    = "SMOVE"
	SMEMBERS_COMMAND = "SMEMBERS"
	SISMEMBER_CMD    = "SISMEMBER"
	SCARD_COMMAND    = "SCARD"
	SPOP_COMMAND     = "SPOP"
)

func SAdd(key string, members ...interface{}) (int, error) {
	conn := globalPool.Get()
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	args := make([]interface{}, 0, len(members)+1)
	args = append(args, key)
	args = append(args, members...)
	return redigo.Int(conn.Do(SADD_COMMAND, args...))
}

func SRem(key string, members ...interface{}) (int, error) {
	conn := globalPool.Get()
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	args := make([]interface{}, 0, len(members)+1)
	args = append(args, key)
	args = append(args, members...)
	return redigo.Int(conn.Do(SREM_COMMAND, args...))
}

func SMove(source, destination string, member interface{}) (bool, error) {
	conn := globalPool.Get()
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redigo.Bool(conn.Do(SMOVE_COMMAND, source, destination, member))
}

func SMembers(key string) ([]string, error) {
	conn := globalPool.Get()
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redigo.Strings(conn.Do(SMEMBERS_COMMAND, key))
}

func SIsMember(key string, member interface{}) (bool, error) {
	conn := globalPool.Get()
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redigo.Bool(conn.Do(SISMEMBER_CMD, key, member))
}

func SCard(key string) (int, error) {
	conn := globalPool.Get()
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redigo.Int(conn.Do(SCARD_COMMAND, key))
}

func SPop(key string) (string, error) {
	conn := globalPool.Get()
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("redis client close err: %s", err.Error())
		}
	}()
	return redigo.String(conn.Do(SPOP_COMMAND, key))
}
