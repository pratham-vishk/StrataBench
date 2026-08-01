package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pratham-vishk/stratabench/internal/agentloop"
	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/llm"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/planner"
	"github.com/pratham-vishk/stratabench/internal/profile"
)

// Tools exposes StrataBench operations for MCP clients (Cursor, Claude Code, etc.).
type Tools struct {
	DataDir string
}

func (t *Tools) dataDir() string {
	if t.DataDir != "" {
		return t.DataDir
	}
	return paths.DataDir()
}

func (t *Tools) ListProfiles(_ context.Context, _ map[string]any) (any, error) {
	profiles, err := profile.List(paths.ProfilesDir())
	if err != nil {
		return nil, err
	}
	type row struct {
		Name        string `json:"name"`
		Layer       string `json:"layer"`
		Engine      string `json:"engine"`
		Load        string `json:"load"`
		Description string `json:"description"`
	}
	out := make([]row, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, row{Name: p.Name, Layer: p.Layer, Engine: p.Engine, Load: p.Load, Description: p.Description})
	}
	return out, nil
}

func (t *Tools) Plan(ctx context.Context, args map[string]any) (any, error) {
	intent, _ := args["intent"].(string)
	if intent == "" {
		return nil, fmt.Errorf("intent is required")
	}
	useLLM, _ := args["use_llm"].(bool)
	profiles, err := profile.List(paths.ProfilesDir())
	if err != nil {
		return nil, err
	}
	cfg := llm.FromEnv()
	if url, ok := args["llm_url"].(string); ok && url != "" {
		cfg.BaseURL = url
	}
	if model, ok := args["model"].(string); ok && model != "" {
		cfg.Model = model
	}
	if provider, ok := args["llm_provider"].(string); ok && provider != "" {
		cfg.Provider = provider
	}
	res := planner.Plan(ctx, planner.PlanOptions{
		Intent:   intent,
		Profiles: profiles,
		Hardware: discovery.Snapshot(),
		UseLLM:   useLLM,
		LLM:      cfg,
	})
	return res, nil
}

func (t *Tools) Validate(ctx context.Context, args map[string]any) (any, error) {
	name, _ := args["profile"].(string)
	target, _ := args["target"].(string)
	mock, _ := args["mock"].(bool)
	checkHW := true
	if v, ok := args["check_hardware"].(bool); ok {
		checkHW = v
	}
	if name == "" {
		return nil, fmt.Errorf("profile is required")
	}
	p, err := profile.LoadByName(paths.ProfilesDir(), name)
	if err != nil {
		return nil, err
	}
	svc, err := orchestrator.NewService(t.dataDir())
	if err != nil {
		return nil, err
	}
	defer svc.Close()
	return svc.Validate(orchestrator.RunOptions{
		Profile:       p,
		Target:        target,
		Mock:          mock,
		CheckHardware: checkHW,
		DataDir:       t.dataDir(),
	}), nil
}

