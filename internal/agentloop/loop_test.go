package agentloop_test

import (
	"context"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/agentloop"
)

func TestAgentLoopMockRun(t *testing.T) {
	res, err := agentloop.Run(context.Background(), agentloop.Options{
		Intent:         "nvme random oltp database",
		Target:         "/dev/null",
		Mock:           true,
		SkipValidate:   true,
		AssumeDefaults: true,
		DataDir:        t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Run == nil || res.Run.RunID == "" {
		t.Fatal("expected run result")
	}
	if res.ReportPath == "" {
		t.Fatal("expected report path")
	}
	if res.Plan.Profile == "" {
		t.Fatal("expected plan profile")
	}
}
