package engine

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestFioRunnerLiveIntervals(t *testing.T) {
	if _, err := exec.LookPath("fio"); err != nil {
		t.Skip("fio not installed")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "fio-live.dat")
	p := &profile.Profile{
		Name:   "ssd-random-4k",
		Engine: "fio",
		Layer:  "block",
		Params: map[string]any{
			"runtime":   3,
			"ramp_time": 0,
			"size":      "64m",
			"iodepth":   4,
			"numjobs":   1,
			"bs":        "4k",
			"rw":        "randread",
			"ioengine":  "psync",
			"direct":    0,
		},
	}

	var samples int
	_, _, err := (&FioRunner{}).Run(context.Background(), RunInput{
		Profile: p,
		Target:  target,
		WorkDir: dir,
		OnInterval: func(_ schema.IntervalSample) {
			samples++
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if samples < 1 {
		t.Fatalf("expected live fio intervals, got %d", samples)
	}
}
