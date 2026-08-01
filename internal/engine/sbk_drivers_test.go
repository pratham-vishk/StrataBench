package engine

import (
	"testing"
)

func TestParsePgBenchOutput(t *testing.T) {
	text := `tps = 12345.67 (including connections establishing)
latency average = 2.580 ms`
	res, err := parsePgBenchOutput(text)
	if err != nil {
		t.Fatal(err)
	}
	if res.IOPS != 12345.67 {
		t.Fatalf("iops=%v", res.IOPS)
	}
}

func TestParseDBBenchOutput(t *testing.T) {
	text := `readrandom   :      10.123 micros/op 98765.43 ops/sec`
	res, err := parseDBBenchOutput(text)
	if err != nil {
		t.Fatal(err)
	}
	if res.IOPS != 98765.43 {
		t.Fatalf("iops=%v", res.IOPS)
	}
}
