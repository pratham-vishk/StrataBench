package report

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestWorkloadLabelsObject(t *testing.T) {
	run := &schema.RunResult{
		Layer:   "object",
		Profile: "s3-put-throughput",
		Workload: schema.Workload{Pattern: "put"},
	}
	lbl := workloadLabels(run)
	if !lbl.IsObject || lbl.OpsRate != "Ops/s" || lbl.WriteOp != "PUT ops/s" {
		t.Fatalf("object labels: %+v", lbl)
	}
	if lbl.Operation != "put" {
		t.Fatalf("operation=%q", lbl.Operation)
	}
}

func TestWorkloadLabelsBlock(t *testing.T) {
	run := &schema.RunResult{Layer: "block", Profile: "nvme-random-oltp"}
	lbl := workloadLabels(run)
	if lbl.IsObject || lbl.OpsRate != "IOPS" {
		t.Fatalf("block labels: %+v", lbl)
	}
}

func TestObjectChartsAdded(t *testing.T) {
	run := &schema.RunResult{
		Layer: "object", Profile: "s3-mixed-workload", Engine: "warp",
		Workload: schema.Workload{Pattern: "mixed", DurationSec: 60},
		Validation: schema.ValidationResult{Passed: true},
		Results: schema.Results{
			IOPS: 5000, ReadIOPS: 2000, WriteIOPS: 3000, ThroughputMBps: 800,
			LatencyUS: schema.LatencyUS{P50: 5000, P99: 12000},
			Intervals: []schema.IntervalSample{
				{Seq: 1, IOPS: 5000, ReadIOPS: 2000, WriteIOPS: 3000, ThroughputMBps: 800},
			},
		},
	}
	built := buildAllCharts(run)
	found := false
	for _, g := range built.Groups {
		if g.Title == "S3 operations" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected S3 operations chart group")
	}
}
