package engine

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// SBKRunner runs SBK-style application workloads (mock until native SBK driver lands).
type SBKRunner struct{}

func (s *SBKRunner) Name() string { return "sbk" }

func (s *SBKRunner) Run(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	driver := in.Profile.ParamString("driver", "generic")
	threads := in.Profile.ParamInt("threads", in.Profile.ParamInt("connections", 8))
	duration := in.Profile.ParamInt("duration_sec", 300)

	baseIOPS := 25000.0
	p99 := 800.0
	switch driver {
	case "kafka":
		baseIOPS, p99 = 80000, 1200
	case "rocksdb":
		baseIOPS, p99 = 150000, 350
	case "postgresql":
		baseIOPS, p99 = 45000, 2500
	}

	iops := baseIOPS * (0.9 + float64(threads%5)*0.02)
	valueSize := in.Profile.ParamInt("value_size", in.Profile.ParamInt("record_size_bytes", 1024))

	res := &schema.Results{
		IOPS:            iops,
		OpsPerSec:       iops,
		ThroughputMBps:  iops * float64(valueSize) / (1024 * 1024),
		LatencyUS:       schema.LatencyUS{P50: p99 * 0.3, P99: p99, Mean: p99 * 0.4},
		TotalOperations: int64(iops * float64(duration)),
	}
	logPath := filepath.Join(in.WorkDir, "sbk-output.txt")
	_ = os.WriteFile(logPath, []byte("sbk mock driver="+driver), 0o644)
	return res, &schema.RawEngineOutput{Format: "sbk-mock", Path: logPath}, nil
}
