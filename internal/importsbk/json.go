package importsbk

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

type jsonEnvelope struct {
	Runs []*schema.RunResult `json:"runs"`
}

// ParseJSON imports SBK or StrataBench JSON exports into normalized run results.
func ParseJSON(path string) ([]*schema.RunResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseJSONReader(f, path)
}

func ParseJSONReader(r io.Reader, source string) ([]*schema.RunResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	data = trimBOM(data)

	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}
	if _, ok := probe["runs"]; ok {
		var env jsonEnvelope
		if err := json.Unmarshal(data, &env); err == nil && len(env.Runs) > 0 {
			for _, item := range env.Runs {
				normalizeImportedRun(item, source)
			}
			return env.Runs, nil
		}
	}
	if _, ok := probe["results"]; ok {
		var run schema.RunResult
		if err := json.Unmarshal(data, &run); err == nil {
			normalizeImportedRun(&run, source)
			return []*schema.RunResult{&run}, nil
		}
	}
	if _, ok := probe["iops"]; ok {
		var simple map[string]any
		_ = json.Unmarshal(data, &simple)
		run := simpleToRun(simple, source)
		return []*schema.RunResult{&run}, nil
	}

	var runs []*schema.RunResult
	if err := json.Unmarshal(data, &runs); err == nil && len(runs) > 0 {
		for _, item := range runs {
			normalizeImportedRun(item, source)
		}
		return runs, nil
	}

	var run schema.RunResult
	if err := json.Unmarshal(data, &run); err == nil && (run.RunID != "" || run.Profile != "") {
		normalizeImportedRun(&run, source)
		return []*schema.RunResult{&run}, nil
	}

	var simple map[string]any
	if err := json.Unmarshal(data, &simple); err != nil {
		return nil, fmt.Errorf("unsupported json format: %w", err)
	}
	run = simpleToRun(simple, source)
	return []*schema.RunResult{&run}, nil
}

func trimBOM(b []byte) []byte {
	return []byte(strings.TrimPrefix(string(b), "\ufeff"))
}

func normalizeImportedRun(run *schema.RunResult, source string) {
	if run == nil {
		return
	}
	if run.RunID == "" {
		run.RunID = uuid.New().String()
	}
	if run.SchemaVersion == "" {
		run.SchemaVersion = schema.SchemaVersion
	}
	if run.Status == "" {
		run.Status = "completed"
	}
	if run.Engine == "" {
		run.Engine = "sbk"
	}
	if run.Layer == "" {
		run.Layer = "application"
	}
	if run.Target.Metadata == nil {
		run.Target.Metadata = map[string]string{}
	}
	if source != "" {
		run.Target.Metadata["source"] = source
	}
	if run.Validation.RulesChecked == nil {
		run.Validation = schema.ValidationResult{Passed: true, RulesChecked: []string{"import"}}
	}
	now := time.Now().UTC()
	if run.Timestamps.StartedAt.IsZero() {
		run.Timestamps.StartedAt = now
	}
	if run.Timestamps.CompletedAt.IsZero() {
		run.Timestamps.CompletedAt = now
	}
}

func simpleToRun(m map[string]any, source string) schema.RunResult {
	profile := stringField(m, "profile", "type", "action", "driver")
	if profile == "" {
		profile = "sbk-import"
	}
	now := time.Now().UTC()
	run := schema.RunResult{
		SchemaVersion: schema.SchemaVersion,
		RunID:         uuid.New().String(),
		Profile:       profile,
		Layer:         stringField(m, "layer"),
		Engine:        stringField(m, "engine"),
		Status:        "completed",
		Validation:    schema.ValidationResult{Passed: true, RulesChecked: []string{"import"}},
		Target:        schema.Target{Type: "sbk", Metadata: map[string]string{"source": source}},
		Results: schema.Results{
			IOPS:           floatField(m, "iops", "records_per_sec", "records/sec"),
			ThroughputMBps: floatField(m, "throughput_mbps", "mb_per_sec", "mb/sec"),
			OpsPerSec:      floatField(m, "ops_per_sec", "iops"),
			LatencyUS: schema.LatencyUS{
				P50:  floatField(m, "latency_p50", "p50"),
				P95:  floatField(m, "latency_p95", "p95"),
				P99:  floatField(m, "latency_p99", "p99"),
				P999: floatField(m, "latency_p99_9", "p99_9"),
				Mean: floatField(m, "latency_mean", "avg_latency"),
			},
		},
		Timestamps: schema.Timestamps{StartedAt: now, CompletedAt: now},
	}
	if run.Layer == "" {
		run.Layer = "application"
	}
	if run.Engine == "" {
		run.Engine = "sbk"
	}
	return run
}

func stringField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

func floatField(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case json.Number:
			f, _ := n.Float64()
			return f
		}
	}
	return 0
}
