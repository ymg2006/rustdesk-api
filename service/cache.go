package service

import (
	"context"
	"errors"
	"time"

	cachelib "github.com/ymg2006/rustdesk-api/v2/lib/cache"
)

const (
	appReleaseCacheTTL     = 5 * time.Minute
	clientDownloadCacheTTL = 5 * time.Minute
	announcementCacheTTL   = 2 * time.Minute

	clientDownloadActiveCacheKey = "rustdesk-api:v1:client-downloads:active"
	announcementActiveCacheKey   = "rustdesk-api:v1:announcements:active"
)

var Cache cachelib.Cache

// SetCache supplies the optional application cache. A nil cache disables
// caching without changing service behavior.
func SetCache(c cachelib.Cache) {
	Cache = c
}

func cacheGet(key string, dest interface{}) bool {
	if Cache == nil {
		return false
	}
	err := Cache.Get(context.Background(), key, dest)
	if err == nil {
		return true
	}
	if !errors.Is(err, cachelib.ErrCacheMiss) && Logger != nil {
		Logger.Warnf("cache get failed for %q: %v", key, err)
	}
	return false
}

func cacheSet(key string, value interface{}, ttl time.Duration) {
	if Cache == nil {
		return
	}
	if err := Cache.Set(context.Background(), key, value, ttl); err != nil && Logger != nil {
		Logger.Warnf("cache set failed for %q: %v", key, err)
	}
}

func cacheDelete(keys ...string) {
	if Cache == nil {
		return
	}
	for _, key := range keys {
		if err := Cache.Delete(context.Background(), key); err != nil && Logger != nil {
			Logger.Warnf("cache delete failed for %q: %v", key, err)
		}
	}
}
