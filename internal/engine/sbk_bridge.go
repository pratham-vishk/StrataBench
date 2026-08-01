package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// runSBKBridge invokes STRATABENCH_SBK_BRIDGE (Python Storage Benchmark Kit wrapper).
// Contract matches native engine: run --config <json> --output <json>
func runSBKBridge(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	bin := os.Getenv("STRATABENCH_SBK_BRIDGE")
	if bin == "" {
		return nil, nil, fmt.Errorf("STRATABENCH_SBK_BRIDGE not set")
	}
	if strings.HasSuffix(strings.ToLower(bin), ".py") && !filepath.IsAbs(bin) {
		if abs, err := filepath.Abs(bin); err == nil {
			bin = abs
		}
	}

	driver := in.Profile.ParamString("driver", "generic")
	cfg := map[string]any{
		"target":  in.Target,
		"profile": in.Profile.Name,
		"driver":  driver,
		"params":  in.Profile.Params,
	}
	cfgPath := filepath.Join(in.WorkDir, "sbk-bridge-config.json")
	outPath := filepath.Join(in.WorkDir, "sbk-bridge-results.json")
	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(in.WorkDir, 0o755); err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(cfgPath, cfgBytes, 0o644); err != nil {
		return nil, nil, err
	}

	cmd := bridgeCommand(ctx, bin, "run", "--config", cfgPath, "--output", outPath)
	cmd.Dir = in.WorkDir
	out, err := cmd.CombinedOutput()
	logPath := filepath.Join(in.WorkDir, "sbk-bridge.log")
	_ = os.WriteFile(logPath, out, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("sbk bridge failed: %w\n%s", err, string(out))
	}

	rawBytes, err := os.ReadFile(outPath)
	if err != nil {
		return nil, nil, err
	}
	var res schema.Results
	if err := json.Unmarshal(rawBytes, &res); err != nil {
		return nil, nil, fmt.Errorf("parse sbk bridge output: %w", err)
	}
	return &res, &schema.RawEngineOutput{Path: outPath, Format: "sbk-bridge-json"}, nil
}

func bridgeCommand(ctx context.Context, bin string, args ...string) *exec.Cmd {
	if strings.HasSuffix(strings.ToLower(bin), ".py") {
		interp := os.Getenv("STRATABENCH_PYTHON")
		if interp == "" {
			interp = "python"
		}
		return exec.CommandContext(ctx, interp, append([]string{bin}, args...)...)
	}
	return exec.CommandContext(ctx, bin, args...)
}
