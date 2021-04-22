package stores

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisStore uses Redis
type RedisStore struct {
	Client *redis.Client
}

// NewRedisStore creates a new RedisStore
func NewRedisStore(addr string) RedisStore {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
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
	ctx := context.TODO()
	cmd := r.Client.Get(ctx, key)
	return cmd.Bytes()
}

// Has checks if the key exists.
func (r RedisStore) Has(key string) bool {
	ctx := context.TODO()
	cmd := r.Client.Get(ctx, key)
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
	ctx := context.TODO()
	r.Client.Set(ctx, key, value, expiration)
}
