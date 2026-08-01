package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

var reElbLiveIOPS = regexp.MustCompile(`(?i)IOPS\s*[:=]\s*([0-9.]+)`)
var reElbLiveMB = regexp.MustCompile(`(?i)([0-9.]+)\s*MiB/s`)

func scanElbenchoStream(ctx context.Context, r io.Reader, onInterval func(schema.IntervalSample), acc *bytes.Buffer) {
	if r == nil {
		return
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	seq := 0
	var csvHeader []string
	for sc.Scan() {
		line := sc.Text()
		if acc != nil {
			acc.WriteString(line)
			acc.WriteByte('\n')
		}
		if onInterval != nil {
			if strings.Contains(line, ",") {
				if sample, hdr, ok := parseElbenchoCSVLine(line, csvHeader); ok {
					if len(hdr) > 0 && len(csvHeader) == 0 {
						csvHeader = hdr
					} else if sample.IOPS > 0 || sample.ThroughputMBps > 0 {
						seq++
						sample.Seq = seq
						sample.Timestamp = time.Now().UTC()
						onInterval(sample)
					}
				}
			} else if iops, mbps, ok := parseElbenchoLiveLine(line); ok {
				seq++
				onInterval(schema.IntervalSample{
					Seq:            seq,
					Timestamp:      time.Now().UTC(),
					IOPS:           iops,
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

func parseElbenchoCSVLine(line string, header []string) (schema.IntervalSample, []string, bool) {
	rec, err := csv.NewReader(strings.NewReader(line)).Read()
	if err != nil || len(rec) < 2 {
		return schema.IntervalSample{}, header, false
	}
	if len(header) == 0 && looksLikeElbenchoCSVHeader(rec) {
		return schema.IntervalSample{}, rec, true
	}
	if len(header) == 0 {
		return schema.IntervalSample{}, header, false
	}
	col := map[string]int{}
	for i, h := range header {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	iops := elbenchoCSVFloat(rec, col, "iops", "ops/s", "ops")
	mbps := elbenchoCSVFloat(rec, col, "mib/s", "mb/s", "throughput", "bw")
	latMs := elbenchoCSVFloat(rec, col, "latency", "lat ms", "lat")
	if iops == 0 && mbps == 0 {
		return schema.IntervalSample{}, header, false
	}
	return schema.IntervalSample{
		ElapsedSec:     1,
		IOPS:           iops,
		ThroughputMBps: mbps,
		AvgLatencyUS:   latMs * 1000,
	}, header, true
}

func looksLikeElbenchoCSVHeader(rec []string) bool {
	for _, f := range rec {
		lf := strings.ToLower(strings.TrimSpace(f))
		if lf == "iops" || lf == "mib/s" || lf == "phase" || lf == "mixtype" {
			return true
		}
	}
	return false
}

func elbenchoCSVFloat(rec []string, col map[string]int, names ...string) float64 {
	for _, name := range names {
		if i, ok := col[name]; ok && i < len(rec) {
			if v, err := strconv.ParseFloat(strings.TrimSpace(rec[i]), 64); err == nil && v > 0 {
				return v
			}
		}
	}
	return 0
}

func parseElbenchoLiveLine(line string) (iops, mbps float64, ok bool) {
	if m := reElbLiveIOPS.FindStringSubmatch(line); len(m) >= 2 {
		iops, _ = strconv.ParseFloat(m[1], 64)
		ok = true
	}
	if m := reElbLiveMB.FindStringSubmatch(line); len(m) >= 2 {
		mbps, _ = strconv.ParseFloat(m[1], 64)
		ok = true
	}
	return iops, mbps, ok
}
