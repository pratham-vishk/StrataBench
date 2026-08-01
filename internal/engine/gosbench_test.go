package engine

import (
	"context"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/profile"
)

func TestGOSBenchMockSynthetic(t *testing.T) {
	p := &profile.Profile{
		Name:   "s3-gosbench-write",
		Engine: "gosbench",
		Params: map[string]any{"duration_sec": 30, "workers": 2},
	}
	r := &GosbenchRunner{}
	res, raw, err := r.Run(context.Background(), RunInput{Profile: p, Mock: true, WorkDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if res.IOPS <= 0 || raw.Format != "gosbench-synthetic" {
		t.Fatalf("res=%+v raw=%+v", res, raw)
	}
}

func TestParseGosbenchOutput(t *testing.T) {
	text := `
Test write complete
ops/s: 1250.5
bandwidth: 48.2 MB/s
`
	res, err := parseGosbenchOutput(text, 60)
	if err != nil {
		t.Fatal(err)
	}
	if res.OpsPerSec != 1250.5 || res.ThroughputMBps != 48.2 {
		t.Fatalf("%+v", res)
	}
}

func TestWriteGosbenchConfig(t *testing.T) {
	p := &profile.Profile{
		Name:   "s3-gosbench-write",
		Engine: "gosbench",
		Params: map[string]any{"duration_sec": 90, "workers": 3},
	}
	path, err := writeGosbenchConfig(RunInput{
		Profile: p,
		Target:  "10.0.0.1:9000",
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("empty config path")
	}
}
