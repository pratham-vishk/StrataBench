package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

var reWarpLiveStats = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*MiB/s,\s*(\d+(?:\.\d+)?)\s*obj/s`)

func warpBenchPrefix(workDir string) string {
	return filepath.Join(workDir, "stratabench-warp")
}

func findWarpBenchData(workDir, prefix string) string {
	patterns := []string{
		prefix + "*.csv.zst",
		prefix + "*",
		filepath.Join(workDir, "warp-*-*.csv.zst"),
	}
	for _, pat := range patterns {
		if matches, _ := filepath.Glob(pat); len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

func warpAnalyzeDuration(durationSec int) string {
	if durationSec >= 120 {
		return "5s"
	}
	if durationSec >= 30 {
		return "2s"
	}
	return "1s"
}

func runWarpAnalyzeCSV(ctx context.Context, benchPath, outCSV string, durationSec int) error {
	cmd := exec.CommandContext(ctx, "warp", "analyze", benchPath,
		"--analyze.dur", warpAnalyzeDuration(durationSec),
		"--analyze.out", outCSV,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("warp analyze failed: %w\n%s", err, string(out))
	}
	return nil
}

func parseWarpAnalyzeCSV(path string) ([]schema.IntervalSample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	col := map[string]int{}
	for i, h := range header {
		col[strings.TrimSpace(strings.ToLower(h))] = i
	}

	var out []schema.IntervalSample
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		idx := atoi(rec, col, "index")
		duration := atof(rec, col, "duration_s")
		mbps := atof(rec, col, "mb_per_sec")
		ops := atof(rec, col, "objs_per_sec")
		if ops == 0 {
			ops = atof(rec, col, "ops_ended_per_sec")
		}
		latMs := atof(rec, col, "reqs_ended_avg_ms")
		if idx <= 0 {
			idx = len(out) + 1
		}
		out = append(out, schema.IntervalSample{
			Seq:            idx,
			ElapsedSec:     duration,
			IOPS:           ops,
			ThroughputMBps: mbps,
			AvgLatencyUS:   latMs * 1000,
		})
	}
	return out, nil
}

func atoi(rec []string, col map[string]int, name string) int {
	i, ok := col[name]
	if !ok || i >= len(rec) {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(rec[i]))
	return n
}

func atof(rec []string, col map[string]int, name string) float64 {
	i, ok := col[name]
	if !ok || i >= len(rec) {
		return 0
	}
	v, _ := strconv.ParseFloat(strings.TrimSpace(rec[i]), 64)
	return v
}

func attachWarpIntervals(ctx context.Context, workDir, benchPrefix string, durationSec int, res *schema.Results) {
	if res == nil {
		return
	}
	benchPath := findWarpBenchData(workDir, benchPrefix)
	if benchPath == "" {
		return
	}
	outCSV := filepath.Join(workDir, "warp-intervals.csv")
	if err := runWarpAnalyzeCSV(ctx, benchPath, outCSV, durationSec); err != nil {
		return
	}
	iv, err := parseWarpAnalyzeCSV(outCSV)
	if err != nil || len(iv) == 0 {
		return
	}
	res.Intervals = iv
}

func scanWarpStream(ctx context.Context, r io.Reader, onInterval func(schema.IntervalSample), acc *bytes.Buffer) {
	if r == nil {
		return
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	seq := 0
	for sc.Scan() {
		line := sc.Text()
		if acc != nil {
			acc.WriteString(line)
			acc.WriteByte('\n')
		}
		if onInterval != nil {
			if mbps, ops, ok := parseWarpLiveLine(line); ok {
				seq++
				onInterval(schema.IntervalSample{
					Seq:            seq,
					Timestamp:      time.Now().UTC(),
					IOPS:           ops,
					ThroughputMBps: mbps,
					ElapsedSec:     1,
				})
			}
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func parseWarpLiveLine(line string) (mbps, ops float64, ok bool) {
	if m := reWarpLiveStats.FindStringSubmatch(line); len(m) >= 3 {
		mbps, _ = strconv.ParseFloat(m[1], 64)
		ops, _ = strconv.ParseFloat(m[2], 64)
		return mbps, ops, true
	}
	if m := reOPS.FindStringSubmatch(line); len(m) >= 2 {
		ops, _ = strconv.ParseFloat(m[1], 64)
		return 0, ops, true
	}
	return 0, 0, false
}
