package cache

var _ Cache = (*PrefixedCache)(nil)

type PrefixedCache struct {
	prefix string
	cache  Cache
}

// List implements Cache.
func (p *PrefixedCache) List(prefix string) ([]string, error) {
	keys, err := p.cache.List(p.prefix + prefix)
	if err != nil {
		return nil, err
	}
	for i, key := range keys {
		keys[i] = key[len(p.prefix):]
	}
	return keys, nil
}

// Delete implements Cache.
func (p *PrefixedCache) Delete(key string) error {
	return p.cache.Delete(p.prefix + key)
}

// Exist implements Cache.
func (p *PrefixedCache) Exist(key string) (bool, error) {
	return p.cache.Exist(p.prefix + key)
}

// Get implements Cache.
func (p *PrefixedCache) Get(key string) (string, error) {
	return p.cache.Get(p.prefix + key)
}

// Set implements Cache.
func (p *PrefixedCache) Set(key string, val string, expire int) error {
	return p.cache.Set(p.prefix+key, val, expire)
}

func WithPrefix(prefix string, cache Cache) Cache {
	return &PrefixedCache{
		prefix: prefix,
		cache:  cache,
	}
}