func (t *Tools) Run(ctx context.Context, args map[string]any) (any, error) {
	name, _ := args["profile"].(string)
	target, _ := args["target"].(string)
	mock := true
	if v, ok := args["mock"].(bool); ok {
		mock = v
	}
	skipValidate, _ := args["skip_validate"].(bool)
	checkHW := true
	if v, ok := args["check_hardware"].(bool); ok {
		checkHW = v
	}
	if name == "" {
		return nil, fmt.Errorf("profile is required")
	}
	if target == "" {
		return nil, fmt.Errorf("target is required")
	}
	p, err := profile.LoadByName(paths.ProfilesDir(), name)
	if err != nil {
		return nil, err
	}
	svc, err := orchestrator.NewService(t.dataDir())
	if err != nil {
		return nil, err
	}
	defer svc.Close()
	run, err := svc.Run(ctx, orchestrator.RunOptions{
		Profile:       p,
		Target:        target,
		Mock:          mock,
		SkipValidate:  skipValidate,
		CheckHardware: checkHW,
		DataDir:       t.dataDir(),
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"run_id":  run.RunID,
		"profile": run.Profile,
		"status":  run.Status,
		"iops":    run.Results.IOPS,
		"p99_us":  run.Results.LatencyUS.P99,
		"mock":    run.Mock,
	}, nil
}

func (t *Tools) Agent(ctx context.Context, args map[string]any) (any, error) {
	intent, _ := args["intent"].(string)
	target, _ := args["target"].(string)
	if intent == "" || target == "" {
		return nil, fmt.Errorf("intent and target are required")
	}
	mock := true
	if v, ok := args["mock"].(bool); ok {
		mock = v
	}
	useLLM, _ := args["use_llm"].(bool)
	cfg := llm.FromEnv()
	res, err := agentloop.Run(ctx, agentloop.Options{
		Intent:        intent,
		Target:        target,
		Mock:          mock,
		UseLLM:        useLLM,
		UseOllama:     useLLM,
		LLM:           cfg,
		CheckHardware: !mock,
		DataDir:       t.dataDir(),
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"run_id":     res.Run.RunID,
		"profile":    res.Run.Profile,
		"plan":       res.Plan,
		"validation": res.Validation,
		"summary":    res.Summary,
		"report":     res.ReportPath,
	}, nil
}

func (t *Tools) Analyze(ctx context.Context, args map[string]any) (any, error) {
	runID, _ := args["run_id"].(string)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}
	svc, err := orchestrator.NewService(t.dataDir())
	if err != nil {
		return nil, err
	}
	defer svc.Close()
	run, err := svc.Store.Get(runID)
	if err != nil {
		return nil, err
	}
	regression := svc.CheckRegression(run)
	insights := analyst.Analyze(run, regression)
	return map[string]any{
		"run_id":   runID,
		"insights": insights,
		"summary":  analyst.SummaryText(run, insights),
	}, nil
}

func (t *Tools) ListRuns(ctx context.Context, args map[string]any) (any, error) {
	limit := 20
	if v, ok := args["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}
	svc, err := orchestrator.NewService(t.dataDir())
	if err != nil {
		return nil, err
	}
	defer svc.Close()
	runs, err := svc.Store.List(limit)
	if err != nil {
		return nil, err
	}
	type row struct {
		RunID   string  `json:"run_id"`
		Profile string  `json:"profile"`
		Status  string  `json:"status"`
		IOPS    float64 `json:"iops"`
		Mock    bool    `json:"mock"`
	}
	out := make([]row, 0, len(runs))
	for _, r := range runs {
		out = append(out, row{RunID: r.RunID, Profile: r.Profile, Status: r.Status, IOPS: r.Results.IOPS, Mock: r.Mock})
	}
	return out, nil
}

// ToolCatalog returns MCP tool definitions.
func ToolCatalog() []ToolDef {
	return []ToolDef{
		{
			Name:        "stratabench_list_profiles",
			Description: "List all StrataBench workload profiles (HDD, NVMe, AFA, S3, VM, app).",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			Name:        "stratabench_plan",
			Description: "Map natural-language benchmark intent to a profile. Set use_llm=true for LLM planner.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"intent":       map[string]any{"type": "string", "description": "What to benchmark, e.g. nvme oltp database"},
					"use_llm":      map[string]any{"type": "boolean", "description": "Use LLM planner (Ollama or OpenAI-compatible)"},
					"llm_provider": map[string]any{"type": "string", "description": "ollama or openai"},
					"llm_url":      map[string]any{"type": "string"},
					"model":        map[string]any{"type": "string"},
				},
				"required": []string{"intent"},
			},
		},
		{
			Name:        "stratabench_validate",
			Description: "Validate workload design and hardware for a profile before running.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"profile":         map[string]any{"type": "string"},
					"target":          map[string]any{"type": "string"},
					"mock":            map[string]any{"type": "boolean"},
					"check_hardware":  map[string]any{"type": "boolean"},
				},
				"required": []string{"profile"},
			},
		},
		{
			Name:        "stratabench_run",
			Description: "Run a benchmark profile. Defaults to mock=true for safe agent use.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"profile":         map[string]any{"type": "string"},
					"target":          map[string]any{"type": "string"},
					"mock":            map[string]any{"type": "boolean"},
					"skip_validate":   map[string]any{"type": "boolean"},
					"check_hardware":  map[string]any{"type": "boolean"},
				},
				"required": []string{"profile", "target"},
			},
		},
		{
			Name:        "stratabench_agent",
			Description: "Full agentic loop: plan → validate → run → analyze → report from natural language intent.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"intent":  map[string]any{"type": "string"},
					"target":  map[string]any{"type": "string"},
					"mock":    map[string]any{"type": "boolean"},
					"use_llm": map[string]any{"type": "boolean"},
				},
				"required": []string{"intent", "target"},
			},
		},
		{
			Name:        "stratabench_analyze",
			Description: "Analyze a completed run and return insights.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"run_id": map[string]any{"type": "string"},
				},
				"required": []string{"run_id"},
			},
		},
		{
			Name:        "stratabench_list_runs",
			Description: "List recent benchmark runs from the local store.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{"type": "number"},
				},
			},
		},
	}
}

type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func (t *Tools) Call(ctx context.Context, name string, args map[string]any) (any, error) {
	switch name {
	case "stratabench_list_profiles":
		return t.ListProfiles(ctx, args)
	case "stratabench_plan":
		return t.Plan(ctx, args)
	case "stratabench_validate":
		return t.Validate(ctx, args)
	case "stratabench_run":
		return t.Run(ctx, args)
	case "stratabench_agent":
		return t.Agent(ctx, args)
	case "stratabench_analyze":
		return t.Analyze(ctx, args)
	case "stratabench_list_runs":
		return t.ListRuns(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool %q", name)
	}
}

// ServeStdio runs the MCP server on stdin/stdout.
func ServeStdio(ctx context.Context, tools *Tools) error {
	if tools == nil {
		tools = &Tools{}
	}
	srv := newServer(tools)
	enc := json.NewEncoder(os.Stdout)
	dec := json.NewDecoder(os.Stdin)
	for {
		var req rpcRequest
		if err := dec.Decode(&req); err != nil {
			return err
		}
		resp := srv.handle(ctx, req)
		if req.ID != nil && resp != nil {
			if err := enc.Encode(resp); err != nil {
				return err
			}
		}
	}
}
