package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// NativeRunner invokes an external stratabench-engine binary (Rust implementation).
// Contract: stratabench-engine run --config <json> --output <json>
type NativeRunner struct{}

func (n *NativeRunner) Name() string { return "stratabench" }

type nativeEngineConfig struct {
	Target       string         `json:"target"`
	Profile      string         `json:"profile"`
	Layer        string         `json:"layer"`
	Pattern      string         `json:"pattern"`
	BlockSize    string         `json:"block_size"`
	DatasetSize  string         `json:"dataset_size"`
	DurationSec  int            `json:"duration_sec"`
	RampSec      int            `json:"ramp_sec"`
	QueueDepth   int            `json:"queue_depth"`
	Threads      int            `json:"threads"`
	ReadWriteMix int            `json:"read_write_mix"`
	DirectIO     bool           `json:"direct_io"`
	ProgressPath string         `json:"progress_path,omitempty"`
	Params       map[string]any `json:"params,omitempty"`
}

func (n *NativeRunner) Run(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	bin := resolveNativeEngineBin()
	if bin == "" {
		return nil, nil, fmt.Errorf("native stratabench engine binary not found (set STRATABENCH_ENGINE_BIN or use --mock)")
	}

	pattern, blockSize, datasetSize, durationSec, rampSec, qd, threads, rwMix, directIO := in.Profile.ToWorkload()
	cfgPath := filepath.Join(in.WorkDir, "native-engine-config.json")
	outPath := filepath.Join(in.WorkDir, "native-engine-results.json")
	progressPath := ""
	if in.OnInterval != nil {
		progressPath = filepath.Join(in.WorkDir, "native-engine-progress.jsonl")
		_ = os.Remove(progressPath)
	}
	cfg := nativeEngineConfig{
		Target:       in.Target,
		Profile:      in.Profile.Name,
		Layer:        in.Profile.Layer,
		Pattern:      pattern,
		BlockSize:    blockSize,
		DatasetSize:  datasetSize,
		DurationSec:  durationSec,
		RampSec:      rampSec,
		QueueDepth:   qd,
		Threads:      threads,
		ReadWriteMix: rwMix,
		DirectIO:     directIO,
		ProgressPath: progressPath,
		Params:       in.Profile.Params,
	}
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

	cmd := exec.CommandContext(ctx, bin, "run", "--config", cfgPath, "--output", outPath)
	cmd.Dir = in.WorkDir
	logPath := filepath.Join(in.WorkDir, "native-engine.log")

	var out []byte
	if in.OnInterval != nil && progressPath != "" {
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		tailCtx, cancel := context.WithCancel(ctx)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			watchNativeProgress(tailCtx, progressPath, in.OnInterval)
		}()
		waitErr := cmd.Wait()
		cancel()
		wg.Wait()
		if waitErr != nil {
			return nil, nil, fmt.Errorf("stratabench-engine failed: %w", waitErr)
		}
	} else {
		combined, err := cmd.CombinedOutput()
		out = combined
		if err != nil {
			_ = os.WriteFile(logPath, out, 0o644)
			return nil, nil, fmt.Errorf("stratabench-engine failed: %w\n%s", err, string(out))
		}
	}
	_ = os.WriteFile(logPath, out, 0o644)

	rawBytes, err := os.ReadFile(outPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read engine output %s: %w", outPath, err)
	}
	var res schema.Results
	if err := json.Unmarshal(rawBytes, &res); err != nil {
		return nil, nil, fmt.Errorf("parse engine output: %w", err)
	}
	return &res, &schema.RawEngineOutput{Path: outPath, Format: "stratabench-json"}, nil
}

func resolveNativeEngineBin() string {
	if v := os.Getenv("STRATABENCH_ENGINE_BIN"); v != "" {
		return v
	}
	if p, err := exec.LookPath("stratabench-engine"); err == nil {
		return p
	}
	return ""
}
