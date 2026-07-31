package analyst_test

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/baseline"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestTailLatencyAnomaly(t *testing.T) {
	run := &schema.RunResult{
		Results: schema.Results{
			IOPS: 100000,
			LatencyUS: schema.LatencyUS{P50: 100, P99: 1500},
		},
		Validation: schema.ValidationResult{Passed: true},
	}
	insights := analyst.Analyze(run, nil)
	found := false
	for _, ins := range insights {
		if ins.Type == "anomaly" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tail latency anomaly, got %+v", insights)
	}
}

func TestRegressionInsight(t *testing.T) {
	run := &schema.RunResult{
		Results:    schema.Results{IOPS: 80000, LatencyUS: schema.LatencyUS{P99: 250}},
		Validation: schema.ValidationResult{Passed: true},
	}
	alerts := []baseline.Alert{{Metric: "iops", Message: "IOPS regressed 20%"}}
	insights := analyst.Analyze(run, alerts)
	if len(insights) == 0 || insights[0].Type != "regression" {
		t.Fatalf("expected regression insight, got %+v", insights)
	}
}

func TestClientVariance(t *testing.T) {
	run := &schema.RunResult{
		Results:    schema.Results{IOPS: 100000, LatencyUS: schema.LatencyUS{P50: 200, P99: 400}},
		Validation: schema.ValidationResult{Passed: true},
		Clients: []schema.ClientResult{
			{Host: "a", Results: schema.Results{IOPS: 100000}},
			{Host: "b", Results: schema.Results{IOPS: 50000}},
		},
	}
	insights := analyst.Analyze(run, nil)
	found := false
	for _, ins := range insights {
		if ins.Type == "variance" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected variance insight, got %+v", insights)
	}
}
