package util

import "testing"

func TestGetProviderNameStaticFallbackForGPT55(t *testing.T) {
	providers := GetProviderName("gpt-5.5")
	if len(providers) == 0 {
		t.Fatal("expected a provider for gpt-5.5")
	}
	if providers[0] != "codex" {
		t.Fatalf("expected codex provider for gpt-5.5, got %v", providers)
	}
}

func TestGetProviderNameUnknownModel(t *testing.T) {
	if providers := GetProviderName("definitely-not-a-real-model"); len(providers) != 0 {
		t.Fatalf("expected no providers for unknown model, got %v", providers)
	}
}
