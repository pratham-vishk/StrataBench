package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/manifest"
)

func TestLoadBenchmark(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bench.yaml")
	content := `apiVersion: stratabench.io/v1alpha1
kind: Benchmark
metadata:
  name: test-bench
spec:
  profile: ssd-random-4k
  target: /dev/sdb
  mock: true
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := manifest.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if b.Spec.Profile != "ssd-random-4k" {
		t.Fatalf("profile=%s", b.Spec.Profile)
	}
}
