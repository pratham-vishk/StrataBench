package planner

import (
	"strings"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/profile"
)

func TestGuideLayerConfusion(t *testing.T) {
	p := &profile.Profile{Name: "nvme-random-oltp", Engine: "fio", Layer: "block"}
	plan := PlanResult{
		Profile: "nvme-random-oltp",
		Params:  map[string]any{"object_size_min": "3KiB", "object_size_max": "100KiB"},
	}
	g := Guide(plan, "nvme object size 3kb-100kb", p)
	if g.Ready {
		t.Fatal("expected not ready due to layer question")
	}
	if len(g.Questions) == 0 {
		t.Fatal("expected layer question")
	}
}

func TestGuideMissingTarget(t *testing.T) {
	p := &profile.Profile{Name: "s3-put-throughput", Engine: "warp", Layer: "object"}
	plan := PlanResult{Profile: "s3-put-throughput", Params: map[string]any{"duration_sec": 3600}}
	g := Guide(plan, "s3 put 1 hour", p)
	if g.Ready {
		t.Fatal("expected missing target question")
	}
}

func TestGuideReadyWithTarget(t *testing.T) {
	p := &profile.Profile{Name: "nvme-random-oltp", Engine: "fio", Layer: "block", Params: map[string]any{"runtime": 600}}
	p.Validation.MinRuntimeSec = 600
	plan := PlanResult{
		Profile: "nvme-random-oltp",
		Target:  "/dev/nvme0n1",
		Params:  map[string]any{"runtime": 3600, "iodepth": 32},
	}
	g := Guide(plan, "nvme oltp /dev/nvme0n1 1 hour", p)
	if !g.Ready {
		t.Fatalf("expected ready, questions=%v", g.Questions)
	}
}

func TestEngineCatalogAllEngines(t *testing.T) {
	for _, e := range []string{"fio", "vdbench", "spdk", "elbencho", "warp", "sbk"} {
		if len(EngineCatalog(e)) == 0 {
			t.Fatalf("no catalog for %s", e)
		}
	}
}

func TestFormatGuidance(t *testing.T) {
	out := FormatGuidance(Guidance{
		Summary: "test",
		Questions: []GuideItem{{Kind: "question", Topic: "target", Message: "need device"}},
	})
	if !strings.Contains(out, "need device") {
		t.Fatal(out)
	}
}
