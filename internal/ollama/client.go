package ollama

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

type Config struct {
	URL   string
	Model string
}

func (c Config) url() string {
	if c.URL != "" {
		return strings.TrimRight(c.URL, "/")
	}
	if v := os.Getenv("OLLAMA_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:11434"
}

func (c Config) model() string {
	if c.Model != "" {
		return c.Model
	}
	if v := os.Getenv("OLLAMA_MODEL"); v != "" {
		return v
	}
	return "llama3.2"
}

// Generate calls Ollama /api/generate and returns the response text.
func Generate(ctx context.Context, cfg Config, prompt string, jsonFormat bool) (string, error) {
	body := map[string]any{
		"model":  cfg.model(),
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.url()+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(b))
	}

	var genResp struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", err
	}
	return strings.TrimSpace(genResp.Response), nil
}
