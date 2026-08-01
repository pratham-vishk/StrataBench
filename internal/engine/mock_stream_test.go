package engine

import (
	"context"
	"testing"
	"time"

	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestMockRunnerStreamsIntervals(t *testing.T) {
	p := &profile.Profile{
		Name:   "nvme-random-oltp",
		Engine: "mock",
		Layer:  "block",
		Params: map[string]any{"duration_sec": 15},
	}
	var samples int
	_, _, err := (&MockRunner{}).Run(context.Background(), RunInput{
		Profile: p,
		Mock:    true,
		OnInterval: func(_ schema.IntervalSample) {
			samples++
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if samples < 3 {
		t.Fatalf("expected at least 3 interval callbacks, got %d", samples)
	}
}

func TestStreamMockIntervalsRespectsContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := streamMockIntervals(ctx, RunInput{}, []schema.IntervalSample{
		{Seq: 1}, {Seq: 2}, {Seq: 3}, {Seq: 4},
	}, 2*time.Second)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}
