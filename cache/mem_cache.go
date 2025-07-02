package cache

import (
	"sync"
	"time"
)

type entry struct {
	value  string
	expire time.Time
}

// memcache for test
type MemCache struct {
	sync.Mutex
	c map[string]*entry
}

func NewMemCache() Cache {
	return &MemCache{
		c: make(map[string]*entry),
	}
}

func (m *MemCache) Get(key string) (string, error) {
	m.Lock()
	defer m.Unlock()
	entry, ok := m.c[key]
	if !ok {
		return "", ErrNotFound
	}
	if entry.expire.Before(time.Now()) {
		delete(m.c, key)
		return "", ErrNotFound
	}
	return entry.value, nil
}
func (m *MemCache) Set(key string, val string, expire int) error {
	m.Lock()
	defer m.Unlock()
	expireAt := time.Now().Add(time.Duration(expire) * time.Second)
	ent := &entry{
		value:  val,
		expire: expireAt,
	}
	m.c[key] = ent
	return nil
}
func (m *MemCache) Exist(key string) (bool, error) {
	m.Lock()
	defer m.Unlock()
	entry, ok := m.c[key]
	if !ok {
		return false, ErrNotFound
	}
	if entry.expire.Before(time.Now()) {
		delete(m.c, key)
		return false, ErrNotFound
	}
	return true, nil
}
func (m *MemCache) Delete(key string) error {
	m.Lock()
	defer m.Unlock()
	delete(m.c, key)
	return nil
}
