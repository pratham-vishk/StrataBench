package engine

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

type WarpRunner struct{}

func (w *WarpRunner) Name() string { return "warp" }

func (w *WarpRunner) Run(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	if _, err := exec.LookPath("warp"); err != nil {
		return nil, nil, fmt.Errorf("warp not found in PATH (install MinIO warp or use --mock)")
	}

	op := in.Profile.ParamString("operation", "put")
	duration := in.Profile.ParamInt("duration_sec", 60)
	concurrent := in.Profile.ParamInt("concurrent", 32)
	objSize := in.Profile.ParamString("object_size", "4MiB")
	bucket := in.Profile.ParamString("bucket", "stratabench")
	host := in.Target
	if host == "" {
		host = in.Profile.ParamString("host", "127.0.0.1:9000")
	}

	accessKey := envOr("WARP_ACCESS_KEY", "minioadmin")
	secretKey := envOr("WARP_SECRET_KEY", "minioadmin")

	benchPrefix := warpBenchPrefix(in.WorkDir)

	args := []string{
		op,
		"--host", host,
		"--access-key", accessKey,
		"--secret-key", secretKey,
		"--duration", fmt.Sprintf("%ds", duration),
		"--concurrent", fmt.Sprintf("%d", concurrent),
		"--obj.size", objSize,
		"--bucket", bucket,
		"--benchdata", benchPrefix,
	}

	for _, client := range in.Profile.ParamStringSlice("warp_clients") {
		args = append(args, "--warp-client", client)
	}
	if rdma := in.Profile.ParamString("rdma", ""); rdma != "" {
		args = append(args, "--rdma="+rdma)
	}

	cmd := exec.CommandContext(ctx, "warp", args...)
	cmd.Dir = in.WorkDir
	logPath := filepath.Join(in.WorkDir, "warp-output.txt")

	var out []byte
	if in.OnInterval != nil {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, nil, err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return nil, nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		tailCtx, cancel := context.WithCancel(ctx)
		var outBuf bytes.Buffer
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			scanWarpStream(tailCtx, stdout, in.OnInterval, &outBuf)
		}()
		go func() {
			defer wg.Done()
			scanWarpStream(tailCtx, stderr, in.OnInterval, &outBuf)
		}()
		waitErr := cmd.Wait()
		cancel()
		wg.Wait()
		out = outBuf.Bytes()
		if waitErr != nil {
			_ = os.WriteFile(logPath, out, 0o644)
			return nil, nil, fmt.Errorf("warp failed: %w\n%s", waitErr, string(out))
		}
	} else {
		combined, err := cmd.CombinedOutput()
		if err != nil {
			_ = os.WriteFile(logPath, combined, 0o644)
			return nil, nil, fmt.Errorf("warp failed: %w\n%s", err, string(combined))
		}
		out = combined
	}
	_ = os.WriteFile(logPath, out, 0o644)

	res, parseErr := parseWarpOutput(string(out), duration)
	if parseErr != nil {
		if in.Mock {
			res = &schema.Results{
				OpsPerSec:      float64(concurrent) * 10,
				ThroughputMBps: float64(concurrent) * 2.5,
				LatencyUS:      schema.LatencyUS{P50: 5000, P99: 15000},
			}
		} else {
			return nil, nil, fmt.Errorf("parse warp output: %w (see %s)", parseErr, logPath)
		}
	}

	attachWarpIntervals(ctx, in.WorkDir, benchPrefix, duration, res)
	return res, &schema.RawEngineOutput{Path: logPath, Format: "warp-text"}, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

var (
	reThroughput = regexp.MustCompile(`(?i)(?:throughput|avg.*?)\s*[:=]?\s*([0-9.]+)\s*([KMGT]?B/s)`)
	reOPS        = regexp.MustCompile(`(?i)(?:operations/s|ops/s|obj/s)\s*[:=]?\s*([0-9.]+)`)
)

func parseWarpOutput(text string, durationSec int) (*schema.Results, error) {
	res := &schema.Results{LatencyUS: schema.LatencyUS{P50: 3000, P99: 12000}}
	if m := reOPS.FindStringSubmatch(text); len(m) >= 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			res.OpsPerSec = v
			res.IOPS = v
		}
	}
	if m := reThroughput.FindStringSubmatch(text); len(m) >= 3 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			mult := 1.0
			switch strings.ToUpper(m[2]) {
			case "KB/S":
				mult = 1.0 / 1024
			case "MB/S":
				mult = 1.0
			case "GB/S":
				mult = 1024.0
			}
			res.ThroughputMBps = v * mult
		}
	}
	if res.OpsPerSec == 0 && res.ThroughputMBps == 0 {
		return nil, fmt.Errorf("could not parse warp output")
	}
	res.TotalOperations = int64(res.OpsPerSec * float64(durationSec))
	return res, nil
}
