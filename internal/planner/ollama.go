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
	Profile   string         `json:"profile"`
	Rationale string         `json:"rationale"`
	Source    string         `json:"source"`
	Target    string         `json:"target,omitempty"`
	Targets   []string       `json:"targets,omitempty"`
	Clients   []string       `json:"clients,omitempty"`
	Topology  string         `json:"topology,omitempty"`
	Params    map[string]any `json:"params,omitempty"`
}

func Plan(ctx context.Context, opts PlanOptions) PlanResult {
	parsed := ParseIntent(opts.Intent)
	var base PlanResult
	if opts.UseLLM || opts.UseOllama {
		if res, err := planWithLLM(ctx, opts); err == nil {
			base = res
		}
	}
	if base.Profile == "" {
		name := SuggestProfile(opts.Intent, opts.Profiles)
		base = PlanResult{
			Profile:   name,
			Rationale: "keyword match (LLM unavailable or disabled)",
			Source:    "keyword",
		}
	}
	return mergeParsedIntoPlan(base, parsed)
}

func mergeParsedIntoPlan(base PlanResult, parsed ParsedIntent) PlanResult {
	if base.Params == nil {
		base.Params = map[string]any{}
	}
	for k, v := range parsed.Params {
		if _, exists := base.Params[k]; !exists {
			base.Params[k] = v
		}
	}
	if base.Target == "" {
		base.Target = parsed.Target
	}
	if len(base.Targets) == 0 {
		base.Targets = parsed.Targets
	}
	if len(base.Clients) == 0 {
		base.Clients = parsed.Clients
	}
	if base.Topology == "" {
		base.Topology = parsed.Topology
	}
	return base
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
		Profile   string         `json:"profile"`
		Rationale string         `json:"rationale"`
		Target    string         `json:"target"`
		Targets   []string       `json:"targets"`
		Clients   []string       `json:"clients"`
		Topology  string         `json:"topology"`
		Params    map[string]any `json:"params"`
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
		Target:    parsed.Target,
		Targets:   parsed.Targets,
		Clients:   parsed.Clients,
		Topology:  parsed.Topology,
		Params:    parsed.Params,
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
