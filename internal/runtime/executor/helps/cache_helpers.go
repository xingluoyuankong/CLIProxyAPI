package helps

import (
	"sync"
	"time"
)

type CodexCache struct {
	ID     string
	Expire time.Time
}

// codexCacheMap stores prompt cache IDs keyed by model+user_id.
// Protected by codexCacheMu. Entries use sliding expiration.
var (
	codexCacheMap = make(map[string]CodexCache)
	codexCacheMu  sync.RWMutex
)

const (
	// CodexCacheTTL controls how long CPA keeps stable Codex prompt cache IDs.
	CodexCacheTTL = 24 * time.Hour

	// CodexPromptCacheRetention is sent upstream for longer-lived prompt caching.
	CodexPromptCacheRetention = "24h"

	// codexCacheCleanupInterval controls how often expired entries are purged.
	codexCacheCleanupInterval = 30 * time.Minute
)

// codexCacheCleanupOnce ensures the background cleanup goroutine starts only once.
var codexCacheCleanupOnce sync.Once

// startCodexCacheCleanup launches a background goroutine that periodically
// removes expired entries from codexCacheMap to prevent memory leaks.
func startCodexCacheCleanup() {
	go func() {
		ticker := time.NewTicker(codexCacheCleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			purgeExpiredCodexCache()
		}
	}()
}

// purgeExpiredCodexCache removes entries that have expired.
func purgeExpiredCodexCache() {
	now := time.Now()
	codexCacheMu.Lock()
	defer codexCacheMu.Unlock()
	for key, cache := range codexCacheMap {
		if !cache.Expire.After(now) {
			delete(codexCacheMap, key)
		}
	}
}

// GetCodexCache retrieves a cached entry, returning ok=false if not found or expired.
// A hit renews the TTL so active sessions keep the same upstream prompt cache key.
func GetCodexCache(key string) (CodexCache, bool) {
	codexCacheCleanupOnce.Do(startCodexCacheCleanup)
	now := time.Now()

	codexCacheMu.Lock()
	defer codexCacheMu.Unlock()

	cache, ok := codexCacheMap[key]
	if !ok {
		return CodexCache{}, false
	}
	if !cache.Expire.After(now) {
		delete(codexCacheMap, key)
		return CodexCache{}, false
	}
	cache.Expire = now.Add(CodexCacheTTL)
	codexCacheMap[key] = cache
	return cache, true
}

// SetCodexCache stores a cache entry.
func SetCodexCache(key string, cache CodexCache) {
	codexCacheCleanupOnce.Do(startCodexCacheCleanup)
	if cache.Expire.IsZero() {
		cache.Expire = time.Now().Add(CodexCacheTTL)
	}
	codexCacheMu.Lock()
	codexCacheMap[key] = cache
	codexCacheMu.Unlock()
}

// ClearCodexCache removes all cached Codex prompt cache IDs.
func ClearCodexCache() {
	codexCacheMu.Lock()
	codexCacheMap = make(map[string]CodexCache)
	codexCacheMu.Unlock()
}
