package planner_test

import (
	"context"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/planner"
	"github.com/pratham-vishk/stratabench/internal/profile"
)

func TestPlanKeywordFallback(t *testing.T) {
	profiles := []*profile.Profile{
		{Name: "nvme-random-oltp", Layer: "block", Engine: "fio"},
		{Name: "s3-put-throughput", Layer: "object", Engine: "stratabench"},
	}
	res := planner.Plan(context.Background(), planner.PlanOptions{
		Intent:    "nvme oltp database",
		Profiles:  profiles,
		UseOllama: false,
	})
	if res.Profile != "nvme-random-oltp" {
		t.Fatalf("profile=%s source=%s", res.Profile, res.Source)
	}
	if res.Source != "keyword" {
		t.Fatalf("expected keyword source, got %s", res.Source)
	}
}
