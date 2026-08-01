package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/ollama"
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
	hw := opts.Hardware
	if hw.CPUCores == 0 {
		hw = discovery.Snapshot()
	}

	prompt := buildPlannerPrompt(opts.Intent, opts.Profiles, hw)
	raw, err := ollama.Generate(ctx, ollama.Config{
		URL:   opts.OllamaURL,
		Model: opts.OllamaModel,
	}, prompt, true)
	if err != nil {
		return PlanResult{}, err
	}

	var parsed struct {
		Profile   string `json:"profile"`
		Rationale string `json:"rationale"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
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
