package management

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/cache"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/runtime/executor/helps"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

func TestClearRuntimeCacheClearsRuntimeCachesOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	helps.ClearCodexCache()
	cache.ClearSignatureCache("")

	helps.SetCodexCache("model-user", helps.CodexCache{
		ID:     "cache-id",
		Expire: time.Now().Add(time.Hour),
	})
	cache.CacheSignature("claude-3-5-sonnet", "thinking text", strings.Repeat("s", 50))

	if _, ok := helps.GetCodexCache("model-user"); !ok {
		t.Fatal("expected Codex prompt cache to be seeded")
	}
	if got := cache.GetCachedSignature("claude-3-5-sonnet", "thinking text"); got == "" {
		t.Fatal("expected signature cache to be seeded")
	}

	stats := usage.NewRequestStatistics()
	stats.Record(context.Background(), coreusage.Record{
		APIKey:      "test-key",
		Model:       "gpt-5.4",
		RequestedAt: time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC),
		Detail: coreusage.Detail{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
	})
	before := stats.Snapshot()

	router := gin.New()
	handler := &Handler{usageStats: stats}
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

	after := stats.Snapshot()
	if after.TotalRequests != before.TotalRequests ||
		after.SuccessCount != before.SuccessCount ||
		after.FailureCount != before.FailureCount ||
		after.TotalTokens != before.TotalTokens {
		t.Fatalf("usage stats changed: before=%+v after=%+v", before, after)
	}
}
