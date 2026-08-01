package engine

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestParseWarpAnalyzeCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "intervals.csv")
	csv := `index,op,duration_s,mb_per_sec,objs_per_sec,reqs_ended_avg_ms
1,PUT,1,4000,1000,12.5
2,PUT,1,4400,1100,11.2
`
	if err := os.WriteFile(path, []byte(csv), 0o644); err != nil {
		t.Fatal(err)
	}
	iv, err := parseWarpAnalyzeCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(iv) != 2 || iv[0].IOPS != 1000 || iv[1].ThroughputMBps != 4400 {
		t.Fatalf("%+v", iv)
	}
}

func TestParseWarpLiveLine(t *testing.T) {
	mbps, ops, ok := parseWarpLiveLine("* Average: 10.06 MiB/s, 1030.01 obj/s")
	if !ok || mbps != 10.06 || ops != 1030.01 {
		t.Fatalf("mbps=%v ops=%v ok=%v", mbps, ops, ok)
	}
	_, ops, ok = parseWarpLiveLine("operations/s: 2500")
	if !ok || ops != 2500 {
		t.Fatalf("ops=%v ok=%v", ops, ok)
	}
}

func TestScanWarpStreamEmitsIntervals(t *testing.T) {
	input := bytes.NewBufferString("* Average: 5.00 MiB/s, 500.00 obj/s\n* Average: 6.00 MiB/s, 600.00 obj/s\n")
	var samples []schema.IntervalSample
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanWarpStream(ctx, input, func(s schema.IntervalSample) {
			samples = append(samples, s)
		}, nil)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("scan did not finish")
	}
	cancel()
	if len(samples) != 2 || samples[1].IOPS != 600 {
		t.Fatalf("%+v", samples)
	}
}
