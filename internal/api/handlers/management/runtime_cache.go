package management

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/cache"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
)

// ClearRuntimeCache clears runtime-only caches without touching usage statistics.
func (h *Handler) ClearRuntimeCache(c *gin.Context) {
	helps.ClearCodexCache()
	cache.ClearSignatureCache("")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"cleared": []string{
			"codex_prompt_cache",
			"antigravity_signature_cache",
		},
	})
}
