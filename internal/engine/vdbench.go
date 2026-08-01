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

type VdbenchRunner struct{}

func (v *VdbenchRunner) Name() string { return "vdbench" }

func (v *VdbenchRunner) Run(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	if isVMSSH(in) {
		return runVMVdbench(ctx, in)
	}
	if _, err := exec.LookPath("vdbench"); err != nil {
		return nil, nil, fmt.Errorf("vdbench not found in PATH (install vdbench or use --mock)")
	}

	parmPath, outDir, err := v.writeParmfile(in)
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.CommandContext(ctx, "vdbench", "-f", parmPath, "-o", outDir)
	cmd.Dir = in.WorkDir
	logPath := filepath.Join(in.WorkDir, "vdbench-output.txt")

	var out []byte
	if in.OnInterval != nil {
		out, err = runStreamedCommand(ctx, cmd, in.OnInterval, scanVdbenchStream)
		writeCommandOutput(logPath, out)
		if err != nil {
			return nil, nil, fmt.Errorf("vdbench failed: %w\n%s", err, string(out))
		}
	} else {
		out, err = cmd.CombinedOutput()
		writeCommandOutput(logPath, out)
		if err != nil {
			return nil, nil, fmt.Errorf("vdbench failed: %w\n%s", err, string(out))
		}
	}

	res, parseErr := parseVdbenchOutput(string(out), outDir)
	if parseErr != nil {
		return nil, nil, parseErr
	}
	return res, &schema.RawEngineOutput{Path: logPath, Format: "vdbench-text"}, nil
}

func (v *VdbenchRunner) writeParmfile(in RunInput) (parmPath, outDir string, err error) {
	p := in.Profile
	luns := p.ParamStringSlice("luns")
	if len(luns) == 0 && in.Target != "" {
		for _, part := range strings.Split(in.Target, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				luns = append(luns, part)
			}
		}
	}
	if len(luns) == 0 {
		luns = []string{filepath.Join(in.WorkDir, "vdbench-test.dat")}
	}

	threads := p.ParamInt("threads", 8)
	xfer := p.ParamString("block_size", "4k")
	duration := p.ParamInt("duration_sec", 300)
	warmup := p.ParamInt("warmup_sec", p.ParamInt("ramp_time_sec", 60))
	pattern := p.ParamString("pattern", "randread")
	rdpct, seekpct := vdbenchPattern(pattern, p.ParamInt("read_pct", -1))

	var b strings.Builder
	b.WriteString("messagescan=no\n")
	b.WriteString("data_errors=1\n")
	b.WriteString("hd=default\n")
	for i, lun := range luns {
		fmt.Fprintf(&b, "sd=sd%d,lun=%s,threads=%d,openflags=o_direct\n", i+1, lun, threads)
	}
	fmt.Fprintf(&b, "wd=wd1,sd=sd*,xfersize=%s,rdpct=%d,seekpct=%d\n", xfer, rdpct, seekpct)
	fmt.Fprintf(&b, "rd=rd1,wd=wd*,iorate=max,elapsed=%d,warmup=%d,interval=1\n", duration, warmup)

	parmPath = filepath.Join(in.WorkDir, "vdbench.parm")
	outDir = filepath.Join(in.WorkDir, "vdbench-out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(parmPath, []byte(b.String()), 0o644); err != nil {
		return "", "", err
	}
	return parmPath, outDir, nil
}

func vdbenchPattern(pattern string, readPct int) (rdpct, seekpct int) {
	if readPct >= 0 {
		rdpct = readPct
	} else {
		switch strings.ToLower(pattern) {
		case "randwrite", "write":
			rdpct = 0
		case "readwrite", "mixed":
			rdpct = 50
		default:
			rdpct = 100
		}
	}
	switch strings.ToLower(pattern) {
	case "randread", "randwrite", "random", "mixed", "readwrite":
		seekpct = 100
	default:
		seekpct = 0
	}
	return rdpct, seekpct
}

var (
	reVdbRate = regexp.MustCompile(`(?i)(?:rate|iops)[^\d]*([0-9][0-9.,]*)`)
	reVdbResp = regexp.MustCompile(`(?i)(?:resp|latency)[^\d]*([0-9][0-9.,]*)`)
	reVdbAvg  = regexp.MustCompile(`(?i)avg[^\d]*([0-9][0-9.,]+)[^\d]+([0-9][0-9.,]+)`)
)

func parseVdbenchOutput(stdout, outDir string) (*schema.Results, error) {
	text := stdout
	if entries, err := os.ReadDir(outDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := strings.ToLower(e.Name())
			if strings.Contains(name, "summary") || strings.Contains(name, "totals") || strings.HasSuffix(name, ".txt") {
				if data, err := os.ReadFile(filepath.Join(outDir, e.Name())); err == nil {
					text += "\n" + string(data)
				}
			}
		}
	}

	res := &schema.Results{LatencyUS: schema.LatencyUS{P50: 500, P99: 2000}}
	if m := reVdbAvg.FindStringSubmatch(text); len(m) >= 3 {
		if rate, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64); err == nil {
			res.IOPS = rate
		}
		if respMS, err := strconv.ParseFloat(strings.ReplaceAll(m[2], ",", ""), 64); err == nil {
			res.LatencyUS.Mean = respMS * 1000
			res.LatencyUS.P99 = respMS * 1000 * 2
		}
	}
	if res.IOPS == 0 {
		if m := reVdbRate.FindStringSubmatch(text); len(m) >= 2 {
			if v, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64); err == nil {
				res.IOPS = v
			}
		}
	}
	if res.LatencyUS.Mean == 0 {
		if m := reVdbResp.FindStringSubmatch(text); len(m) >= 2 {
			if v, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64); err == nil {
				res.LatencyUS.Mean = v * 1000
				res.LatencyUS.P99 = v * 1000 * 2
			}
		}
	}
	if res.IOPS == 0 {
		return nil, fmt.Errorf("could not parse vdbench output")
	}
	bs := 4096
	res.ThroughputMBps = res.IOPS * float64(bs) / (1024 * 1024)
	return res, nil
}
