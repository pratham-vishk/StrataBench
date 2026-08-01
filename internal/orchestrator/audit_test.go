package orchestrator

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestAuditRunWarningsSynthetic(t *testing.T) {
	run := &schema.RunResult{
		Mock:   false,
		Engine: "sbk",
		RawOutput: &schema.RawEngineOutput{Format: "sbk-synthetic"},
	}
	w := auditRunWarnings(run)
	if len(w) != 1 || w[0].Rule != "synthetic_output" {
		t.Fatalf("%v", w)
	}
}

func TestAuditRunWarningsSkipsMock(t *testing.T) {
	run := &schema.RunResult{
		Mock:      true,
		RawOutput: &schema.RawEngineOutput{Format: "mock"},
	}
	if len(auditRunWarnings(run)) != 0 {
		t.Fatal("expected no warnings for mock run")
	}
}
