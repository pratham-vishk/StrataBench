package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/compare"
	"github.com/pratham-vishk/stratabench/internal/export"
	"github.com/pratham-vishk/stratabench/internal/importsbk"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/report"
	"github.com/pratham-vishk/stratabench/internal/runstate"
)

func (t *Tools) CompareRuns(ctx context.Context, args map[string]any) (any, error) {
	runIDA, _ := args["run_id"].(string)
	runIDB, _ := args["run_id_b"].(string)
	if runIDA == "" || runIDB == "" {
		return nil, fmt.Errorf("run_id and run_id_b are required")
	}
	svc, err := orchestrator.NewService(t.dataDir())
	if err != nil {
		return nil, err
	}
	defer svc.Close()
	a, err := svc.Store.Get(runIDA)
	if err != nil {
		return nil, err
	}
	b, err := svc.Store.Get(runIDB)
	if err != nil {
		return nil, err
	}
	diff := compare.Diff(a, b)
	return map[string]any{
		"diff":      diff,
		"regressed": diff.Regressed,
		"improved":  diff.Improved,
		"summary":   diff.Summary,
	}, nil
}

func (t *Tools) Report(ctx context.Context, args map[string]any) (any, error) {
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
	outPath := filepath.Join(paths.ReportsDir(), runID+".html")
	summary := analyst.SummaryText(run, insights)
	if err := report.WriteHTMLWithInsights(run, insights, summary, outPath); err != nil {
		return nil, err
	}
	return map[string]any{
		"run_id":    runID,
		"html_path": outPath,
		"summary":   summary,
		"insights":  insights,
	}, nil
}

func (t *Tools) BaselineCheck(ctx context.Context, args map[string]any) (any, error) {
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
	alerts := svc.CheckRegression(run)
	summary := "No regression vs baseline."
	if len(alerts) > 0 {
		var msgs []string
		for _, a := range alerts {
			msgs = append(msgs, a.Message)
		}
		summary = strings.Join(msgs, "; ")
	}
	return map[string]any{
		"run_id":  runID,
		"alerts":  alerts,
		"passed":  len(alerts) == 0,
		"summary": summary,
	}, nil
}

func (t *Tools) ExportJSON(ctx context.Context, args map[string]any) (any, error) {
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
	outPath := filepath.Join(paths.ReportsDir(), runID+".json")
	if err := export.WriteJSON(run, outPath); err != nil {
		return nil, err
	}
	return map[string]any{"run_id": runID, "json_path": outPath}, nil
}

func (t *Tools) ImportJSON(ctx context.Context, args map[string]any) (any, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	runs, err := importsbk.ParseJSON(path)
	if err != nil {
		return nil, err
	}
	svc, err := orchestrator.NewService(t.dataDir())
	if err != nil {
		return nil, err
	}
	defer svc.Close()
	var imported []string
	for _, run := range runs {
		if err := svc.Store.Save(run); err != nil {
			return nil, err
		}
		imported = append(imported, run.RunID)
	}
	return map[string]any{"imported": imported, "count": len(imported)}, nil
}

func runOptionsFromArgs(t *Tools, args map[string]any) (orchestrator.RunOptions, error) {
	name, _ := args["profile"].(string)
	target, _ := args["target"].(string)
	if name == "" {
		return orchestrator.RunOptions{}, fmt.Errorf("profile is required")
	}
	if target == "" && len(stringSliceArg(args, "targets")) == 0 {
		return orchestrator.RunOptions{}, fmt.Errorf("target is required")
	}
	p, err := profile.LoadByName(paths.ProfilesDir(), name)
	if err != nil {
		return orchestrator.RunOptions{}, err
	}
	mock := true
	if v, ok := args["mock"].(bool); ok {
		mock = v
	}
	skipValidate, _ := args["skip_validate"].(bool)
	checkHW := true
	if v, ok := args["check_hardware"].(bool); ok {
		checkHW = v
	}
	topologyMode, _ := args["topology"].(string)
	return orchestrator.RunOptions{
		Profile:       p,
		Target:        target,
		Targets:       stringSliceArg(args, "targets"),
		Clients:       stringSliceArg(args, "clients"),
		WarpClients:   stringSliceArg(args, "warp_clients"),
		Topology:      topologyMode,
		Mock:          mock,
		SkipValidate:  skipValidate,
		CheckHardware: checkHW,
		DataDir:       t.dataDir(),
	}, nil
}

func (t *Tools) RunProgress(ctx context.Context, args map[string]any) (any, error) {
	runID, _ := args["run_id"].(string)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}
	if p, ok := runstate.Get(runID); ok {
		return p, nil
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
	return map[string]any{
		"run_id": runID,
		"phase":  run.Status,
		"profile": run.Profile,
		"iops":   run.Results.IOPS,
	}, nil
}

// extendedToolCatalog returns MCP tools beyond the core set.
func extendedToolCatalog() []ToolDef {
	return []ToolDef{
		{
			Name:        "stratabench_compare_runs",
			Description: "Compare two completed benchmark runs by run_id.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"run_id":   map[string]any{"type": "string"},
					"run_id_b": map[string]any{"type": "string"},
				},
				"required": []string{"run_id", "run_id_b"},
			},
		},
		{
			Name:        "stratabench_report",
			Description: "Generate HTML report for a completed run.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"run_id": map[string]any{"type": "string"},
				},
				"required": []string{"run_id"},
			},
		},
		{
			Name:        "stratabench_baseline_check",
			Description: "Check a run against stored or rolling baseline regression thresholds.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"run_id": map[string]any{"type": "string"},
				},
				"required": []string{"run_id"},
			},
		},
		{
			Name:        "stratabench_export_json",
			Description: "Export a completed run to normalized JSON.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"run_id": map[string]any{"type": "string"},
				},
				"required": []string{"run_id"},
			},
		},
		{
			Name:        "stratabench_run_progress",
			Description: "Poll in-flight run progress (phase, assignments completed).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"run_id": map[string]any{"type": "string"},
				},
				"required": []string{"run_id"},
			},
		},
		{
			Name:        "stratabench_import_json",
			Description: "Import SBK or StrataBench JSON results into the local store.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string"},
				},
				"required": []string{"path"},
			},
		},
	}
}
