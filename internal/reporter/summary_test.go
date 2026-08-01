package reporter_test

import (
	"context"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/reporter"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestSummarizeFallback(t *testing.T) {
	run := &schema.RunResult{
		Profile: "ssd-random-4k",
		Layer:   "block",
		Engine:  "mock",
		Target:  schema.Target{Device: "/dev/sdb"},
		Results: schema.Results{IOPS: 100000, ThroughputMBps: 400, LatencyUS: schema.LatencyUS{P99: 200}},
	}
	text := reporter.Summarize(context.Background(), run, nil, reporter.SummaryOptions{UseOllama: false})
	if text == "" {
		t.Fatal("expected summary")
	}
}

func TestSummarizeWithInsights(t *testing.T) {
	run := &schema.RunResult{
		Profile:    "nvme-random-oltp",
		Layer:      "block",
		Engine:     "fio",
		Target:     schema.Target{Device: "test"},
		Results:    schema.Results{IOPS: 50000, LatencyUS: schema.LatencyUS{P50: 100, P99: 1500}},
		Validation: schema.ValidationResult{Passed: true},
	}
	insights := analyst.Analyze(run, nil)
	text := reporter.Summarize(context.Background(), run, insights, reporter.SummaryOptions{UseOllama: false})
	if text == "" {
		t.Fatal("expected summary")
	}
}
