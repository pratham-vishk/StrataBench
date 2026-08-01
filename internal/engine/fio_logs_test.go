package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestParseFioLogFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stratabench_iops.1.log")
	content := `5000, 120000, 0, 4096
10000, 118500, 0, 4096
15000, 121200, 0, 4096
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	m := parseFioLogFile(path)
	if len(m) != 3 || m[1] != 120000 {
		t.Fatalf("%v", m)
	}
}

func TestParseFioLogIntervals(t *testing.T) {
	dir := t.TempDir()
	iops := `5000, 100000, 0, 4096
10000, 110000, 0, 4096
`
	bw := `5000, 524288000, 0, 4096
10000, 576716800, 0, 4096
`
	_ = os.WriteFile(filepath.Join(dir, "stratabench_iops.1.log"), []byte(iops), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "stratabench_bw.1.log"), []byte(bw), 0o644)

	iv := parseFioLogIntervals(dir, "stratabench")
	if len(iv) != 2 {
		t.Fatalf("got %d intervals", len(iv))
	}
	if iv[0].IOPS != 100000 {
		t.Fatalf("iops=%v", iv[0].IOPS)
	}
	if iv[0].ThroughputMBps < 400 {
		t.Fatalf("mbps=%v", iv[0].ThroughputMBps)
	}
}

func TestWatchFioLogIntervals(t *testing.T) {
	dir := t.TempDir()
	iopsPath := filepath.Join(dir, "stratabench_iops.1.log")
	bwPath := filepath.Join(dir, "stratabench_bw.1.log")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var samples []schema.IntervalSample
	done := make(chan struct{})
	go func() {
		defer close(done)
		watchFioLogIntervals(ctx, dir, "stratabench", func(s schema.IntervalSample) {
			samples = append(samples, s)
		})
	}()

	if err := os.WriteFile(iopsPath, []byte("5000, 100000, 0, 4096\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bwPath, []byte("5000, 524288000, 0, 4096\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(samples) >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(samples) < 1 {
		t.Fatalf("expected at least 1 live sample, got %d", len(samples))
	}

	f, err := os.OpenFile(iopsPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("10000, 110000, 0, 4096\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if f, err := os.OpenFile(bwPath, os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
		_, _ = f.WriteString("10000, 576716800, 0, 4096\n")
		f.Close()
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(samples) >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	cancel()
	<-done

	if len(samples) < 2 {
		t.Fatalf("expected 2 live samples, got %d", len(samples))
	}
	if samples[0].IOPS != 100000 || samples[1].IOPS != 110000 {
		t.Fatalf("samples=%+v", samples)
	}
}
