package planner_test

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/planner"
)

func TestParseIntentDuration(t *testing.T) {
	p := planner.ParseIntent("benchmark nvme for 1 hour with p99 latency")
	if p.Params["duration_sec"] != 3600 {
		t.Fatalf("duration_sec=%v", p.Params["duration_sec"])
	}
}

func TestParseIntentObjectSizeRange(t *testing.T) {
	p := planner.ParseIntent("s3 object size 3kb-100kb for 30 minutes")
	if p.Params["object_size_min"] != "3KiB" || p.Params["object_size_max"] != "100KiB" {
		t.Fatalf("sizes=%v", p.Params)
	}
	if p.Params["duration_sec"] != 1800 {
		t.Fatalf("duration=%v", p.Params["duration_sec"])
	}
}

func TestParseIntentClientsServers(t *testing.T) {
	text := "clients 10.0.1.1:7777,10.0.1.2:7777 servers 10.0.1.10:9000,10.0.1.11:9000 nvme"
	p := planner.ParseIntent(text)
	if len(p.Clients) != 2 || len(p.Targets) != 2 {
		t.Fatalf("clients=%v targets=%v", p.Clients, p.Targets)
	}
	if p.Topology != "shard" {
		t.Fatalf("topology=%s", p.Topology)
	}
}

func TestParseIntentIodepthThreads(t *testing.T) {
	p := planner.ParseIntent("nvme random iodepth 64 threads 8 ramp 120")
	if p.Params["iodepth"] != 64 || p.Params["numjobs"] != 8 || p.Params["ramp_time"] != 120 {
		t.Fatalf("params=%v", p.Params)
	}
}

func TestParseIntentColocatedSameNode(t *testing.T) {
	text := "s3 rdma clients 10.0.1.10:7777 servers 10.0.1.10:9000 duration 1 hour"
	p := planner.ParseIntent(text)
	if p.Topology != "single" {
		t.Fatalf("topology=%s want single", p.Topology)
	}
	if p.Params["colocated"] != true {
		t.Fatalf("colocated=%v", p.Params["colocated"])
	}
}

func TestParseIntentDeployContext(t *testing.T) {
	p := planner.ParseIntent("virtual vm nvme oltp guest disk")
	if p.Params["deploy_context"] != "virtual" {
		t.Fatalf("deploy_context=%v", p.Params["deploy_context"])
	}
	p2 := planner.ParseIntent("physical bare metal nvme on /dev/nvme0n1")
	if p2.Params["deploy_context"] != "physical" {
		t.Fatalf("deploy_context=%v", p2.Params["deploy_context"])
	}
}

func TestMergePlanCLIOverrides(t *testing.T) {
	base := planner.PlanResult{
		Profile: "s3-put-throughput",
		Target:  "10.0.1.10:9000",
		Params:  map[string]any{"duration_sec": 3600},
	}
	out := planner.MergePlan(base, planner.ParsedIntent{}, "/dev/nvme0n1", nil, []string{"10.0.1.1:7777"}, "pool")
	if out.Target != "/dev/nvme0n1" {
		t.Fatal("cli target should win")
	}
	if len(out.Clients) != 1 || out.Topology != "pool" {
		t.Fatalf("clients=%v topology=%s", out.Clients, out.Topology)
	}
}
