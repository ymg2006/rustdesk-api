package cache

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type FileCache struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
	Dir   string
}

func (fc *FileCache) getLock(key string) *sync.Mutex {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	if fc.locks == nil {
		fc.locks = make(map[string]*sync.Mutex)
	}
	if _, ok := fc.locks[key]; !ok {
		fc.locks[key] = new(sync.Mutex)
	}
	return fc.locks[key]
}

// fileItem is the internal encapsulation of the downloaded file, which writes the expiration timestamp into the file content.
// Avoid relying on file mtime (future time is fragile and can easily be reset by backup/synchronization/antivirus software, causing the cache to be permanently missed)
type fileItem struct {
	Exp  int64  `json:"exp"`  // Expiration unix timestamp; <=0 means use MaxTimeOut
	Data string `json:"data"` // JSON string after EncodeValue
}

func (c *FileCache) fileName(key string) string {
	f := c.Dir + string(os.PathSeparator) + fmt.Sprintf("%x", md5.Sum([]byte(key)))
	return f
}

// getValue reads the disk value and reports missing, expired, or damaged
// entries as cache misses.
func (c *FileCache) getValue(key string) (string, error) {
	f := c.fileName(key)
	lock := c.getLock(f)
	lock.Lock()
	defer lock.Unlock()

	data, err := os.ReadFile(f)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrCacheMiss
		}
		return "", err
	}
	var item fileItem
	if err := json.Unmarshal(data, &item); err != nil {
		// The file is damaged (including old format) and will be considered a miss after deletion.
		os.Remove(f)
		return "", ErrCacheMiss
	}
	if item.Exp > 0 && time.Now().Unix() >= item.Exp {
		os.Remove(f)
		return "", ErrCacheMiss
	}
	return item.Data, nil
}

// Get reads and decodes a cache entry.
func (c *FileCache) Get(ctx context.Context, key string, value interface{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := c.getValue(key)
	if err != nil {
		return err
	}
	return DecodeValue(data, value)
}

func (c *FileCache) saveValue(key string, value string, ttl time.Duration) error {
	f := c.fileName(key)
	lock := c.getLock(f)
	lock.Lock()
	defer lock.Unlock()

	if ttl <= 0 {
		ttl = time.Duration(MaxTimeOut) * time.Second
	}
	item := fileItem{
		Exp:  time.Now().Add(ttl).Unix(),
		Data: value,
	}
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return os.WriteFile(f, b, 0644)
}

func (c *FileCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	str, err := EncodeValue(value)
	if err != nil {
		return err
	}
	return c.saveValue(key, str, ttl)
}

func (c *FileCache) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f := c.fileName(key)
	lock := c.getLock(f)
	lock.Lock()
	defer lock.Unlock()
	if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (c *FileCache) SetDir(path string) {
	c.Dir = path
}

func (c *FileCache) Gc() error {
	// Expiration cleanup is completed by Get lazy deletion; if you need active recycling, you can traverse Dir here
	return nil
}

func NewFileCache() *FileCache {
	return &FileCache{
		locks: make(map[string]*sync.Mutex),
		Dir:   os.TempDir(),
	}
}
