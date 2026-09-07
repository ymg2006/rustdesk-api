package cache

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"
)

// A simple cache is implemented here for testing
// SimpleCache is a simple cache implementation
type SimpleCache struct {
	data      map[string]interface{}
	mu        sync.Mutex
	maxBytes  int64
	usedBytes int64
}

type simpleItem struct {
	value     interface{}
	expiresAt time.Time
}

func (s *SimpleCache) Get(ctx context.Context, key string, value interface{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use reflection to set the stored value into the passed pointer variable
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Ptr {
		return errors.New("value must be a pointer")
	}
	v, ok := s.data[key]
	if !ok {
		//set to null
		val.Elem().Set(reflect.Zero(val.Elem().Type()))
		return ErrCacheMiss
	}
	item := v.(simpleItem)
	if time.Now().After(item.expiresAt) {
		delete(s.data, key)
		val.Elem().Set(reflect.Zero(val.Elem().Type()))
		return ErrCacheMiss
	}

	vval := reflect.ValueOf(item.value)
	if val.Elem().Type() != vval.Type() {
		//set to null
		val.Elem().Set(reflect.Zero(val.Elem().Type()))
		return ErrCacheMiss
	}

	val.Elem().Set(vval)
	return nil
}

func (s *SimpleCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Check if the passed in value is a pointer, if so take its value
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if ttl <= 0 {
		ttl = time.Duration(MaxTimeOut) * time.Second
	}
	s.data[key] = simpleItem{value: val.Interface(), expiresAt: time.Now().Add(ttl)}
	return nil
}

func (s *SimpleCache) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func NewSimpleCache() *SimpleCache {
	return &SimpleCache{
		data: make(map[string]interface{}),
	}
}
