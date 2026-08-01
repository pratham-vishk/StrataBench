package llm

import "testing"

func TestFromEnvDefaults(t *testing.T) {
	cfg := Config{}.resolved()
	if cfg.Provider != ProviderOllama {
		t.Fatalf("provider=%s", cfg.Provider)
	}
	if cfg.BaseURL != "http://localhost:11434" {
		t.Fatalf("baseURL=%s", cfg.BaseURL)
	}
}

func TestFromEnvOpenAI(t *testing.T) {
	cfg := Config{Provider: ProviderOpenAI, APIKey: "sk-test", Model: "gpt-4o-mini"}.resolved()
	if cfg.Provider != ProviderOpenAI {
		t.Fatalf("provider=%s", cfg.Provider)
	}
	if cfg.BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("baseURL=%s", cfg.BaseURL)
	}
}
