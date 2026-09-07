package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryCacheMissDeleteAndContext(t *testing.T) {
	c := NewMemoryCache(0)
	t.Cleanup(func() { _ = c.Close() })

	var value string
	if err := c.Get(context.Background(), "missing", &value); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("missing Get error = %v, want ErrCacheMiss", err)
	}
	if err := c.Set(context.Background(), "key", "value", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(context.Background(), "key"); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(context.Background(), "key", &value); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("deleted Get error = %v, want ErrCacheMiss", err)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Set(cancelled, "key", "value", time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Set error = %v, want context.Canceled", err)
	}
}

func TestFileCacheMissAndDelete(t *testing.T) {
	c := NewFileCache()
	c.SetDir(t.TempDir())
	ctx := context.Background()

	var value string
	if err := c.Get(ctx, "missing", &value); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("missing Get error = %v, want ErrCacheMiss", err)
	}
	if err := c.Set(ctx, "key", "value", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(ctx, "key"); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(ctx, "key", &value); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("deleted Get error = %v, want ErrCacheMiss", err)
	}
}

func TestNoopCacheAlwaysMisses(t *testing.T) {
	c := NewNoopCache()
	ctx := context.Background()
	var value string
	if err := c.Get(ctx, "key", &value); !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("Get error = %v, want ErrCacheMiss", err)
	}
	if err := c.Set(ctx, "key", "value", time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(ctx, "key"); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeValueInvalidJSONDoesNotMutateDestination(t *testing.T) {
	type item struct {
		Name string `json:"name"`
	}
	dest := []item{{Name: "database-placeholder"}}
	if err := DecodeValue(`[{"name":"cached"},{"name":`, &dest); err == nil {
		t.Fatal("invalid JSON unexpectedly decoded")
	}
	if len(dest) != 1 || dest[0].Name != "database-placeholder" {
		t.Fatalf("destination was partially mutated: %#v", dest)
	}
}
