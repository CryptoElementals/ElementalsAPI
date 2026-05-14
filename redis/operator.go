package redis

import (
	"errors"
	"fmt"

	"github.com/CryptoElementals/common/log"
	"github.com/gomodule/redigo/redis"
)

const defaultPoolName = "default"

// redisOperator runs Redis commands against a single connection pool.
type redisOperator struct {
	pool RedisPool
}

// newOperator wraps a pool for command execution. Pool must not be nil.
func newOperator(pool RedisPool) *redisOperator {
	if pool == nil {
		panic("redis: NewOperator with nil pool")
	}
	return &redisOperator{pool: pool}
}

// Pool returns the backing Redis pool.
func (o *redisOperator) Pool() RedisPool {
	if o == nil {
		return nil
	}
	return o.pool
}

func (o *redisOperator) closeConn(c RedisConn) {
	if err := c.Close(); err != nil {
		log.Errorf("redis client close err: %s", err.Error())
	}
}

type operatorProvider struct {
	ops           map[string]*redisOperator
	defaultOp     *redisOperator
	defaultConfig *Config // default pool config (set by Init)
	configs       map[string]*Config
}

var globalOperatorProvider *operatorProvider

func mustDefault() *redisOperator {
	if globalOperatorProvider == nil || globalOperatorProvider.defaultOp == nil {
		panic("redis: Init was not called (default operator is nil)")
	}
	return globalOperatorProvider.defaultOp
}

// DefaultOperator returns the default pool's operator, or an error if Redis is not initialized.
func DefaultOperator() (*redisOperator, error) {
	if globalOperatorProvider == nil || globalOperatorProvider.defaultOp == nil {
		return nil, errors.New("redis: not initialized")
	}
	return globalOperatorProvider.defaultOp, nil
}

// registerPool registers an additional named pool. Name must be non-empty and not "default".
// Duplicate names return an error. Not synchronized: finish all registerPool calls before
// concurrent use of Pool or package-level redis funcs.
func registerPool(name string, pool RedisPool) error {
	if name == "" {
		return errors.New("redis: pool name is empty")
	}
	if name == defaultPoolName {
		return fmt.Errorf("redis: pool name %q is reserved", defaultPoolName)
	}
	if pool == nil {
		return errors.New("redis: pool is nil")
	}
	if globalOperatorProvider == nil {
		return errors.New("redis: call Init before registerPool")
	}
	if _, exists := globalOperatorProvider.ops[name]; exists {
		return fmt.Errorf("redis: pool %q already registered", name)
	}
	globalOperatorProvider.ops[name] = newOperator(pool)
	return nil
}

// Pool returns the operator for a named pool. Use defaultPoolName or empty string for the default operator.
func Pool(name string) (*redisOperator, error) {
	if name == "" || name == defaultPoolName {
		return DefaultOperator()
	}
	if globalOperatorProvider == nil {
		return nil, errors.New("redis: not initialized")
	}
	op, ok := globalOperatorProvider.ops[name]
	if !ok {
		return nil, fmt.Errorf("redis: unknown pool %q", name)
	}
	return op, nil
}

func setDefaultPool(pool RedisPool, cfg *Config) {
	if globalOperatorProvider == nil {
		globalOperatorProvider = &operatorProvider{
			ops:     make(map[string]*redisOperator),
			configs: make(map[string]*Config),
		}
	}
	op := newOperator(pool)
	globalOperatorProvider.defaultOp = op
	globalOperatorProvider.ops[defaultPoolName] = op
	if cfg != nil {
		globalOperatorProvider.defaultConfig = cfg
	}
}

func defaultConfig() *Config {
	if globalOperatorProvider == nil {
		return nil
	}
	return globalOperatorProvider.defaultConfig
}

// --- KV (see operator_kv.go for bodies if split; kept here for small surface) ---

func (o *redisOperator) Get(key string) (string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.String(conn.Do(GET_COMMAND, key))
}

func (o *redisOperator) Set(key string, val string, expire int) error {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	if expire <= 0 {
		_, err := conn.Do(SET_COMMAND, key, val)
		return err
	}
	_, err := conn.Do(SET_COMMAND, key, val, EXPIRE_COMMAND, expire)
	return err
}

func (o *redisOperator) Delete(key string) error {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	_, err := conn.Do(DELETE_COMMAND, key)
	return err
}

func (o *redisOperator) Exist(key string) (bool, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	return redis.Bool(conn.Do(EXISTS_COMMAND, key))
}

func (o *redisOperator) Ping() error {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	_, err := conn.Do(PING_COMMAND)
	return err
}

func (o *redisOperator) Scan(prefix string) ([]string, error) {
	conn := o.pool.Get()
	defer o.closeConn(conn)
	var keys []string
	cursor := 0
	for {
		var scanCursor int
		var scanKeys []string
		res, err := redis.Values(conn.Do(SCAN_COMMAND, cursor, MATCH_COMMAND, prefix+"*", COUNT_COMMAND, 10000))
		if err != nil {
			return nil, err
		}
		rest, err := redis.Scan(res, &scanCursor, &scanKeys)
		if err != nil {
			return nil, err
		}
		if len(rest) != 0 {
			return nil, errors.New("scan error: unexpected result number")
		}
		if len(scanKeys) != 0 {
			keys = append(keys, scanKeys...)
		}
		cursor = scanCursor
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}
