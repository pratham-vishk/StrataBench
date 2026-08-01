package engine

import "testing"

func TestProbeSBKDrivers(t *testing.T) {
	rep := ProbeSBKDrivers()
	if len(rep.Drivers) != 3 {
		t.Fatalf("drivers=%v", rep.Drivers)
	}
	names := map[string]bool{}
	for _, d := range rep.Drivers {
		names[d.Driver] = true
	}
	for _, want := range []string{"postgresql", "rocksdb", "kafka"} {
		if !names[want] {
			t.Fatalf("missing driver %s", want)
		}
	}
}

func TestRocksDBBenchmark(t *testing.T) {
	if rocksDBBenchmark("randread") != "readrandom" {
		t.Fatal()
	}
	if rocksDBBenchmark("write") != "fillrandom" {
		t.Fatal()
	}
}

func TestParseKafkaOutput(t *testing.T) {
	text := `500000 records sent, 1234.567 MB/sec (1300.123 MB/sec), 12.345 ms avg latency`
	res, err := parseKafkaOutput(text, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if res.ThroughputMBps != 1234.567 {
		t.Fatalf("mbps=%v", res.ThroughputMBps)
	}
	if res.OpsPerSec <= 0 {
		t.Fatalf("ops=%v", res.OpsPerSec)
	}
}
