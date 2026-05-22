package helps

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func resetSessionIDCache() {
	sessionIDCacheMu.Lock()
	sessionIDCache = make(map[string]sessionIDCacheEntry)
	sessionIDCacheMu.Unlock()
}

func TestCachedSessionID_IsStableAfterCacheReset(t *testing.T) {
	resetSessionIDCache()

	first := CachedSessionID("api-key-stable")
	resetSessionIDCache()
	second := CachedSessionID("api-key-stable")

	if first == "" || second == "" {
		t.Fatalf("expected stable session IDs to be non-empty, got %q and %q", first, second)
	}
	if first != second {
		t.Fatalf("expected session ID to survive cache reset by derivation, got %q and %q", first, second)
	}
	if _, err := uuid.Parse(first); err != nil {
		t.Fatalf("stable session ID is not a UUID: %v", err)
	}
}

func TestCachedSessionID_RenewsSevenDayTTLOnHit(t *testing.T) {
	resetSessionIDCache()

	key := "api-key-renew"
	id := CachedSessionID(key)
	cacheKey := sessionIDCacheKey(key)

	soon := time.Now()
	sessionIDCacheMu.Lock()
	sessionIDCache[cacheKey] = sessionIDCacheEntry{
		value:  id,
		expire: soon.Add(2 * time.Second),
	}
	sessionIDCacheMu.Unlock()

	if refreshed := CachedSessionID(key); refreshed != id {
		t.Fatalf("expected cached session ID to be reused before expiry, got %q", refreshed)
	}

	sessionIDCacheMu.RLock()
	entry := sessionIDCache[cacheKey]
	sessionIDCacheMu.RUnlock()

	if entry.expire.Sub(soon) < 6*24*time.Hour {
		t.Fatalf("expected TTL to renew, got %v remaining", entry.expire.Sub(soon))
	}
}
