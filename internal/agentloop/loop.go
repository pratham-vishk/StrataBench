package agentloop

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/export"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/planner"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/report"
	"github.com/pratham-vishk/stratabench/internal/reporter"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

type Options struct {
	Intent        string
	Target        string
	Targets       []string
	Clients       []string
	Topology      string
	Mock          bool
	SkipValidate  bool
	CheckBaseline bool
	CacheBytes    int64
	UseOllama     bool
	OllamaURL     string
	OllamaModel   string
	DataDir       string
}

type Result struct {
	Plan       planner.PlanResult
	Validation schema.ValidationResult
	Run        *schema.RunResult
	Insights   []analyst.Insight
	ReportPath string
	JSONPath   string
	Summary    string
}

// Run executes plan → validate → run → analyze → report.
func Run(ctx context.Context, opts Options) (*Result, error) {
	if opts.DataDir == "" {
		opts.DataDir = paths.DataDir()
	}

	profiles, err := profile.List(paths.ProfilesDir())
	if err != nil {
		return nil, err
	}

	fmt.Println("→ Planning...")
	plan := planner.Plan(ctx, planner.PlanOptions{
		Intent:      opts.Intent,
		Profiles:    profiles,
		Hardware:    discovery.Snapshot(),
		UseOllama:   opts.UseOllama,
		OllamaURL:   opts.OllamaURL,
		OllamaModel: opts.OllamaModel,
	})
	fmt.Printf("  profile=%s source=%s\n  %s\n", plan.Profile, plan.Source, plan.Rationale)

	p, err := profile.LoadByName(paths.ProfilesDir(), plan.Profile)
	if err != nil {
		return nil, err
	}

	svc, err := orchestrator.NewService(opts.DataDir)
	if err != nil {
		return nil, err
	}
	defer svc.Close()

	fmt.Println("→ Validating...")
	runOpts := orchestrator.RunOptions{
		Profile:       p,
		Target:        opts.Target,
		Targets:       opts.Targets,
		Clients:       opts.Clients,
		Topology:      opts.Topology,
		Mock:          opts.Mock,
		SkipValidate:  opts.SkipValidate,
		CheckBaseline: opts.CheckBaseline,
		CacheBytes:    opts.CacheBytes,
		DataDir:       opts.DataDir,
	}
	validation := svc.Validate(runOpts)
	if validation.Passed {
		fmt.Println("  validation PASSED")
	} else {
		fmt.Println("  validation FAILED")
		for _, e := range validation.Errors {
			fmt.Printf("    ERROR: %s\n", e)
		}
		if !opts.SkipValidate {
			return &Result{Plan: plan, Validation: validation}, fmt.Errorf("validation failed")
		}
	}

	fmt.Println("→ Running benchmark...")
	run, err := svc.Run(ctx, runOpts)
	if err != nil {
		return nil, err
	}
	fmt.Printf("  run_id=%s IOPS=%.0f p99=%.0fµs\n", run.RunID, run.Results.IOPS, run.Results.LatencyUS.P99)

	fmt.Println("→ Analyzing...")
	regression := svc.CheckRegression(run)
	insights := analyst.Analyze(run, regression)
	analyst.PrintInsights(insights)

	reportPath := filepath.Join(paths.ReportsDir(), run.RunID+".html")
	jsonPath := filepath.Join(paths.ReportsDir(), run.RunID+".json")

	fmt.Println("→ Reporting...")
	summary := reporter.Summarize(ctx, run, insights, reporter.SummaryOptions{
		UseOllama:   opts.UseOllama,
		OllamaURL:   opts.OllamaURL,
		OllamaModel: opts.OllamaModel,
	})
	if err := report.WriteHTMLWithInsights(run, insights, summary, reportPath); err != nil {
		return nil, err
	}
	_ = export.WriteJSON(run, jsonPath)

	fmt.Printf("\n%s\n", summary)
	fmt.Printf("Report: %s\n", reportPath)

	return &Result{
		Plan:       plan,
		Validation: validation,
		Run:        run,
		Insights:   insights,
		ReportPath: reportPath,
		JSONPath:   jsonPath,
		Summary:    summary,
	}, nil
}
