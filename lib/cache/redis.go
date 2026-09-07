package cache

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"time"
)

type RedisCache struct {
	rdb *redis.Client
}

func RedisCacheInit(conf *redis.Options) *RedisCache {
	return NewRedisWithClient(redis.NewClient(conf))
}

// NewRedisWithClient builds the cache adapter around a caller-owned client.
// The caller remains solely responsible for closing the client.
func NewRedisWithClient(client *redis.Client) *RedisCache {
	return &RedisCache{rdb: client}
}

func (c *RedisCache) Get(ctx context.Context, key string, value interface{}) error {
	data, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCacheMiss
		}
		return err
	}
	return DecodeValue(data, value)
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	str, err := EncodeValue(value)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		ttl = time.Duration(MaxTimeOut) * time.Second
	}
	_, err1 := c.rdb.Set(ctx, key, str, ttl).Result()
	return err1
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

func NewRedis(conf *redis.Options) *RedisCache {
	cache := RedisCacheInit(conf)
	return cache
}
