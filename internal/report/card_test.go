package report

import (
	"strings"
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestBuildCardData(t *testing.T) {
	run := &schema.RunResult{
		RunID:   "test-id",
		Profile: "nvme-random-oltp",
		Engine:  "fio",
		Layer:   "block",
		Validation: schema.ValidationResult{Passed: true},
		Results: schema.Results{
			IOPS:           100000,
			ThroughputMBps: 1500,
			LatencyUS:      schema.LatencyUS{P50: 100, P99: 500, P999: 800},
		},
		Workload: schema.Workload{DurationSec: 600},
		Clients: []schema.ClientResult{
			{Host: "client1.lab", Target: "nvme0n1", Results: schema.Results{
				IOPS: 50000, ThroughputMBps: 750,
				LatencyUS: schema.LatencyUS{P50: 110, P99: 520},
			}},
		},
		Targets: []schema.TargetResult{
			{Target: "storage-node.lab", Results: schema.Results{
				IOPS: 100000, ThroughputMBps: 1500,
				LatencyUS: schema.LatencyUS{P50: 95, P99: 480},
			}},
		},
		Timestamps: schema.Timestamps{CompletedAt: time.Now()},
	}
	cd, err := buildCardData(run, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if cd.IOPS != "100000" || !cd.HonestPass {
		t.Fatalf("%+v", cd)
	}
	if len(cd.ChartsJS) == 0 {
		t.Fatal("expected chart json")
	}
	if !strings.Contains(string(cd.ChartsJS), `"charts"`) {
		t.Fatal("missing charts map in payload")
	}
	if cd.ChartCount < 10 {
		t.Fatalf("expected industry chart count, got %d", cd.ChartCount)
	}
}

func TestFormatDelta(t *testing.T) {
	if formatDelta(5.2, true) == "" {
		t.Fatal()
	}
	if deltaClass(-12, true) != "bad" {
		t.Fatal(deltaClass(-12, true))
	}
}

func TestLatencyPercentileSeries(t *testing.T) {
	labels, vals := latencyPercentileSeries(schema.LatencyUS{P50: 100, P99: 500})
	if len(labels) < 2 || len(vals) != len(labels) {
		t.Fatalf("labels=%v vals=%v", labels, vals)
	}
}

func TestAllNodeRows(t *testing.T) {
	run := &schema.RunResult{
		Results: schema.Results{LatencyUS: schema.LatencyUS{P50: 1}},
		Clients: []schema.ClientResult{{Host: "c1", Results: schema.Results{}}},
	}
	rows := AllNodeRows(run)
	if len(rows) != 2 {
		t.Fatalf("got %d rows", len(rows))
	}
}
