package cache

import (
	"errors"
)

var ErrNotFound = errors.New("key not found")

type Cache interface {
	Get(key string) (string, error)
	Set(key string, val string, expire int) error
	Exist(key string) (bool, error)
	Delete(key string) error
	List(prefix string) ([]string, error)
}
