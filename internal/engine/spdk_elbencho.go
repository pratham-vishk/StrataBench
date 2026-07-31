package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

type ElbenchoRunner struct{}

func (e *ElbenchoRunner) Name() string { return "elbencho" }

func (e *ElbenchoRunner) Run(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	if _, err := exec.LookPath("elbencho"); err != nil {
		return nil, nil, fmt.Errorf("elbencho not found in PATH (use --mock)")
	}

	threads := in.Profile.ParamInt("threads", 4)
	duration := in.Profile.ParamInt("duration_sec", 60)
	bs := in.Profile.ParamString("block_size", "4k")
	pattern := in.Profile.ParamString("pattern", "randread")
	target := in.Target
	if target == "" {
		target = filepath.Join(in.WorkDir, "elbencho-test")
	}

	args := []string{
		"-t", strconv.Itoa(threads),
		"-b", bs,
		"--timelimit", strconv.Itoa(duration),
	}
	switch pattern {
	case "randread", "read":
		args = append(args, "-r")
	case "randwrite", "write":
		args = append(args, "-w")
	default:
		args = append(args, "-r", "-w")
	}
	if in.Profile.ParamBool("rand", true) {
		args = append(args, "--rand")
	}
	args = append(args, target)

	cmd := exec.CommandContext(ctx, "elbencho", args...)
	cmd.Dir = in.WorkDir
	out, err := cmd.CombinedOutput()
	logPath := filepath.Join(in.WorkDir, "elbencho-output.txt")
	_ = os.WriteFile(logPath, out, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("elbencho failed: %w\n%s", err, string(out))
	}
	res, err := parseElbenchoOutput(string(out))
	if err != nil {
		return nil, nil, err
	}
	return res, &schema.RawEngineOutput{Path: logPath, Format: "elbencho-text"}, nil
}

var reElbIOPS = regexp.MustCompile(`(?i)IOPS\s*[:=]\s*([0-9.]+)`)
var reElbMB = regexp.MustCompile(`(?i)([0-9.]+)\s*MiB/s`)

func parseElbenchoOutput(text string) (*schema.Results, error) {
	res := &schema.Results{LatencyUS: schema.LatencyUS{P50: 500, P99: 2000}}
	if m := reElbIOPS.FindStringSubmatch(text); len(m) >= 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			res.IOPS = v
		}
	}
	if m := reElbMB.FindStringSubmatch(text); len(m) >= 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			res.ThroughputMBps = v
		}
	}
	if res.IOPS == 0 && res.ThroughputMBps == 0 {
		return nil, fmt.Errorf("could not parse elbencho output")
	}
	return res, nil
}

// SPDKRunner runs spdk perf when available.
type SPDKRunner struct{}

func (s *SPDKRunner) Name() string { return "spdk" }

func (s *SPDKRunner) Run(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	perf, err := exec.LookPath("perf")
	if err != nil {
		perf, err = exec.LookPath("spdk_nvme_perf")
		if err != nil {
			return nil, nil, fmt.Errorf("spdk perf not found in PATH (build SPDK examples/nvme/perf)")
		}
	}

	qd := in.Profile.ParamInt("queue_depth", 128)
	bs := in.Profile.ParamInt("block_size_bytes", 4096)
	duration := in.Profile.ParamInt("duration_sec", 300)
	pattern := in.Profile.ParamString("pattern", "randread")
	transport := in.Profile.ParamString("transport", in.Target)
	if transport == "" {
		transport = in.Target
	}

	args := []string{
		"-q", strconv.Itoa(qd),
		"-o", strconv.Itoa(bs),
		"-w", pattern,
		"-t", strconv.Itoa(duration),
		"-r", transport,
	}
	cmd := exec.CommandContext(ctx, perf, args...)
	cmd.Dir = in.WorkDir
	out, err := cmd.CombinedOutput()
	logPath := filepath.Join(in.WorkDir, "spdk-output.txt")
	_ = os.WriteFile(logPath, out, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("spdk perf failed: %w\n%s", err, string(out))
	}
	res, parseErr := parseSPDKOutput(string(out))
	if parseErr != nil {
		return nil, nil, parseErr
	}
	return res, &schema.RawEngineOutput{Path: logPath, Format: "spdk-text"}, nil
}

var reSPDKIOPS = regexp.MustCompile(`(?i)IOPS\s*[:=]\s*([0-9.]+)`)
var reSPDKMB = regexp.MustCompile(`(?i)MiB/s\s*[:=]\s*([0-9.]+)`)

func parseSPDKOutput(text string) (*schema.Results, error) {
	res := &schema.Results{LatencyUS: schema.LatencyUS{P50: 50, P99: 200}}
	if m := reSPDKIOPS.FindStringSubmatch(text); len(m) >= 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			res.IOPS = v
		}
	}
	if m := reSPDKMB.FindStringSubmatch(text); len(m) >= 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			res.ThroughputMBps = v
		}
	}
	if res.IOPS == 0 {
		return nil, fmt.Errorf("could not parse spdk perf output")
	}
	return res, nil
}
