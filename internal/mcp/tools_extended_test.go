package mcp

import "testing"

func TestExtendedToolCatalog(t *testing.T) {
	names := map[string]bool{}
	for _, td := range ToolCatalog() {
		if names[td.Name] {
			t.Fatalf("duplicate tool %q", td.Name)
		}
		names[td.Name] = true
	}
	for _, want := range []string{
		"stratabench_compare_runs",
		"stratabench_report",
		"stratabench_baseline_check",
		"stratabench_export_json",
		"stratabench_run_progress",
		"stratabench_import_json",
	} {
		if !names[want] {
			t.Fatalf("missing tool %q", want)
		}
	}
	if len(ToolCatalog()) < 14 {
		t.Fatalf("expected 14+ tools, got %d", len(ToolCatalog()))
	}
}
