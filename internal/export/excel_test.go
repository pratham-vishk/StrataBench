package export

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestWriteExcel(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	run := &schema.RunResult{
		RunID:   "abc",
		Profile: "s3-put-throughput",
		Engine:  "warp",
		Layer:   "object",
		Topology: "distributed",
		Validation: schema.ValidationResult{Passed: true},
		Target:  schema.Target{Endpoint: "10.0.0.1:9000"},
		Results: schema.Results{
			IOPS:           50000,
			ReadIOPS:       35000,
			WriteIOPS:      15000,
			ThroughputMBps: 200,
			LatencyUS: schema.LatencyUS{
				Min: 80, Mean: 150, P50: 120, P99: 1200, Max: 5000,
			},
		},
		Workload: schema.Workload{DurationSec: 300, Pattern: "put", BlockSize: "4k"},
		Timestamps: schema.Timestamps{StartedAt: now.Add(-5 * time.Minute), CompletedAt: now},
		Clients: []schema.ClientResult{
			{Host: "10.0.0.2:7777", Target: "10.0.0.1:9000", Results: schema.Results{
				IOPS: 25000, ReadIOPS: 20000, WriteIOPS: 5000,
				LatencyUS: schema.LatencyUS{Min: 90, Mean: 160, P99: 1100, Max: 4800},
			}},
		},
		Targets: []schema.TargetResult{},
	}
	path := filepath.Join(dir, "report.xlsx")
	if err := WriteExcel(run, path); err != nil {
		t.Fatal(err)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	wantSheets := []string{
		"Summary", "Durations", "Report", "Latency", "Nodes",
		"Total_Throughput_MB", "Write_Read",
		"Total_Min_Latency", "Total_Avg_Latency", "Total_Max_Latency",
	}
	got := map[string]bool{}
	for _, s := range f.GetSheetList() {
		got[s] = true
	}
	for _, name := range wantSheets {
		if !got[name] {
			t.Fatalf("missing sheet %q, have %v", name, f.GetSheetList())
		}
	}
}

func TestWriteExcelRuns(t *testing.T) {
	dir := t.TempDir()
	runs := []*schema.RunResult{
		{
			RunID: "r1", Profile: "a", Engine: "fio",
			Validation: schema.ValidationResult{Passed: true},
			Target: schema.Target{Device: "/dev/nvme0n1"},
			Results: schema.Results{IOPS: 100, ThroughputMBps: 50, LatencyUS: schema.LatencyUS{P99: 200}},
			Timestamps: schema.Timestamps{CompletedAt: time.Now()},
		},
		{
			RunID: "r2", Profile: "b", Engine: "fio",
			Validation: schema.ValidationResult{Passed: true},
			Target: schema.Target{Device: "/dev/nvme1n1"},
			Results: schema.Results{IOPS: 120, ThroughputMBps: 60, LatencyUS: schema.LatencyUS{P99: 180}},
			Timestamps: schema.Timestamps{CompletedAt: time.Now()},
		},
	}
	path := filepath.Join(dir, "history.xlsx")
	if err := WriteExcelRuns(runs, path); err != nil {
		t.Fatal(err)
	}
}

func TestCollectNodeRows(t *testing.T) {
	run := &schema.RunResult{
		Results: schema.Results{IOPS: 1},
		Clients: []schema.ClientResult{{Host: "c1", Results: schema.Results{IOPS: 2}}},
		Targets: []schema.TargetResult{{Target: "t1", Results: schema.Results{IOPS: 3}}},
	}
	rows := collectNodeRows(run)
	if len(rows) != 3 {
		t.Fatalf("got %d rows", len(rows))
	}
}
