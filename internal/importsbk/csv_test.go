package importsbk_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/importsbk"
)

func TestParseCSVReader(t *testing.T) {
	csv := `Type,IOPS,MB/sec,99.0
NVMe,125000,488,450
`
	runs, err := importsbk.ParseCSVReader(strings.NewReader(csv), "test.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0].Results.IOPS != 125000 {
		t.Fatalf("iops=%v", runs[0].Results.IOPS)
	}
	if runs[0].Results.LatencyUS.P99 != 450 {
		t.Fatalf("p99=%v", runs[0].Results.LatencyUS.P99)
	}
}

func TestParseSBKFullCSV(t *testing.T) {
	path := filepath.Join("..", "..", ".tmp", "sbk-charts", "samples", "charts", "sbk-file-read.csv")
	if _, err := os.Stat(path); err != nil {
		t.Skip("sbk sample csv not available:", path)
	}
	runs, err := importsbk.ParseCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	run := runs[0]
	if len(run.Results.Intervals) < 10 {
		t.Fatalf("intervals=%d", len(run.Results.Intervals))
	}
	if len(run.Results.Percentiles) < 20 {
		t.Fatalf("percentiles=%d", len(run.Results.Percentiles))
	}
	if len(run.Results.PercentileCounts) < 20 {
		t.Fatalf("counts=%d", len(run.Results.PercentileCounts))
	}
	if run.Results.IOPS < 1_000_000 {
		t.Fatalf("iops=%v", run.Results.IOPS)
	}
}
