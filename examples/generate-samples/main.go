// Generate all HTML sample reports (sequential base→candidate benchmark + SBK + S3).
// Run: go run ./examples/generate-samples   or   make samples
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/samples"
)

func main() {
	out := filepath.Join("examples", "sample-report", "output")
	_ = os.MkdirAll(out, 0o755)
	_ = os.MkdirAll(paths.ReportsDir(), 0o755)
	_ = os.MkdirAll(paths.DataDir(), 0o755)

	svc, err := orchestrator.NewService(paths.DataDir())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer svc.Close()

	ctx := context.Background()
	fmt.Println("Running sequential benchmarks (base → candidate, not parallel)...")
	if err := samples.GenerateAll(ctx, svc, out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("\nHTML samples (no Excel):")
	fmt.Println("  base-benchmark.html      — base run (run this first)")
	fmt.Println("  candidate-benchmark.html — candidate run (after base)")
	fmt.Println("  compare-sample.html      — Grafana overlay compare")
	fmt.Println("  sample-result.html       — SBK import (if CSV present)")
	fmt.Println("  s3-put-sample.html       — S3 PUT operations")
}
