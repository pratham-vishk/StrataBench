package importsbk_test

import (
	"strings"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/importsbk"
)

func TestParseJSONSimple(t *testing.T) {
	raw := `{"profile":"kafka-write","iops":50000,"latency_p99":1200,"throughput_mbps":200}`
	runs, err := importsbk.ParseJSONReader(strings.NewReader(raw), "test.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Results.IOPS != 50000 {
		t.Fatalf("%+v", runs[0])
	}
}

func TestParseJSONRunResult(t *testing.T) {
	raw := `{"run_id":"abc","profile":"nvme-random-oltp","results":{"iops":100000}}`
	runs, err := importsbk.ParseJSONReader(strings.NewReader(raw), "export.json")
	if err != nil {
		t.Fatal(err)
	}
	if runs[0].RunID != "abc" {
		t.Fatalf("%+v", runs[0])
	}
}
