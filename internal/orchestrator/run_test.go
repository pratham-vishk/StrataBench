package orchestrator_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/paths"
)

func TestMockRunEndToEnd(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	_ = os.MkdirAll(dataDir, 0o755)
	svc, err := orchestrator.NewService(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	prof, err := profile.LoadByName(paths.ProfilesDir(), "nvme-random-oltp")
	if err != nil {
		t.Fatal(err)
	}
	run, err := svc.Run(context.Background(), orchestrator.RunOptions{
		Profile:      prof,
		Target:       "/dev/null",
		Mock:         true,
		SkipValidate: true,
		DataDir:      dataDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.RunID == "" || run.Results.IOPS <= 0 {
		t.Fatalf("invalid run: %+v", run)
	}
	stored, err := svc.Store.Get(run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Profile != prof.Name {
		t.Fatalf("profile=%s", stored.Profile)
	}
}

func TestMockRunMultiTargetSweep(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	svc, err := orchestrator.NewService(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()

	prof, err := profile.LoadByName(paths.ProfilesDir(), "s3-put-throughput")
	if err != nil {
		t.Fatal(err)
	}
	run, err := svc.Run(context.Background(), orchestrator.RunOptions{
		Profile:      prof,
		Targets:      []string{"10.0.0.1:9000", "10.0.0.2:9000"},
		Topology:     "sweep",
		Mock:         true,
		SkipValidate: true,
		DataDir:      dataDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Targets) != 2 {
		t.Fatalf("targets=%d", len(run.Targets))
	}
	if len(run.Results.Intervals) == 0 {
		t.Fatal("expected merged aggregate intervals")
	}
}
