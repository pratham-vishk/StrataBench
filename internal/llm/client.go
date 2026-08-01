package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Provider names for STRATABENCH_LLM_PROVIDER.
const (
	ProviderOllama  = "ollama"
	ProviderOpenAI  = "openai"
	ProviderAuto    = "auto"
)

// Config selects an LLM backend. Works with Ollama, OpenAI, and any OpenAI-compatible API
// (Cursor models via proxy, LiteLLM, vLLM, localai, etc.).
type Config struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
}

func FromEnv() Config {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("STRATABENCH_LLM_PROVIDER")))
	if provider == "" {
		provider = ProviderAuto
	}
	cfg := Config{
		Provider: provider,
		BaseURL:  firstNonEmpty(os.Getenv("STRATABENCH_LLM_URL"), os.Getenv("OPENAI_BASE_URL"), os.Getenv("OLLAMA_URL")),
		APIKey:   firstNonEmpty(os.Getenv("STRATABENCH_LLM_API_KEY"), os.Getenv("OPENAI_API_KEY")),
		Model:    firstNonEmpty(os.Getenv("STRATABENCH_LLM_MODEL"), os.Getenv("OPENAI_MODEL"), os.Getenv("OLLAMA_MODEL")),
	}
	if cfg.Model == "" {
		cfg.Model = "llama3.2"
	}
	return cfg
}

func (c Config) resolved() Config {
	out := c
	if out.Provider == "" || out.Provider == ProviderAuto {
		if out.APIKey != "" || strings.Contains(strings.ToLower(out.BaseURL), "openai") {
			out.Provider = ProviderOpenAI
		} else {
			out.Provider = ProviderOllama
		}
	}
	if out.BaseURL == "" {
		if out.Provider == ProviderOpenAI {
			out.BaseURL = "https://api.openai.com/v1"
		} else {
			out.BaseURL = "http://localhost:11434"
		}
	}
	out.BaseURL = strings.TrimRight(out.BaseURL, "/")
	return out
}

// Generate returns model text for a single user prompt.
func Generate(ctx context.Context, cfg Config, prompt string, jsonFormat bool) (string, error) {
	cfg = cfg.resolved()
	switch cfg.Provider {
	case ProviderOpenAI:
		return generateOpenAI(ctx, cfg, prompt, jsonFormat)
	default:
		return generateOllama(ctx, cfg, prompt, jsonFormat)
	}
}

func generateOllama(ctx context.Context, cfg Config, prompt string, jsonFormat bool) (string, error) {
	body := map[string]any{
		"model":  cfg.Model,
		"prompt": prompt,
		"stream": false,
	}
	if jsonFormat {
		body["format"] = "json"
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	return doTextRequest(req, func(raw []byte) (string, error) {
		var genResp struct {
			Response string `json:"response"`
		}
		if err := json.Unmarshal(raw, &genResp); err != nil {
			return "", err
		}
		return strings.TrimSpace(genResp.Response), nil
	})
}

func generateOpenAI(ctx context.Context, cfg Config, prompt string, jsonFormat bool) (string, error) {
	msg := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	if jsonFormat {
		msg["response_format"] = map[string]string{"type": "json_object"}
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	url := cfg.BaseURL
	if !strings.HasSuffix(url, "/chat/completions") {
		url += "/chat/completions"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	return doTextRequest(req, func(raw []byte) (string, error) {
		var chatResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(raw, &chatResp); err != nil {
			return "", err
		}
		if chatResp.Error != nil {
			return "", fmt.Errorf("openai error: %s", chatResp.Error.Message)
		}
		if len(chatResp.Choices) == 0 {
			return "", fmt.Errorf("openai returned no choices")
		}
		return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
	})
}

func doTextRequest(req *http.Request, parse func([]byte) (string, error)) (string, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm status %d: %s", resp.StatusCode, string(raw))
	}
	text, err := parse(raw)
	if err != nil {
		return "", err
	}
	if text == "" {
		return "", fmt.Errorf("llm returned empty response")
	}
	return text, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
