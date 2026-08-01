package engine

import "testing"

func TestParsePgBenchProgressLine(t *testing.T) {
	tps, latMs, ok := parsePgBenchProgressLine("progress: 2.0 s, 1234.5 tps, lat 1.234 ms")
	if !ok || tps != 1234.5 || latMs != 1.234 {
		t.Fatalf("tps=%v latMs=%v ok=%v", tps, latMs, ok)
	}
}

func TestSbkMockIntervals(t *testing.T) {
	iv := sbkMockIntervals("kafka", 2, 8)
	if len(iv) != 2 || iv[0].IOPS != 76800 {
		t.Fatalf("%+v", iv)
	}
}
