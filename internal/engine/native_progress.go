package engine

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// watchNativeProgress polls a JSONL progress file written by stratabench-engine.
func watchNativeProgress(ctx context.Context, path string, onInterval func(schema.IntervalSample)) {
	if onInterval == nil || path == "" {
		return
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	emitted := 0
	flush := func() { drainNativeProgressFile(path, onInterval, &emitted) }
	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-ticker.C:
			flush()
		}
	}
}

func drainNativeProgressFile(path string, onInterval func(schema.IntervalSample), emitted *int) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var s schema.IntervalSample
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			continue
		}
		if s.Seq <= *emitted {
			continue
		}
		if s.Timestamp.IsZero() {
			s.Timestamp = time.Now().UTC()
		}
		onInterval(s)
		*emitted = s.Seq
	}
}
