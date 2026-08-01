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

type FioRunner struct{}

func (f *FioRunner) Name() string { return "fio" }

func (f *FioRunner) Run(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	if isVMSSH(in) {
		return runVMFio(ctx, in)
	}
	if _, err := exec.LookPath("fio"); err != nil {
		return nil, nil, fmt.Errorf("fio not found in PATH (install fio or use --mock)")
	}

	jobPath, outPath, err := f.writeJob(in)
	if err != nil {
		return nil, nil, err
	}
	defer os.Remove(jobPath)

	cmd := exec.CommandContext(ctx, "fio", jobPath, "--output-format=json", "--output="+outPath)
	cmd.Dir = in.WorkDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, nil, fmt.Errorf("fio failed: %w\n%s", err, string(out))
	}
	defer os.Remove(outPath)

	raw, err := os.ReadFile(outPath)
	if err != nil {
		return nil, nil, err
	}
	res, err := parseFioJSON(raw)
	if err != nil {
		return nil, nil, err
	}
	return res, &schema.RawEngineOutput{Path: outPath, Format: "fio-json"}, nil
}

func (f *FioRunner) writeJob(in RunInput) (jobPath, outPath string, err error) {
	p := in.Profile
	pattern := p.ParamString("rw", "read")
	bs := p.ParamString("bs", "4k")
	iodepth := p.ParamInt("iodepth", 1)
	numjobs := p.ParamInt("numjobs", 1)
	size := p.ParamString("size", "1g")
	runtime := p.ParamInt("runtime", 60)
	ramp := p.ParamInt("ramp_time", 0)
	direct := p.ParamInt("direct", 1)
	ioengine := p.ParamString("ioengine", "libaio")
	rwmix := p.ParamInt("rwmixread", 70)
	plist := p.ParamString("percentile_list", "50:95:99:99.9")

	filename := in.Target
	if filename == "" {
		filename = filepath.Join(in.WorkDir, "stratabench-fio.dat")
	}

	job := fmt.Sprintf(`[global]
ioengine=%s
direct=%d
runtime=%d
time_based=1
group_reporting=1
filename=%s
ramp_time=%d
percentile_list=%s

[job]
rw=%s
bs=%s
iodepth=%d
numjobs=%d
size=%s
`, ioengine, direct, runtime, filename, ramp, plist, pattern, bs, iodepth, numjobs, size)

	if strings.Contains(pattern, "rw") {
		job += fmt.Sprintf("rwmixread=%d\n", rwmix)
	}

	jobPath = filepath.Join(in.WorkDir, "stratabench.fio")
	outPath = filepath.Join(in.WorkDir, "stratabench-fio-out.json")
	if err := os.WriteFile(jobPath, []byte(job), 0o644); err != nil {
		return "", "", err
	}
	return jobPath, outPath, nil
}

type fioOutput struct {
	Jobs []struct {
		Read struct {
			IOPS      float64 `json:"iops"`
			BWBytes   int64   `json:"bw_bytes"`
			ClatNS    struct {
				Percentile map[string]float64 `json:"percentile"`
				Mean      float64            `json:"mean"`
				Min       float64            `json:"min"`
				Max       float64            `json:"max"`
			} `json:"clat_ns"`
		} `json:"read"`
		Write struct {
			IOPS    float64 `json:"iops"`
			BWBytes int64   `json:"bw_bytes"`
			ClatNS  struct {
				Percentile map[string]float64 `json:"percentile"`
				Mean       float64            `json:"mean"`
				Min        float64            `json:"min"`
				Max        float64            `json:"max"`
			} `json:"clat_ns"`
		} `json:"write"`
	} `json:"jobs"`
}

func parseFioJSON(raw []byte) (*schema.Results, error) {
	var out fioOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if len(out.Jobs) == 0 {
		return nil, fmt.Errorf("fio output has no jobs")
	}
	job := out.Jobs[0]
	readIOPS := job.Read.IOPS
	writeIOPS := job.Write.IOPS
	totalIOPS := readIOPS + writeIOPS
	bw := float64(job.Read.BWBytes+job.Write.BWBytes) / (1024 * 1024)

	clat := job.Read.ClatNS
	if writeIOPS > readIOPS {
		clat = job.Write.ClatNS
	}

	res := &schema.Results{
		IOPS:           totalIOPS,
		ReadIOPS:       readIOPS,
		WriteIOPS:      writeIOPS,
		ThroughputMBps: bw,
		LatencyUS: schema.LatencyUS{
			Min:  clat.Min / 1000,
			Max:  clat.Max / 1000,
			Mean: clat.Mean / 1000,
			P50:  clat.Percentile["50.000000"] / 1000,
			P95:  clat.Percentile["95.000000"] / 1000,
			P99:  clat.Percentile["99.000000"] / 1000,
			P999: clat.Percentile["99.900000"] / 1000,
		},
	}
	return res, nil
}
