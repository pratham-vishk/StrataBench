package agentloop

import (
	"context"
	"fmt"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/llm"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/planner"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/report"
	"github.com/pratham-vishk/stratabench/internal/reporter"
	"github.com/pratham-vishk/stratabench/internal/schema"
	"github.com/pratham-vishk/stratabench/internal/topology"
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
	CheckHardware bool
	CacheBytes    int64
	AssumeDefaults bool // --yes: proceed despite open questions
	UseLLM        bool
	UseOllama     bool
	LLM           llm.Config
	OllamaURL     string
	OllamaModel   string
	DataDir       string
	OpenReport    bool
}

type Result struct {
	Plan       planner.PlanResult
	Guidance   planner.Guidance
	Validation schema.ValidationResult
	Run        *schema.RunResult
	Insights   []analyst.Insight
	ReportPath string
	ExcelPath  string
	JSONPath   string
	PDFPath    string
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
	llmCfg := opts.LLM
	if llmCfg.Model == "" && opts.OllamaModel != "" {
		llmCfg.Model = opts.OllamaModel
	}
	if llmCfg.BaseURL == "" && opts.OllamaURL != "" {
		llmCfg.BaseURL = opts.OllamaURL
	}
	if (opts.UseLLM || opts.UseOllama) && llmCfg.Model == "" && llmCfg.BaseURL == "" && llmCfg.APIKey == "" {
		llmCfg = llm.FromEnv()
	}
	plan := planner.Plan(ctx, planner.PlanOptions{
		Intent:      opts.Intent,
		Profiles:    profiles,
		Hardware:    discovery.Snapshot(),
		UseLLM:      opts.UseLLM || opts.UseOllama,
		UseOllama:   opts.UseOllama,
		LLM:         llmCfg,
		OllamaURL:   opts.OllamaURL,
		OllamaModel: opts.OllamaModel,
	})
	plan = planner.MergePlan(plan, planner.ParsedIntent{}, opts.Target, opts.Targets, opts.Clients, opts.Topology)
	printPlan(plan)

	target := plan.Target
	targets := plan.Targets
	if target != "" && len(targets) == 0 {
		targets = []string{target}
	}
	if len(targets) == 0 && opts.Target != "" {
		target = opts.Target
		targets = topology.MergeTargets(opts.Target, opts.Targets)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no target — specify --target/--targets or include endpoints in intent (e.g. servers 10.0.1.10:9000)")
	}
	if target == "" {
		target = targets[0]
	}

	p, err := profile.LoadByName(paths.ProfilesDir(), plan.Profile)
	if err != nil {
		return nil, err
	}
	prof := p.Clone()
	guidance := planner.Guide(plan, opts.Intent, p)
	fmt.Println("→ Guidance...")
	fmt.Print(planner.FormatGuidance(guidance))
	plan = applyGuidanceDefaults(plan, guidance)
	prof.ApplyOverrides(plan.Params)
	if !guidance.Ready && !opts.AssumeDefaults {
		return &Result{Plan: plan, Guidance: guidance}, fmt.Errorf("clarification needed — refine intent or pass --yes to proceed with recommendations")
	}

	svc, err := orchestrator.NewService(opts.DataDir)
	if err != nil {
		return nil, err
	}
	defer svc.Close()

	fmt.Println("→ Validating...")
	runOpts := orchestrator.RunOptions{
		Profile:       prof,
		Target:        target,
		Targets:       targets,
		Clients:       plan.Clients,
		Topology:      plan.Topology,
		Mock:          opts.Mock,
		SkipValidate:  opts.SkipValidate,
		CheckBaseline: opts.CheckBaseline,
		CheckHardware: opts.CheckHardware,
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
			return &Result{Plan: plan, Guidance: guidance, Validation: validation}, fmt.Errorf("validation failed")
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

	fmt.Println("→ Reporting...")
	summary := reporter.Summarize(ctx, run, insights, reporter.SummaryOptions{
		UseLLM:      opts.UseLLM || opts.UseOllama,
		UseOllama:   opts.UseOllama,
		LLM:         llmCfg,
		OllamaURL:   opts.OllamaURL,
		OllamaModel: opts.OllamaModel,
	})
	arts, err := report.WriteRunArtifacts(run, report.OptionsFromAnalysis(insights, summary, regression))
	if err != nil {
		return nil, err
	}

	fmt.Printf("\n%s\n", summary)
	fmt.Printf("Report: %s\n", arts.HTML)
	fmt.Printf("Excel:  %s\n", arts.Excel)
	fmt.Printf("PDF:    %s\n", arts.PDF)
	if opts.OpenReport {
		_ = report.OpenInBrowser(arts.HTML)
	}

	return &Result{
		Plan:       plan,
		Guidance:   guidance,
		Validation: validation,
		Run:        run,
		Insights:   insights,
		ReportPath: arts.HTML,
		ExcelPath:  arts.Excel,
		JSONPath:   arts.JSON,
		PDFPath:    arts.PDF,
		Summary:    summary,
	}, nil
}

func printPlan(plan planner.PlanResult) {
	fmt.Printf("  profile=%s source=%s\n  %s\n", plan.Profile, plan.Source, plan.Rationale)
	if plan.Target != "" {
		fmt.Printf("  target=%s\n", plan.Target)
	}
	if len(plan.Targets) > 0 {
		fmt.Printf("  targets=%v\n", plan.Targets)
	}
	if len(plan.Clients) > 0 {
		fmt.Printf("  clients=%v topology=%s\n", plan.Clients, plan.Topology)
	}
	if len(plan.Params) > 0 {
		fmt.Printf("  params=%v\n", plan.Params)
	}
}

func applyGuidanceDefaults(plan planner.PlanResult, g planner.Guidance) planner.PlanResult {
	if len(g.AppliedDefaults) == 0 {
		return plan
	}
	if plan.Params == nil {
		plan.Params = map[string]any{}
	}
	for k, v := range g.AppliedDefaults {
		if k == "topology" {
			if plan.Topology == "" || plan.Topology == "auto" {
				if s, ok := v.(string); ok {
					plan.Topology = s
				}
			}
			continue
		}
		if _, exists := plan.Params[k]; !exists {
			plan.Params[k] = v
		}
	}
	return plan
}
