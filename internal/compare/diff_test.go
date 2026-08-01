package compare

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestDiffImproved(t *testing.T) {
	base := &schema.RunResult{
		RunID: "aaa", Profile: "p",
		Results: schema.Results{IOPS: 100000, LatencyUS: schema.LatencyUS{P99: 500}},
	}
	head := &schema.RunResult{
		RunID: "bbb", Profile: "p",
		Provenance: schema.Provenance{GitBranch: "feature", GitSHA: "abc1234"},
		Results:    schema.Results{IOPS: 110000, LatencyUS: schema.LatencyUS{P99: 480}},
	}
	d := Diff(base, head)
	if d.HeadLabel == "" {
		t.Fatal("expected head label")
	}
	if d.Metrics[0].Name != "IOPS" {
		t.Fatalf("%+v", d.Metrics[0])
	}
}

func TestPctDelta(t *testing.T) {
	if pctDelta(100, 110) != 10 {
		t.Fatal()
	}
}
