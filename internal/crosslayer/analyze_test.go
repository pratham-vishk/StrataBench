package crosslayer

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestAnalyzeLatencyBottleneck(t *testing.T) {
	block := &schema.RunResult{
		Layer: "block", Profile: "nvme-random-oltp",
		Results: schema.Results{IOPS: 1e6, LatencyUS: schema.LatencyUS{P99: 100}},
	}
	object := &schema.RunResult{
		Layer: "object", Profile: "s3-put-throughput",
		Results: schema.Results{IOPS: 50000, LatencyUS: schema.LatencyUS{P99: 5000}},
	}
	insights := Analyze([]*schema.RunResult{block, object})
	if len(insights) == 0 {
		t.Fatal("expected bottleneck insight")
	}
	found := false
	for _, ins := range insights {
		if ins.Type == "bottleneck" {
			found = true
		}
	}
	if !found {
		t.Fatalf("insights=%v", insights)
	}
}

func TestAnalyzeThroughputGap(t *testing.T) {
	block := &schema.RunResult{
		Layer: "block",
		Results: schema.Results{IOPS: 1e6},
	}
	app := &schema.RunResult{
		Layer: "application",
		Results: schema.Results{IOPS: 1000},
	}
	insights := Analyze([]*schema.RunResult{block, app})
	found := false
	for _, ins := range insights {
		if ins.Type == "throughput_gap" {
			found = true
		}
	}
	if !found {
		t.Fatalf("insights=%v", insights)
	}
}

func TestParseProfilesCSV(t *testing.T) {
	got := ParseProfilesCSV(" a , b ,, c ")
	if len(got) != 3 || got[0] != "a" {
		t.Fatalf("%v", got)
	}
}
