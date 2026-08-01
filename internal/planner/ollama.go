package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/llm"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

type PlanOptions struct {
	Intent      string
	Profiles    []*profile.Profile
	Hardware    schema.HardwareSnapshot
	UseLLM      bool
	UseOllama   bool // deprecated alias for UseLLM
	LLM         llm.Config
	OllamaURL   string // deprecated; use LLM.BaseURL
	OllamaModel string // deprecated; use LLM.Model
}

type PlanResult struct {
	Profile   string `json:"profile"`
	Rationale string `json:"rationale"`
	Source    string `json:"source"`
}

func Plan(ctx context.Context, opts PlanOptions) PlanResult {
	if opts.UseLLM || opts.UseOllama {
		if res, err := planWithLLM(ctx, opts); err == nil {
			return res
		}
	}
	name := SuggestProfile(opts.Intent, opts.Profiles)
	return PlanResult{
		Profile:   name,
		Rationale: "keyword match (LLM unavailable or disabled)",
		Source:    "keyword",
	}
}

func planWithLLM(ctx context.Context, opts PlanOptions) (PlanResult, error) {
	hw := opts.Hardware
	if hw.CPUCores == 0 {
		hw = discovery.Snapshot()
	}

	prompt := buildPlannerPrompt(opts.Intent, opts.Profiles, hw)
	cfg := opts.LLM
	if cfg.Model == "" && opts.OllamaModel != "" {
		cfg.Model = opts.OllamaModel
	}
	if cfg.BaseURL == "" && opts.OllamaURL != "" {
		cfg.BaseURL = opts.OllamaURL
	}
	if cfg.Model == "" && cfg.BaseURL == "" && cfg.APIKey == "" {
		cfg = llm.FromEnv()
	}
	raw, err := llm.Generate(ctx, cfg, prompt, true)
	if err != nil {
		return PlanResult{}, err
	}

	var parsed struct {
		Profile   string `json:"profile"`
		Rationale string `json:"rationale"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return PlanResult{}, fmt.Errorf("parse llm response: %w", err)
	}
	if parsed.Profile == "" {
		return PlanResult{}, fmt.Errorf("llm returned empty profile")
	}
	if !profileExists(opts.Profiles, parsed.Profile) {
		return PlanResult{}, fmt.Errorf("llm suggested unknown profile %q", parsed.Profile)
	}
	source := cfg.Provider
	if source == "" || source == llm.ProviderAuto {
		source = "llm"
	}
	return PlanResult{
		Profile:   parsed.Profile,
		Rationale: parsed.Rationale,
		Source:    source,
	}, nil
}

func planWithOllama(ctx context.Context, opts PlanOptions) (PlanResult, error) {
	return planWithLLM(ctx, opts)
}

func buildPlannerPrompt(intent string, profiles []*profile.Profile, hw schema.HardwareSnapshot) string {
	base := loadPlannerPrompt()
	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\n\nHardware:\n")
	fmt.Fprintf(&b, "- Host: %s, OS: %s/%s, CPU cores: %d, memory: %d GB\n",
		hw.Hostname, hw.OS, hw.Arch, hw.CPUCores, hw.MemoryBytes/(1<<30))
	if len(hw.NVMe) > 0 {
		b.WriteString("- NVMe devices:\n")
		for _, d := range hw.NVMe {
			fmt.Fprintf(&b, "  * %s %s fw=%s\n", d.Device, d.Model, d.Firmware)
		}
	}
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
