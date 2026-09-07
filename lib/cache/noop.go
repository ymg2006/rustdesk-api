package cache

import (
	"context"
	"time"
)

// NoopCache disables caching while satisfying the application cache contract.
// Reads are always misses; writes and invalidations are harmless no-ops.
type NoopCache struct{}

func NewNoopCache() *NoopCache {
	return &NoopCache{}
}

func (c *NoopCache) Get(ctx context.Context, key string, dest interface{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrCacheMiss
}

func (c *NoopCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return ctx.Err()
}

func (c *NoopCache) Delete(ctx context.Context, key string) error {
	return ctx.Err()
}
