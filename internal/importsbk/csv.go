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

	"github.com/pratham-vishk/stratabench/internal/metrics"
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
	if isSBKFormat(header) {
		run, err := parseSBKFile(header, rows[1:], source)
		if err != nil {
			return nil, err
		}
		return []*schema.RunResult{run}, nil
	}
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

func isSBKFormat(header map[string]int) bool {
	for k := range header {
		if strings.HasPrefix(k, "percentile_") {
			return true
		}
	}
	_, hasType := header["type"]
	_, hasStorage := header["storage"]
	_, hasAction := header["action"]
	return hasType && hasStorage && hasAction
}

func parseSBKFile(header map[string]int, rows [][]string, source string) (*schema.RunResult, error) {
	rowAt := func(row []string, names ...string) string {
		for _, n := range names {
			if i, ok := header[n]; ok && i < len(row) {
				return strings.TrimSpace(row[i])
			}
		}
		return ""
	}
	rowF := func(row []string, names ...string) float64 {
		v, _ := strconv.ParseFloat(rowAt(row, names...), 64)
		return v
	}
	rowI := func(row []string, names ...string) int64 {
		v, _ := strconv.ParseInt(rowAt(row, names...), 10, 64)
		return v
	}

	var intervals []schema.IntervalSample
	var totalRow []string
	storage, action, unit := "", "", "MICROSECONDS"

	for _, row := range rows {
		typ := strings.ToLower(rowAt(row, "type"))
		if storage == "" {
			storage = rowAt(row, "storage")
			action = rowAt(row, "action")
			unit = rowAt(row, "latencytimeunit", "latency_time_unit")
			if unit == "" {
				unit = "MICROSECONDS"
			}
		}
		if typ == "total" {
			totalRow = row
			continue
		}
		if typ != "" && typ != "regular" {
			continue
		}
		scale := metrics.LatencyUnitScale(unit)
		ts := parseSBKTimestamp(rowAt(row, "date"), rowAt(row, "time"))
		ivPercentiles, _ := parseSBKPercentiles(header, row, scale)
		intervals = append(intervals, schema.IntervalSample{
			Seq:                len(intervals) + 1,
			Timestamp:          ts,
			ElapsedSec:         rowF(row, "reportseconds", "report_seconds"),
			IOPS:               rowF(row, "records_sec", "records_per_sec"),
			ReadIOPS:           rowF(row, "readrequestrecords_sec"),
			WriteIOPS:          rowF(row, "writerequestrecords_sec"),
			ThroughputMBps:     rowF(row, "mb_sec", "mb_per_sec"),
			ReadMBps:           rowF(row, "readrequestmb_sec"),
			WriteMBps:          rowF(row, "writerequestmb_sec"),
			AvgLatencyUS:       rowF(row, "avglatency", "avg_latency") * scale,
			MinLatencyUS:       rowF(row, "minlatency", "min_latency") * scale,
			MaxLatencyUS:       rowF(row, "maxlatency", "max_latency") * scale,
			WriteTimeoutEvents: rowI(row, "writetimeoutevents", "write_timeout_events"),
			ReadTimeoutEvents:  rowI(row, "readtimeoutevents", "read_timeout_events"),
			WriteTimeoutPerSec: rowF(row, "writetimeouteventspersec", "write_timeout_events_per_sec"),
			ReadTimeoutPerSec:  rowF(row, "readtimeouteventspersec", "read_timeout_events_per_sec"),
			Percentiles:        ivPercentiles,
		})
	}
	if totalRow == nil {
		return nil, fmt.Errorf("sbk csv missing Total row")
	}

	scale := metrics.LatencyUnitScale(unit)
	percentiles, counts := parseSBKPercentiles(header, totalRow, scale)
	lat := schema.LatencyUS{
		Min:  rowF(totalRow, "minlatency", "min_latency") * scale,
		Max:  rowF(totalRow, "maxlatency", "max_latency") * scale,
		Mean: rowF(totalRow, "avglatency", "avg_latency") * scale,
	}
	metrics.PopulateLatencyUS(&lat, percentiles)

	profile := action
	if profile == "" {
		profile = storage
	}
	if profile == "" {
		profile = "sbk-import"
	}

	start := time.Now().UTC()
	end := start
	if len(intervals) > 0 && !intervals[0].Timestamp.IsZero() {
		start = intervals[0].Timestamp
		end = intervals[len(intervals)-1].Timestamp
		if intervals[len(intervals)-1].ElapsedSec > 0 {
			end = end.Add(time.Duration(intervals[len(intervals)-1].ElapsedSec) * time.Second)
		}
	}

	run := &schema.RunResult{
		SchemaVersion: schema.SchemaVersion,
		RunID:         uuid.New().String(),
		Profile:       profile,
		Layer:         "application",
		Engine:        "sbk",
		Status:        "completed",
		Validation:    schema.ValidationResult{Passed: true, RulesChecked: []string{"import"}},
		Target: schema.Target{
			Type: "sbk",
			Metadata: map[string]string{
				"source": source, "storage": storage, "action": action, "latency_unit": unit,
			},
		},
		Workload: schema.Workload{
			DurationSec: int(rowF(totalRow, "reportseconds", "report_seconds")),
		},
		Results: schema.Results{
			IOPS:             rowF(totalRow, "records_sec", "records_per_sec"),
			ReadIOPS:         rowF(totalRow, "readrequestrecords_sec"),
			WriteIOPS:        rowF(totalRow, "writerequestrecords_sec"),
			ThroughputMBps:   rowF(totalRow, "mb_sec", "mb_per_sec"),
			OpsPerSec:        rowF(totalRow, "records_sec", "records_per_sec"),
			LatencyUS:        lat,
			Percentiles:      percentiles,
			PercentileCounts: counts,
			Intervals:        intervals,
			TotalOperations:  rowI(totalRow, "records"),
			Totals: schema.TotalStats{
				TotalMB:             rowF(totalRow, "mb"),
				TotalRecords:        rowI(totalRow, "records"),
				WriteRequestMB:      rowF(totalRow, "writerequestmb"),
				WriteRequestRecords: rowI(totalRow, "writerequestrecords"),
				ReadRequestMB:       rowF(totalRow, "readrequestmb"),
				ReadRequestRecords:  rowI(totalRow, "readrequestrecords"),
				WritePendingMB:      rowF(totalRow, "writeresponsependingmb", "writereadrequestpendingmb"),
				WritePendingRecords: rowI(totalRow, "writeresponsependingrecords", "writereadrequestpendingrecords"),
				ReadPendingMB:       rowF(totalRow, "readresponsependingmb"),
				ReadPendingRecords:  rowI(totalRow, "readresponsependingrecords"),
				WriteTimeoutEvents:  rowI(totalRow, "writetimeoutevents", "write_timeout_events"),
				ReadTimeoutEvents:   rowI(totalRow, "readtimeoutevents", "read_timeout_events"),
				InvalidLatencies:    rowI(totalRow, "invalidlatencies"),
				LowerDiscard:        rowI(totalRow, "lowerdiscard"),
				HigherDiscard:       rowI(totalRow, "higherdiscard"),
				SLC1:                rowI(totalRow, "slc1"),
				SLC2:                rowI(totalRow, "slc2"),
			},
		},
		Timestamps: schema.Timestamps{StartedAt: start, CompletedAt: end},
	}
	return run, nil
}

func parseSBKPercentiles(header map[string]int, row []string, scale float64) (map[string]float64, map[string]int64) {
	p := map[string]float64{}
	c := map[string]int64{}
	for col, idx := range header {
		if idx >= len(row) {
			continue
		}
		raw := strings.TrimSpace(row[idx])
		if raw == "" {
			continue
		}
		if key := metrics.PercentileKeyFromSBKColumn(col); key != "" {
			if v, err := strconv.ParseFloat(raw, 64); err == nil {
				p[key] = v * scale
			}
			continue
		}
		if key := metrics.PercentileCountKeyFromSBKColumn(col); key != "" {
			if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
				c[key] = v
			}
		}
	}
	return p, c
}

func parseSBKTimestamp(date, clock string) time.Time {
	if date == "" {
		return time.Time{}
	}
	if clock == "" {
		clock = "00:00:00"
	}
	t, err := time.Parse("2006-01-02 15:04:05", date+" "+clock)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func normalizeHeader(h []string) map[string]int {
	idx := map[string]int{}
	for i, col := range h {
		key := strings.ToLower(strings.TrimSpace(col))
		key = strings.ReplaceAll(key, " ", "_")
		key = strings.ReplaceAll(key, "/", "_")
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
