package helps

import (
	"testing"
	"time"
)

func TestSetCodexCacheDefaultsToLongTTL(t *testing.T) {
	ClearCodexCache()

	SetCodexCache("model-user", CodexCache{ID: "cache-id"})

	cache, ok := GetCodexCache("model-user")
	if !ok {
		t.Fatal("expected cache entry")
	}
	if cache.ID != "cache-id" {
		t.Fatalf("cache ID = %q, want cache-id", cache.ID)
	}
	if remaining := time.Until(cache.Expire); remaining < 23*time.Hour {
		t.Fatalf("cache TTL = %v, want close to %v", remaining, CodexCacheTTL)
	}
}

func TestGetCodexCacheRenewsTTL(t *testing.T) {
	ClearCodexCache()

	SetCodexCache("model-user", CodexCache{
		ID:     "cache-id",
		Expire: time.Now().Add(2 * time.Second),
	})

	cache, ok := GetCodexCache("model-user")
	if !ok {
		t.Fatal("expected cache entry")
	}
	if remaining := time.Until(cache.Expire); remaining < 23*time.Hour {
		t.Fatalf("renewed cache TTL = %v, want close to %v", remaining, CodexCacheTTL)
	}
}
