package report

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestDisplayEngineHidesSBK(t *testing.T) {
	if got := displayEngine("sbk"); got != "" {
		t.Fatalf("sbk should be hidden, got %q", got)
	}
	if got := displayEngine("fio"); got != "FIO" {
		t.Fatalf("fio = %q", got)
	}
}

func TestBenchmarkLabel(t *testing.T) {
	run := &schema.RunResult{Engine: "sbk", Layer: "application"}
	if got := benchmarkLabel(run); got != "Application" {
		t.Fatalf("got %q", got)
	}
	run = &schema.RunResult{Engine: "fio", Layer: "block"}
	if got := benchmarkLabel(run); got != "Block / FIO" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildCardDataHidesSBK(t *testing.T) {
	run := &schema.RunResult{
		RunID: "r1", Profile: "Reading", Engine: "sbk", Layer: "application",
		Validation: schema.ValidationResult{Passed: true},
		Results:    schema.Results{IOPS: 1, LatencyUS: schema.LatencyUS{P99: 1}},
		Workload:   schema.Workload{DurationSec: 60},
	}
	cd, err := buildCardData(run, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if cd.EngineLabel != "" {
		t.Fatalf("EngineLabel = %q", cd.EngineLabel)
	}
	if cd.BenchmarkLabel != "Application" {
		t.Fatalf("BenchmarkLabel = %q", cd.BenchmarkLabel)
	}
}
