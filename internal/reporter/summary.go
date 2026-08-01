package reporter

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/llm"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

type SummaryOptions struct {
	UseLLM      bool
	UseOllama   bool // deprecated alias
	LLM         llm.Config
	OllamaURL   string
	OllamaModel string
}

// Summarize returns an NL executive summary, using an LLM when enabled.
func Summarize(ctx context.Context, run *schema.RunResult, insights []analyst.Insight, opts SummaryOptions) string {
	fallback := analyst.SummaryText(run, insights)
	if !(opts.UseLLM || opts.UseOllama) {
		return fallback
	}

	prompt := buildReporterPrompt(run, insights)
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
	text, err := llm.Generate(ctx, cfg, prompt, false)
	if err != nil || text == "" {
		return fallback
	}
	return text
}

func buildReporterPrompt(run *schema.RunResult, insights []analyst.Insight) string {
	base := loadReporterPrompt()
	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\n\nBenchmark data:\n")
	fmt.Fprintf(&b, "- Profile: %s (%s layer, %s engine)\n", run.Profile, run.Layer, run.Engine)
	fmt.Fprintf(&b, "- Target: %s\n", run.Target.Device)
	fmt.Fprintf(&b, "- IOPS: %.0f, Throughput: %.2f MB/s, p99: %.0f µs\n",
		run.Results.IOPS, run.Results.ThroughputMBps, run.Results.LatencyUS.P99)
	if run.Mock {
		b.WriteString("- Mode: mock (synthetic results)\n")
	}
	if len(insights) > 0 {
		b.WriteString("- Analyst findings:\n")
		for _, ins := range insights {
			fmt.Fprintf(&b, "  * [%s] %s\n", ins.Severity, ins.Message)
		}
	}
	return b.String()
}

func loadReporterPrompt() string {
	path := filepath.Join(paths.RepoRoot(), "agents", "reporter.prompt")
	data, err := os.ReadFile(path)
	if err != nil {
		return "Write a brief executive summary of this storage benchmark."
	}
	return string(data)
}
