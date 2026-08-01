package engine

import (
	"context"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/profile"
)

func TestSBKRunnerFailsWithoutTool(t *testing.T) {
	p := &profile.Profile{Name: "app-postgres-tpc-c", Engine: "sbk", Params: map[string]any{"driver": "postgresql"}}
	runner := &SBKRunner{}
	_, _, err := runner.Run(context.Background(), RunInput{
		Profile: p,
		Target:  "postgres://localhost/bench",
		WorkDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error when pgbench unavailable")
	}
}

func TestSBKRunnerMockUsesSynthetic(t *testing.T) {
	p := &profile.Profile{Name: "app-postgres-tpc-c", Engine: "sbk", Params: map[string]any{"driver": "postgresql"}}
	runner := &SBKRunner{}
	res, _, err := runner.Run(context.Background(), RunInput{
		Profile: p,
		Mock:    true,
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IOPS <= 0 {
		t.Fatalf("iops=%v", res.IOPS)
	}
}

func TestParseWarpOutputFailsOnGarbage(t *testing.T) {
	_, err := parseWarpOutput("no metrics here", 60)
	if err == nil {
		t.Fatal("expected parse error")
	}
}
