package manifest

import (
	"path/filepath"
	"testing"
)

func TestWriteReadApplyResult(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "status", "bench.json")
	want := &ApplyResult{RunID: "abc-123", Profile: "nvme-random-oltp", Status: "completed"}
	if err := WriteApplyResult(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadApplyResult(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.RunID != want.RunID || got.Profile != want.Profile {
		t.Fatalf("got %+v", got)
	}
}

func TestBenchmarkToYAML(t *testing.T) {
	b := &Benchmark{
		Metadata: Metadata{Name: "test"},
		Spec:     BenchmarkSpec{Profile: "nvme-random-oltp", Target: "/dev/null", Mock: true},
	}
	data, err := b.ToYAML()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty yaml")
	}
}
