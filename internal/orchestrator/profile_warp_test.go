package orchestrator

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/profile"
)

func TestApplyWarpClients(t *testing.T) {
	p := &profile.Profile{Name: "s3-cluster-put-get", Engine: "warp", Params: map[string]any{}}
	out := applyWarpClients(p, []string{"10.0.0.1:7761", "10.0.0.2:7761"})
	if len(out.ParamStringSlice("warp_clients")) != 2 {
		t.Fatalf("warp_clients=%v", out.ParamStringSlice("warp_clients"))
	}
	// Original profile unchanged.
	if len(p.ParamStringSlice("warp_clients")) != 0 {
		t.Fatal("mutated original profile")
	}
}

func TestApplyWarpClientsSkipsWhenSet(t *testing.T) {
	p := &profile.Profile{
		Engine: "warp",
		Params: map[string]any{"warp_clients": []string{"existing:7761"}},
	}
	out := applyWarpClients(p, []string{"10.0.0.1:7761"})
	if len(out.ParamStringSlice("warp_clients")) != 1 || out.ParamStringSlice("warp_clients")[0] != "existing:7761" {
		t.Fatalf("warp_clients=%v", out.ParamStringSlice("warp_clients"))
	}
}
