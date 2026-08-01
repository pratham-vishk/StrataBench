package lab

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/profile"
)

func TestResolveRunBlockHDD(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Clients = []Node{{Host: "10.0.1.1"}}
	cfg.Targets.Block = "/dev/sdb"
	cfg.Servers = nil
	cfg.S3.Deploy = "skip"

	p := &profile.Profile{Name: "hdd-sequential-read", Layer: "block", Engine: "fio"}
	plan, err := cfg.ResolveRun(p)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Target != "/dev/sdb" {
		t.Fatalf("target=%q", plan.Target)
	}
	if plan.NeedsS3 {
		t.Fatal("HDD should not need S3")
	}
	if plan.Topology != "single" {
		t.Fatalf("topology=%s", plan.Topology)
	}
}

func TestResolveRunS3(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Clients = []Node{{Host: "10.0.1.1"}}
	cfg.Servers = []Node{{Host: "10.0.1.10", Port: 9000}}

	p := &profile.Profile{Name: "s3-put-throughput", Layer: "object", Engine: "warp"}
	plan, err := cfg.ResolveRun(p)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Target != "10.0.1.10:9000" {
		t.Fatalf("target=%q", plan.Target)
	}
	if !plan.NeedsS3 {
		t.Fatal("expected NeedsS3")
	}
}

func TestResolveRunSBK(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Clients = []Node{{Host: "10.0.1.1"}}
	cfg.S3.Deploy = "skip"
	cfg.Targets.PostgresDSN = "postgres://bench@10.0.1.5/db"

	p := &profile.Profile{Name: "app-postgres-tpc-c", Layer: "application", Engine: "sbk"}
	plan, err := cfg.ResolveRun(p)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Target != "postgres://bench@10.0.1.5/db" {
		t.Fatalf("target=%q", plan.Target)
	}
	if plan.NeedsS3 {
		t.Fatal("SBK postgres should not need S3")
	}
}

func TestNeedsMinIO(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Servers = []Node{{Host: "10.0.1.10", Port: 9000}}
	if !cfg.NeedsMinIO() {
		t.Fatal("docker deploy with servers should need minio")
	}
	cfg.S3.Deploy = "skip"
	if cfg.NeedsMinIO() {
		t.Fatal("skip should not need minio")
	}
}
