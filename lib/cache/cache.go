package cache

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"
)

var ErrCacheMiss = errors.New("cache: key not found")

type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// Handler is retained as an alias for source compatibility with older callers.
type Handler = Cache

// MaxTimeOut maximum timeout time

const (
	TypeMem    = "memory"
	TypeRedis  = "redis"
	TypeFile   = "file"
	TypeNone   = "none"
	MaxTimeOut = 365 * 24 * 3600
)

func New(typ string) Cache {
	var cache Cache
	switch typ {
	case TypeFile:
		cache = NewFileCache()
	case TypeRedis:
		// Redis requires connection options or a configured client. The generic
		// constructor must remain safe, so callers use NewRedis instead.
		cache = NewNoopCache()
	case TypeMem: // memory
		cache = NewMemoryCache(0)
	case TypeNone:
		cache = NewNoopCache()
	default:
		cache = NewNoopCache()
	}
	return cache
}

func EncodeValue(value interface{}) (string, error) {
	/*if v, ok := value.(string); ok {
		return v, nil
	}
	if v, ok := value.([]byte); ok {
		return string(v), nil
	}*/
	b, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DecodeValue(value string, rtv interface{}) error {
	dest := reflect.ValueOf(rtv)
	if dest.Kind() != reflect.Ptr || dest.IsNil() {
		return errors.New("cache: destination must be a non-nil pointer")
	}
	// Decode atomically so malformed cache data cannot partially mutate a
	// destination that the caller will reuse for its database fallback.
	tmp := reflect.New(dest.Elem().Type())
	if err := json.Unmarshal([]byte(value), tmp.Interface()); err != nil {
		return err
	}
	dest.Elem().Set(tmp.Elem())
	return nil
}
