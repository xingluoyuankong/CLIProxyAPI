package management

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
)

type apiKeyUsageEntry struct {
	Success        int64                          `json:"success"`
	Failed         int64                          `json:"failed"`
	RecentRequests []coreauth.RecentRequestBucket `json:"recent_requests"`
	Summary        apiKeyUsageSummary             `json:"summary"`
}

type apiKeyUsageSummary struct {
	WindowSeconds    int64   `json:"window_seconds"`
	BucketSeconds    int64   `json:"bucket_seconds"`
	WindowRequests   int64   `json:"window_requests"`
	InstantRPM       float64 `json:"instant_rpm"`
	AverageRPM       float64 `json:"average_rpm"`
	InstantLatencyMs float64 `json:"instant_latency_ms"`
	AverageLatencyMs float64 `json:"average_latency_ms"`
	LatestLatencyMs  int64   `json:"latest_latency_ms"`
}

func mergeRecentRequestBuckets(dst, src []coreauth.RecentRequestBucket) []coreauth.RecentRequestBucket {
	if len(dst) == 0 {
		return src
	}
	if len(src) == 0 {
		return dst
	}
	if len(dst) != len(src) {
		n := len(dst)
		if len(src) < n {
			n = len(src)
		}
		for i := 0; i < n; i++ {
			mergeRecentRequestBucket(&dst[i], src[i])
		}
		return dst
	}
	for i := range dst {
		mergeRecentRequestBucket(&dst[i], src[i])
	}
	return dst
}

func mergeRecentRequestBucket(dst *coreauth.RecentRequestBucket, src coreauth.RecentRequestBucket) {
	if dst == nil {
		return
	}
	dst.Success += src.Success
	dst.Failed += src.Failed
	dst.LatencyMs += src.LatencyMs
	dst.LatencyCount += src.LatencyCount
	if dst.Time == "" {
		dst.Time = src.Time
	}
	if src.LastLatencyMs > 0 {
		dst.LastLatencyMs = src.LastLatencyMs
	}
	if dst.LatencyCount > 0 {
		dst.AvgLatencyMs = float64(dst.LatencyMs) / float64(dst.LatencyCount)
	} else {
		dst.AvgLatencyMs = 0
	}
}

func summarizeRecentRequestBuckets(buckets []coreauth.RecentRequestBucket) apiKeyUsageSummary {
	const bucketSeconds int64 = 10 * 60
	summary := apiKeyUsageSummary{
		WindowSeconds: int64(len(buckets)) * bucketSeconds,
		BucketSeconds: bucketSeconds,
	}
	if len(buckets) == 0 {
		return summary
	}

	var latencyMs int64
	var latencyCount int64
	var latestLatency int64
	for i := range buckets {
		bucket := buckets[i]
		requests := bucket.Success + bucket.Failed
		summary.WindowRequests += requests
		latencyMs += bucket.LatencyMs
		latencyCount += bucket.LatencyCount
		if bucket.LastLatencyMs > 0 {
			latestLatency = bucket.LastLatencyMs
		}
	}

	newest := buckets[len(buckets)-1]
	newestRequests := newest.Success + newest.Failed
	summary.InstantRPM = float64(newestRequests) / (float64(bucketSeconds) / 60.0)
	if newest.LatencyCount > 0 {
		summary.InstantLatencyMs = float64(newest.LatencyMs) / float64(newest.LatencyCount)
	}
	if summary.WindowSeconds > 0 {
		summary.AverageRPM = float64(summary.WindowRequests) / (float64(summary.WindowSeconds) / 60.0)
	}
	if latencyCount > 0 {
		summary.AverageLatencyMs = float64(latencyMs) / float64(latencyCount)
	}
	summary.LatestLatencyMs = latestLatency

	return summary
}

// GetAPIKeyUsage returns recent request buckets for all in-memory api_key auths,
// grouped by provider and keyed by "base_url|api_key".
func (h *Handler) GetAPIKeyUsage(c *gin.Context) {
	if h == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "handler not initialized"})
		return
	}

	h.mu.Lock()
	manager := h.authManager
	h.mu.Unlock()
	if manager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "core auth manager unavailable"})
		return
	}

	now := time.Now()
	out := make(map[string]map[string]apiKeyUsageEntry)
	for _, auth := range manager.List() {
		if auth == nil {
			continue
		}
		kind, apiKey := auth.AccountInfo()
		if !strings.EqualFold(strings.TrimSpace(kind), "api_key") {
			continue
		}
		apiKey = strings.TrimSpace(apiKey)
		if apiKey == "" {
			continue
		}
		baseURL := ""
		if auth.Attributes != nil {
			baseURL = strings.TrimSpace(auth.Attributes["base_url"])
			if baseURL == "" {
				baseURL = strings.TrimSpace(auth.Attributes["base-url"])
			}
		}
		compositeKey := baseURL + "|" + apiKey
		provider := strings.ToLower(strings.TrimSpace(auth.Provider))
		if provider == "" {
			provider = "unknown"
		}

		recent := auth.RecentRequestsSnapshot(now)
		providerBucket, ok := out[provider]
		if !ok {
			providerBucket = make(map[string]apiKeyUsageEntry)
			out[provider] = providerBucket
		}
		if existing, exists := providerBucket[compositeKey]; exists {
			existing.Success += auth.Success
			existing.Failed += auth.Failed
			existing.RecentRequests = mergeRecentRequestBuckets(existing.RecentRequests, recent)
			providerBucket[compositeKey] = existing
			continue
		}
		providerBucket[compositeKey] = apiKeyUsageEntry{
			Success:        auth.Success,
			Failed:         auth.Failed,
			RecentRequests: recent,
			Summary:        summarizeRecentRequestBuckets(recent),
		}
	}

	for provider, providerBucket := range out {
		for compositeKey, entry := range providerBucket {
			entry.Summary = summarizeRecentRequestBuckets(entry.RecentRequests)
			providerBucket[compositeKey] = entry
		}
		out[provider] = providerBucket
	}

	c.JSON(http.StatusOK, out)
}
