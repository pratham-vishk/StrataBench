package report

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestWritePDF(t *testing.T) {
	run := &schema.RunResult{
		RunID:   "pdf-test",
		Profile: "nvme-random-oltp",
		Engine:  "fio",
		Layer:   "block",
		Validation: schema.ValidationResult{Passed: true},
		Results: schema.Results{
			IOPS:           250000,
			ThroughputMBps: 980,
			LatencyUS:      schema.LatencyUS{P50: 120, P95: 450, P99: 800},
			Percentiles:    map[string]float64{"p50": 120, "p99": 800},
		},
		Workload: schema.Workload{DurationSec: 300, Pattern: "randread", BlockSize: "4k", QueueDepth: 32, Threads: 4},
	}
	out := filepath.Join(t.TempDir(), "report.pdf")
	if err := WritePDF(run, out); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() < 1024 {
		t.Fatalf("pdf too small: %d bytes", info.Size())
	}
}
