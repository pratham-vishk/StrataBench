package importsbk

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// ParseCSV imports SBK-style benchmark CSV rows into StrataBench run results.
func ParseCSV(path string) ([]*schema.RunResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseCSVReader(f, path)
}

func ParseCSVReader(r io.Reader, source string) ([]*schema.RunResult, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("csv has no data rows")
	}
	header := normalizeHeader(rows[0])
	var out []*schema.RunResult
	for i, row := range rows[1:] {
		run, err := rowToRun(header, row, source, i)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+2, err)
		}
		out = append(out, run)
	}
	return out, nil
}

func normalizeHeader(h []string) map[string]int {
	idx := map[string]int{}
	for i, col := range h {
		key := strings.ToLower(strings.TrimSpace(col))
		key = strings.ReplaceAll(key, " ", "_")
		idx[key] = i
	}
	return idx
}

func rowToRun(header map[string]int, row []string, source string, rowNum int) (*schema.RunResult, error) {
	get := func(names ...string) string {
		for _, n := range names {
			if i, ok := header[n]; ok && i < len(row) {
				return strings.TrimSpace(row[i])
			}
		}
		return ""
	}
	getF := func(names ...string) float64 {
		v, _ := strconv.ParseFloat(get(names...), 64)
		return v
	}

	profile := get("type", "storage", "driver", "class")
	if profile == "" {
		profile = "sbk-import"
	}
	now := time.Now().UTC()
	run := &schema.RunResult{
		SchemaVersion: schema.SchemaVersion,
		RunID:         uuid.New().String(),
		Profile:       profile,
		Layer:         "application",
		Engine:        "sbk",
		Status:        "completed",
		Validation:    schema.ValidationResult{Passed: true, RulesChecked: []string{"import"}},
		Target:        schema.Target{Type: "sbk", Metadata: map[string]string{"source": source}},
		Results: schema.Results{
			IOPS:           getF("iops", "records/sec", "records_per_sec"),
			ThroughputMBps: getF("mb/sec", "mbps", "throughput_mbps"),
			LatencyUS: schema.LatencyUS{
				P50:  getF("50.0", "p50", "latency_p50"),
				P95:  getF("95.0", "p95", "latency_p95"),
				P99:  getF("99.0", "p99", "latency_p99"),
				P999: getF("99.9", "p99.9", "latency_p99_9"),
				Mean: getF("avg", "average", "latency_avg"),
			},
		},
		Timestamps: schema.Timestamps{StartedAt: now, CompletedAt: now},
	}
	if run.Results.IOPS == 0 {
		run.Results.IOPS = getF("throughput")
	}
	_ = rowNum
	return run, nil
}
