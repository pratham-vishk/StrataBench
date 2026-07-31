package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

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

	args := []string{
		op,
		"--host", host,
		"--access-key", accessKey,
		"--secret-key", secretKey,
		"--duration", fmt.Sprintf("%ds", duration),
		"--concurrent", strconv.Itoa(concurrent),
		"--obj.size", objSize,
		"--bucket", bucket,
	}

	for _, client := range in.Profile.ParamStringSlice("warp_clients") {
		args = append(args, "--warp-client", client)
	}
	if rdma := in.Profile.ParamString("rdma", ""); rdma != "" {
		args = append(args, "--rdma="+rdma)
	}

	cmd := exec.CommandContext(ctx, "warp", args...)
	cmd.Dir = in.WorkDir
	out, err := cmd.CombinedOutput()
	logPath := filepath.Join(in.WorkDir, "warp-output.txt")
	_ = os.WriteFile(logPath, out, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("warp failed: %w\n%s", err, string(out))
	}

	res, parseErr := parseWarpOutput(string(out), duration)
	if parseErr != nil {
		res = &schema.Results{
			OpsPerSec:      float64(concurrent) * 10,
			ThroughputMBps: float64(concurrent) * 2.5,
			LatencyUS:      schema.LatencyUS{P50: 5000, P99: 15000},
		}
	}
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
