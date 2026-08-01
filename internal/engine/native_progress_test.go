package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestWatchNativeProgress(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.jsonl")
	if err := os.WriteFile(path, []byte(`{"seq":1,"elapsed_sec":1,"iops":1000,"throughput_mbps":4}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var samples []schema.IntervalSample
	done := make(chan struct{})
	go func() {
		defer close(done)
		watchNativeProgress(ctx, path, func(s schema.IntervalSample) {
			samples = append(samples, s)
		})
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(samples) >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(`{"seq":2,"elapsed_sec":1,"iops":1100,"throughput_mbps":4.4}
`)
	f.Close()

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(samples) >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	cancel()
	<-done

	if len(samples) < 2 || samples[1].IOPS != 1100 {
		t.Fatalf("samples=%+v", samples)
	}
}

func TestDrainNativeProgressSkipsDuplicates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.jsonl")
	content := strings.Repeat(`{"seq":1,"iops":100,"throughput_mbps":1}`+"\n", 3)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	emitted := 0
	var n int
	drainNativeProgressFile(path, func(_ schema.IntervalSample) { n++ }, &emitted)
	if n != 1 || emitted != 1 {
		t.Fatalf("n=%d emitted=%d", n, emitted)
	}
}
