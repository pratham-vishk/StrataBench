package engine_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/engine"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestNativeRunnerWithEngineStub(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "stratabench-engine")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", bin, "../../cmd/stratabench-engine").CombinedOutput(); err != nil {
		t.Skipf("build engine stub: %v\n%s", err, out)
	}
	t.Setenv("STRATABENCH_ENGINE_BIN", bin)

	p := &profile.Profile{
		Name: "nvme-random-oltp", Engine: "stratabench", Layer: "block",
		Params: map[string]any{"duration_sec": 30, "queue_depth": 64, "threads": 4},
	}
	r := engine.ForProfile(p, false)
	res, raw, err := r.Run(context.Background(), engine.RunInput{
		Profile: p, Target: "/dev/nvme0n1", WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IOPS <= 0 || raw == nil {
		t.Fatalf("res=%+v raw=%+v", res, raw)
	}
}

func TestNativeRunnerStubBinary(t *testing.T) {
	bin, err := exec.LookPath("stratabench-engine")
	if err != nil {
		if p := os.Getenv("STRATABENCH_ENGINE_BIN"); p != "" {
			bin = p
		} else {
			t.Skip("stratabench-engine not on PATH")
		}
	}
	work := t.TempDir()
	cfg := filepath.Join(work, "cfg.json")
	out := filepath.Join(work, "out.json")
	_ = os.WriteFile(cfg, []byte(`{"target":"/dev/null","layer":"block","threads":2,"queue_depth":32,"duration_sec":10,"block_size":"4k"}`), 0o644)
	cmd := exec.Command(bin, "run", "--config", cfg, "--output", out)
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("stub run: %v\n%s", err, outBytes)
	}
	data, err := os.ReadFile(out)
	if err != nil || len(data) < 10 {
		t.Fatalf("output=%s err=%v", data, err)
	}
}

func TestNativeRunnerLiveProgress(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "stratabench-engine")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", bin, "../../cmd/stratabench-engine").CombinedOutput(); err != nil {
		t.Skipf("build engine stub: %v\n%s", err, out)
	}
	t.Setenv("STRATABENCH_ENGINE_BIN", bin)

	p := &profile.Profile{
		Name: "block-native-oltp", Engine: "stratabench", Layer: "block",
		Params: map[string]any{"duration_sec": 9, "queue_depth": 32, "threads": 2},
	}
	var samples int
	r := engine.ForProfile(p, false)
	_, _, err := r.Run(context.Background(), engine.RunInput{
		Profile: p, Target: "/dev/nvme0n1", WorkDir: t.TempDir(),
		OnInterval: func(_ schema.IntervalSample) { samples++ },
	})
	if err != nil {
		t.Fatal(err)
	}
	if samples < 2 {
		t.Fatalf("expected live native progress samples, got %d", samples)
	}
}
