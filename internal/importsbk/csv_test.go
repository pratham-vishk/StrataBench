package importsbk_test

import (
	"strings"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/importsbk"
)

func TestParseCSVReader(t *testing.T) {
	csv := `Type,IOPS,MB/sec,99.0
NVMe,125000,488,450
`
	runs, err := importsbk.ParseCSVReader(strings.NewReader(csv), "test.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0].Results.IOPS != 125000 {
		t.Fatalf("iops=%v", runs[0].Results.IOPS)
	}
	if runs[0].Results.LatencyUS.P99 != 450 {
		t.Fatalf("p99=%v", runs[0].Results.LatencyUS.P99)
	}
}
