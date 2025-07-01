package session

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	redigo_redis "github.com/gomodule/redigo/redis"
)

func New(pool *redigo_redis.Pool) (sessions.Store, error) {
	// 初始化基于redis的存储引擎
	// 参数说明：
	//    第1个参数 - redis pool
	//    第2个参数 - session加密密钥
	s, err := redis.NewStoreWithPool(pool, []byte("secret"))
	if err != nil {
		return nil, err
	}

	return s, nil
}
