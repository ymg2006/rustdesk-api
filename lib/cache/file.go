package cache

import (
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

// getValue reads the disk value; returns an empty string when the file does not exist/has expired/is damaged (filter error)
func (c *FileCache) getValue(key string) string {
	f := c.fileName(key)
	lock := c.getLock(f)
	lock.Lock()
	defer lock.Unlock()

	data, err := os.ReadFile(f)
	if err != nil {
		return ""
	}
	var item fileItem
	if err := json.Unmarshal(data, &item); err != nil {
		// The file is damaged (including old format) and will be considered a miss after deletion.
		os.Remove(f)
		return ""
	}
	if item.Exp > 0 && time.Now().Unix() >= item.Exp {
		os.Remove(f)
		return ""
	}
	return item.Data
}

// Get reads the cache; value remains zero on a miss and no error is returned (consistent with SimpleCache behavior)
func (c *FileCache) Get(key string, value interface{}) error {
	data := c.getValue(key)
	if data == "" {
		return nil
	}
	return DecodeValue(data, value)
}

func (c *FileCache) saveValue(key string, value string, exp int) error {
	f := c.fileName(key)
	lock := c.getLock(f)
	lock.Lock()
	defer lock.Unlock()

	if exp <= 0 {
		exp = MaxTimeOut
	}
	item := fileItem{
		Exp:  time.Now().Add(time.Duration(exp) * time.Second).Unix(),
		Data: value,
	}
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return os.WriteFile(f, b, 0644)
}

func (c *FileCache) Set(key string, value interface{}, exp int) error {
	str, err := EncodeValue(value)
	if err != nil {
		return err
	}
	return c.saveValue(key, str, exp)
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
