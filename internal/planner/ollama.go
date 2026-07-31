package planner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

type PlanOptions struct {
	Intent      string
	Profiles    []*profile.Profile
	Hardware    schema.HardwareSnapshot
	UseOllama   bool
	OllamaURL   string
	OllamaModel string
}

type PlanResult struct {
	Profile   string `json:"profile"`
	Rationale string `json:"rationale"`
	Source    string `json:"source"`
}

func Plan(ctx context.Context, opts PlanOptions) PlanResult {
	if opts.UseOllama {
		if res, err := planWithOllama(ctx, opts); err == nil {
			return res
		}
	}
	name := SuggestProfile(opts.Intent, opts.Profiles)
	return PlanResult{
		Profile:   name,
		Rationale: "keyword match (Ollama unavailable or disabled)",
		Source:    "keyword",
	}
}

func planWithOllama(ctx context.Context, opts PlanOptions) (PlanResult, error) {
	url := opts.OllamaURL
	if url == "" {
		url = os.Getenv("OLLAMA_URL")
	}
	if url == "" {
		url = "http://localhost:11434"
	}
	model := opts.OllamaModel
	if model == "" {
		model = os.Getenv("OLLAMA_MODEL")
	}
	if model == "" {
		model = "llama3.2"
	}

	hw := opts.Hardware
	if hw.CPUCores == 0 {
		hw = discovery.Snapshot()
	}

	prompt := buildPlannerPrompt(opts.Intent, opts.Profiles, hw)
	body, _ := json.Marshal(map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"format": "json",
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(url, "/")+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return PlanResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return PlanResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return PlanResult{}, fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(b))
	}

	var genResp struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return PlanResult{}, err
	}

	var parsed struct {
		Profile   string `json:"profile"`
		Rationale string `json:"rationale"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(genResp.Response)), &parsed); err != nil {
		return PlanResult{}, fmt.Errorf("parse ollama response: %w", err)
	}
	if parsed.Profile == "" {
		return PlanResult{}, fmt.Errorf("ollama returned empty profile")
	}
	if !profileExists(opts.Profiles, parsed.Profile) {
		return PlanResult{}, fmt.Errorf("ollama suggested unknown profile %q", parsed.Profile)
	}
	return PlanResult{
		Profile:   parsed.Profile,
		Rationale: parsed.Rationale,
		Source:    "ollama",
	}, nil
}

func buildPlannerPrompt(intent string, profiles []*profile.Profile, hw schema.HardwareSnapshot) string {
	base := loadPlannerPrompt()
	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\n\nHardware:\n")
	fmt.Fprintf(&b, "- OS: %s/%s, CPU cores: %d, memory: %d GB\n", hw.OS, hw.Arch, hw.CPUCores, hw.MemoryBytes/(1<<30))
	b.WriteString("\nProfile catalog:\n")
	for _, p := range profiles {
		fmt.Fprintf(&b, "- %s: layer=%s engine=%s load=%s — %s\n", p.Name, p.Layer, p.Engine, p.Load, p.Description)
	}
	fmt.Fprintf(&b, "\nUser intent: %s\n", intent)
	return b.String()
}

func loadPlannerPrompt() string {
	path := filepath.Join(paths.RepoRoot(), "agents", "planner.prompt")
	data, err := os.ReadFile(path)
	if err != nil {
		return "Select the best StrataBench workload profile for the user intent. Respond with JSON: {\"profile\":\"...\",\"rationale\":\"...\"}"
	}
	return string(data)
}

func profileExists(profiles []*profile.Profile, name string) bool {
	for _, p := range profiles {
		if p.Name == name {
			return true
		}
	}
	return false
}
