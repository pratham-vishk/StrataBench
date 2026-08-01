package export

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/pratham-vishk/stratabench/internal/importsbk"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestWriteExcelSBKImport(t *testing.T) {
	path := filepath.Join("..", "..", ".tmp", "sbk-charts", "samples", "charts", "sbk-file-read.csv")
	if _, err := os.Stat(path); err != nil {
		t.Skip("sbk sample not available")
	}
	runs, err := importsbk.ParseCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "sbk.xlsx")
	if err := WriteExcel(runs[0], out); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(out)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Intervals", "Throughput_MB", "Total_Percentiles_1",
		"Total_Percentiles_Histogram", "Latencies-1",
	}
	got := map[string]bool{}
	for _, s := range f.GetSheetList() {
		got[s] = true
	}
	for _, name := range want {
		if !got[name] {
			t.Fatalf("missing sheet %s, have %v", name, f.GetSheetList())
		}
	}
}

func TestWriteExcelWithIntervals(t *testing.T) {
	now := time.Now().UTC()
	run := &schema.RunResult{
		RunID: "iv", Profile: "test", Engine: "mock",
		Validation: schema.ValidationResult{Passed: true},
		Results: schema.Results{
			IOPS: 1000, ThroughputMBps: 50,
			LatencyUS: schema.LatencyUS{P50: 100, P99: 500},
			Intervals: []schema.IntervalSample{
				{Seq: 1, IOPS: 900, ThroughputMBps: 45, AvgLatencyUS: 95, MinLatencyUS: 40, MaxLatencyUS: 400},
				{Seq: 2, IOPS: 1100, ThroughputMBps: 55, AvgLatencyUS: 105, MinLatencyUS: 42, MaxLatencyUS: 420},
			},
		},
		Timestamps: schema.Timestamps{StartedAt: now, CompletedAt: now},
	}
	out := filepath.Join(t.TempDir(), "iv.xlsx")
	if err := WriteExcel(run, out); err != nil {
		t.Fatal(err)
	}
}
