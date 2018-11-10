package cache

import (
	"github.com/kyokan/chaind/pkg"
	"time"
)

type CacheableMap map[string][]byte

type Cacher interface {
	pkg.Service
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	SetEx(key string, value []byte, expiration time.Duration) error
	Has(key string) (bool, error)
	MapGet(key string, field string) ([]byte, error)
	MapSetEx(key string, vals CacheableMap, expiration time.Duration) error
	Del(key string) error
}