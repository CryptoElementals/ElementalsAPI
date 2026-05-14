package redis

import (
	"errors"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
)

type Config struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	Size     int    `mapstructure:"size"`

	// 过期时间配置（秒）
	SessionExpire int `mapstructure:"session-expire"` // 会话过期时间
}

type RedisConn = redis.Conn

type RedisPool interface {
	Get() RedisConn
}

const (
	PING_COMMAND   = "PING"
	AUTH_COMMAND   = "AUTH"
	GET_COMMAND    = "GET"
	SET_COMMAND    = "SET"
	EXPIRE_COMMAND = "EX"
	EXISTS_COMMAND = "EXISTS"
	DELETE_COMMAND = "DEL"
	SCAN_COMMAND   = "SCAN"
	MATCH_COMMAND  = "match"
	COUNT_COMMAND  = "count"
)

type ConfigWithName struct {
	Name string `mapstructure:"address"`
	Cfg  *Config
}

// Init connects the default Redis pool from defaultCfg, then for each cfgs entry builds and
// pings a separate pool, registers it under Name, and stores Cfg for GetPoolConfig.
// All pings succeed before global state is updated. Re-Init replaces the default pool and
// drops any previously registered named pools not present in this call's cfgs.
func Init(defaultCfg *Config, cfgs ...*ConfigWithName) error {
	if defaultCfg == nil {
		return errors.New("redis: default config is nil")
	}
	seen := make(map[string]struct{}, len(cfgs))
	for _, cn := range cfgs {
		if cn == nil {
			return errors.New("redis: nil ConfigWithName entry")
		}
		if cn.Name == "" {
			return errors.New("redis: named config has empty name")
		}
		if cn.Name == defaultPoolName {
			return fmt.Errorf("redis: config name %q is reserved", defaultPoolName)
		}
		if cn.Cfg == nil {
			return fmt.Errorf("redis: config for name %q is nil", cn.Name)
		}
		if _, dup := seen[cn.Name]; dup {
			return fmt.Errorf("redis: duplicate named config %q", cn.Name)
		}
		seen[cn.Name] = struct{}{}
	}

	defaultPool := newRedigoPool(defaultCfg)
	if _, err := defaultPool.Get().Do(PING_COMMAND); err != nil {
		return err
	}

	type namedReady struct {
		name string
		pool *redis.Pool
		cfg  *Config
	}
	ready := make([]namedReady, 0, len(cfgs))
	for _, cn := range cfgs {
		p := newRedigoPool(cn.Cfg)
		if _, err := p.Get().Do(PING_COMMAND); err != nil {
			return fmt.Errorf("redis: ping named pool %q: %w", cn.Name, err)
		}
		ready = append(ready, namedReady{name: cn.Name, pool: p, cfg: cn.Cfg})
	}

	setDefaultPool(defaultPool, defaultCfg)

	for name := range globalOperatorProvider.ops {
		if name != defaultPoolName {
			delete(globalOperatorProvider.ops, name)
		}
	}
	if globalOperatorProvider.configs == nil {
		globalOperatorProvider.configs = make(map[string]*Config)
	} else {
		clear(globalOperatorProvider.configs)
	}
	for _, nr := range ready {
		if err := registerPool(nr.name, nr.pool); err != nil {
			return err
		}
		globalOperatorProvider.configs[nr.name] = nr.cfg
	}
	return nil
}

func newRedigoPool(cfg *Config) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     cfg.Size,
		IdleTimeout: 240 * time.Second,
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do(PING_COMMAND)
			return err
		},
		Dial: func() (redis.Conn, error) {
			return dial("tcp", cfg.Address, cfg.Password)
		},
	}
}

func GetGlobalPool() RedisPool {
	return mustDefault().Pool()
}

func GetRedigoPool() (*redis.Pool, error) {
	p, ok := mustDefault().Pool().(*redis.Pool)
	if !ok {
		return nil, errors.New("pool is not a valid redisgo redis pool")
	}
	return p, nil
}

func dial(network, address, password string) (redis.Conn, error) {
	c, err := redis.Dial(network, address)
	if err != nil {
		return nil, err
	}
	if password != "" {
		if _, err := c.Do(AUTH_COMMAND, password); err != nil {
			c.Close()
			return nil, err
		}
	}
	return c, err
}

func Get(key string) (string, error) {
	return mustDefault().Get(key)
}

// expire by seconds
func Set(key string, val string, expire int) error {
	return mustDefault().Set(key, val, expire)
}

func Delete(key string) error {
	return mustDefault().Delete(key)
}

func Exist(key string) (bool, error) {
	return mustDefault().Exist(key)
}

// GetConfig returns the default Redis config set by Init (same lifecycle as the default pool).
// Returns nil if Init has not been called.
func GetConfig() *Config {
	return defaultConfig()
}

// GetPoolConfig returns the config for a named pool key. Empty name or "default" returns the
// default config from Init. Other names match pools registered by Init's variadic cfgs (or registerPool).
func GetPoolConfig(name string) (*Config, error) {
	if globalOperatorProvider == nil {
		return nil, errors.New("redis: not initialized")
	}
	if name == "" || name == defaultPoolName {
		if globalOperatorProvider.defaultConfig == nil {
			return nil, errors.New("redis: default config not set")
		}
		return globalOperatorProvider.defaultConfig, nil
	}
	cfg, ok := globalOperatorProvider.configs[name]
	if !ok {
		return nil, fmt.Errorf("redis: unknown config %q", name)
	}
	return cfg, nil
}

func Ping() error {
	return mustDefault().Ping()
}

// 便捷函数：使用会话过期时间设置键值
func SetSession(key string, val string) error {
	cfg := defaultConfig()
	if cfg == nil {
		return errors.New("redis config not initialized")
	}
	return mustDefault().Set(key, val, cfg.SessionExpire)
}

// 获取会话过期时间
func GetSessionExpire() int {
	cfg := defaultConfig()
	if cfg == nil {
		return 43200 // 默认12小时
	}
	return cfg.SessionExpire
}

func Scan(prefix string) ([]string, error) {
	return mustDefault().Scan(prefix)
}
