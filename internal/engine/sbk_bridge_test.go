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
)

func TestSBKBridgePythonScript(t *testing.T) {
	python := "python3"
	if runtime.GOOS == "windows" {
		python = "python"
	}
	if _, err := exec.LookPath(python); err != nil {
		t.Skipf("%s not in PATH", python)
	}
	if err := exec.Command(python, "--version").Run(); err != nil {
		t.Skipf("%s not runnable: %v", python, err)
	}

	root := filepath.Join("..", "..")
	script := filepath.Join(root, "examples", "sbk-bridge", "sbk_bridge.py")
	if _, err := os.Stat(script); err != nil {
		t.Skip(script)
	}
	t.Setenv("STRATABENCH_SBK_BRIDGE", script)
	t.Setenv("STRATABENCH_PYTHON", python)

	p := &profile.Profile{
		Name:   "custom-sbk", Engine: "sbk", Layer: "application",
		Params: map[string]any{"driver": "custom", "duration_sec": 10, "threads": 2},
	}
	r := &engine.SBKRunner{}
	res, raw, err := r.Run(context.Background(), engine.RunInput{
		Profile: p, Target: "localhost", WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IOPS <= 0 || raw.Format != "sbk-bridge-json" {
		t.Fatalf("res=%+v raw=%+v", res, raw)
	}
}
