package redisstore

import (
	"time"

	"github.com/go-redis/redis"
)

// RedisStore uses Redis
type RedisStore struct {
	Client *redis.Client
}

// NewRedisStore creates a new RedisStore
func NewRedisStore(addr string) RedisStore {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	rs := RedisStore{
		Client: rdb,
	}

	return rs
}

// Get value from Redis.
func (r RedisStore) Get(key string) (interface{}, error) {
	cmd := r.Client.Get(key)
	return cmd.Bytes()
}

// Has checks if the key exists.
func (r RedisStore) Has(key string) bool {
	cmd := r.Client.Get(key)
	bytes, err := cmd.Bytes()
	if err != nil {
		return false
	}
	if len(bytes) > 0 {
		return true
	}

	return false
}

// Put the value in Redis.
func (r RedisStore) Put(key string, value interface{}, expiration time.Duration) {
	r.Client.Set(key, value, expiration)
}
