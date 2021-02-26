package stores

import "time"

// CacheStore represents a way to cache
type CacheStore interface {
	Get(key string) (interface{}, error)
	Has(key string) bool
	Put(key string, value interface{}, expiration time.Duration)
}
