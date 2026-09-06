package cache

import (
	"github.com/go-redis/redis/v8"
	"testing"
)

func TestNewRedisWithClientReusesConfiguredClient(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:     "dragonfly:6379",
		Password: "test-password",
		DB:       1,
	})
	t.Cleanup(func() { _ = client.Close() })

	cache := NewRedisWithClient(client)
	if cache.rdb != client {
		t.Fatal("Redis cache did not retain the shared client")
	}
	if got := cache.rdb.Options().DB; got != 1 {
		t.Fatalf("Redis logical database = %d, want 1", got)
	}
}
