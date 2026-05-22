package management

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/cache"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/redisqueue"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
)

func TestClearRuntimeCacheClearsRuntimeCachesOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	helps.ClearCodexCache()
	cache.ClearSignatureCache("")
	redisqueue.SetEnabled(true)
	defer redisqueue.SetEnabled(false)
	_ = redisqueue.PopOldest(100)

	helps.SetCodexCache("model-user", helps.CodexCache{
		ID:     "cache-id",
		Expire: time.Now().Add(time.Hour),
	})
	cache.CacheSignature("claude-3-5-sonnet", "thinking text", strings.Repeat("s", 50))
	usageBefore := []byte(`{"sentinel":"usage-record"}`)
	redisqueue.Enqueue(usageBefore)

	if _, ok := helps.GetCodexCache("model-user"); !ok {
		t.Fatal("expected Codex prompt cache to be seeded")
	}
	if got := cache.GetCachedSignature("claude-3-5-sonnet", "thinking text"); got == "" {
		t.Fatal("expected signature cache to be seeded")
	}

	router := gin.New()
	handler := &Handler{}
	router.POST("/v0/management/runtime-cache/clear", handler.ClearRuntimeCache)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v0/management/runtime-cache/clear", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Success bool     `json:"success"`
		Cleared []string `json:"cleared"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Success {
		t.Fatal("success = false, want true")
	}
	if len(body.Cleared) != 2 {
		t.Fatalf("cleared len = %d, want 2", len(body.Cleared))
	}

	if _, ok := helps.GetCodexCache("model-user"); ok {
		t.Fatal("expected Codex prompt cache to be cleared")
	}
	if got := cache.GetCachedSignature("claude-3-5-sonnet", "thinking text"); got != "" {
		t.Fatalf("expected signature cache to be cleared, got %q", got)
	}

	usageAfter := redisqueue.PopOldest(10)
	if len(usageAfter) != 1 || !bytes.Equal(usageAfter[0], usageBefore) {
		t.Fatalf("usage queue changed: got %q, want %q", usageAfter, usageBefore)
	}
}
