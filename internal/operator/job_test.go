package operator

import (
	"strings"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/manifest"
)

func TestJobName(t *testing.T) {
	if jobName("nvme-oltp") != "bench-nvme-oltp" {
		t.Fatal(jobName("nvme-oltp"))
	}
	long := strings.Repeat("a", 80)
	if len(jobName(long)) > 63 {
		t.Fatalf("too long: %d", len(jobName(long)))
	}
}

func TestBuildJob(t *testing.T) {
	b := &manifest.Benchmark{
		Metadata: manifest.Metadata{Name: "mock-bench", Namespace: "stratabench"},
		Spec:     manifest.BenchmarkSpec{Profile: "nvme-random-oltp", Target: "/dev/null", Mock: true},
	}
	job, err := buildJob(b)
	if err != nil {
		t.Fatal(err)
	}
	if job.Name != "bench-mock-bench" {
		t.Fatalf("name=%s", job.Name)
	}
	args := job.Spec.Template.Spec.Containers[0].Args
	found := false
	for _, a := range args {
		if strings.Contains(a, "--status-out") {
			found = true
		}
	}
	if !found {
		t.Fatalf("args=%v", args)
	}
}

func TestBuildConfigMap(t *testing.T) {
	b := &manifest.Benchmark{
		Metadata: manifest.Metadata{Name: "x", Namespace: "stratabench"},
		Spec:     manifest.BenchmarkSpec{Profile: "nvme-random-oltp", Target: "/dev/null"},
	}
	cm, err := buildConfigMap(b)
	if err != nil {
		t.Fatal(err)
	}
	if cm.Data["benchmark.yaml"] == "" {
		t.Fatal("empty benchmark.yaml")
	}
}

func TestSpecHashStable(t *testing.T) {
	b := &manifest.Benchmark{
		Spec: manifest.BenchmarkSpec{Profile: "a", Target: "/dev/x", Mock: true},
	}
	h1 := specHash(b)
	h2 := specHash(b)
	if h1 != h2 {
		t.Fatalf("%s vs %s", h1, h2)
	}
}
