package metrics

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestPercentileKeyFromSBKColumn(t *testing.T) {
	if got := PercentileKeyFromSBKColumn("Percentile_99.9"); got != "p99.9" {
		t.Fatalf("got %q", got)
	}
	if got := PercentileCountKeyFromSBKColumn("Percentile_Count_50"); got != "p50" {
		t.Fatalf("got %q", got)
	}
}

func TestPercentileSeriesExtended(t *testing.T) {
	res := schema.Results{
		Percentiles: map[string]float64{"p5": 10, "p50": 100, "p99": 500},
	}
	labels, vals := PercentileSeries(res)
	if len(labels) != 3 || vals[1] != 100 {
		t.Fatalf("%v %v", labels, vals)
	}
}

func TestLatencyUnitScale(t *testing.T) {
	if LatencyUnitScale("NANOSECONDS") != 0.001 {
		t.Fatal()
	}
}
