package orchestrator

import (
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// auditRunWarnings flags non-mock runs whose raw engine output looks synthetic or incomplete.
func auditRunWarnings(run *schema.RunResult) []schema.Warning {
	if run == nil || run.Mock {
		return nil
	}
	var warns []schema.Warning
	if run.RawOutput != nil {
		format := strings.ToLower(run.RawOutput.Format)
		if strings.Contains(format, "synthetic") || format == "mock" {
			warns = append(warns, schema.Warning{
				Rule:     "synthetic_output",
				Message:  "engine output format " + run.RawOutput.Format + " indicates synthetic or mock data on a non-mock run",
				Severity: "error",
			})
		}
	}
	return warns
}
