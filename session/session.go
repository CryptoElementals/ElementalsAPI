package session

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	redigo_redis "github.com/gomodule/redigo/redis"
)

type SessionStore struct {
	redis.Store
	maxAge int
}

func (s *SessionStore) MaxAge() int {
	return s.maxAge
}

var store *SessionStore

func Init(pool *redigo_redis.Pool, maxAge int) error {

	// 初始化基于redis的存储引擎
	// 参数说明：
	//    第1个参数 - redis pool
	//    第2个参数 - session加密密钥
	s, err := redis.NewStoreWithPool(pool, []byte("secret"))
	if err != nil {
		return err
	}
	s.Options(sessions.Options{
		MaxAge: maxAge, // 设置过期时间
	})

	store = &SessionStore{
		Store:  s,
		maxAge: maxAge,
	}
	return nil
}

func Get() *SessionStore {
	return store
}
