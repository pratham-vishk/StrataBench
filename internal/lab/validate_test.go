package lab

import (
	"os"
	"path/filepath"
	"testing"
)

func testProfilesDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "profiles")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("profiles dir: %v", err)
	}
	return dir
}

func TestValidationMatrix(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Clients = []Node{{Host: "10.0.1.1", Port: 7777}}
	cfg.Servers = []Node{{Host: "10.0.1.10", Port: 9000}}
	items, err := ValidationMatrix(cfg, testProfilesDir(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) < 33 {
		t.Fatalf("expected >=33 matrix rows, got %d", len(items))
	}
	foundNative := false
	profileNames := map[string]bool{}
	for _, it := range items {
		if it.Section == "monitoring" {
			continue
		}
		profileNames[it.Profile] = true
		if it.Profile == "block-native-io_uring" {
			foundNative = true
			if it.Engine != "stratabench" {
				t.Fatalf("engine=%s", it.Engine)
			}
		}
		if it.Command == "" || it.Target == "" {
			t.Fatalf("empty command/target for %s", it.Profile)
		}
	}
	if !foundNative {
		t.Fatal("missing block-native-io_uring in matrix")
	}
	if !profileNames["s3-gosbench-read"] {
		t.Fatal("missing s3-gosbench-read in matrix")
	}
}

func TestBlockTargetDefault(t *testing.T) {
	cfg := Config{}
	if cfg.BlockTarget() != "/dev/nvme0n1" {
		t.Fatalf("default block target=%s", cfg.BlockTarget())
	}
}

func TestValidationMatrixJSON(t *testing.T) {
	cfg := DefaultConfig()
	items, err := ValidationMatrix(cfg, testProfilesDir(t))
	if err != nil {
		t.Fatal(err)
	}
	rep := &ValidateReport{Items: items, ProfileCount: len(items) - 1}
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")
	if err := WriteValidationReportJSON(path, rep); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultValidationOutput(t *testing.T) {
	if got := DefaultValidationOutput("lab.yaml"); got != "lab-validation.json" {
		t.Fatalf("got %s", got)
	}
}
