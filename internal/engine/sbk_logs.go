package engine

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

var rePgProgress = regexp.MustCompile(`progress:\s*([0-9.]+)\s*s,\s*([0-9.]+)\s*tps,\s*lat\s*([0-9.]+)\s*ms`)

func scanPgBenchStream(ctx context.Context, r io.Reader, onInterval func(schema.IntervalSample), acc *bytes.Buffer) {
	if r == nil {
		return
	}
	sc := bufio.NewScanner(r)
	seq := 0
	for sc.Scan() {
		line := sc.Text()
		if acc != nil {
			acc.WriteString(line)
			acc.WriteByte('\n')
		}
		if onInterval != nil {
			if tps, latMs, ok := parsePgBenchProgressLine(line); ok {
				seq++
				onInterval(schema.IntervalSample{
					Seq:            seq,
					Timestamp:      time.Now().UTC(),
					IOPS:           tps,
					ThroughputMBps: 0,
					AvgLatencyUS:   latMs * 1000,
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

func parsePgBenchProgressLine(line string) (tps, latMs float64, ok bool) {
	m := rePgProgress.FindStringSubmatch(line)
	if len(m) < 4 {
		return 0, 0, false
	}
	tps, _ = strconv.ParseFloat(m[2], 64)
	latMs, _ = strconv.ParseFloat(m[3], 64)
	return tps, latMs, true
}

func sbkMockIntervals(driver string, durationSec, threads int) []schema.IntervalSample {
	if durationSec <= 0 {
		durationSec = 5
	}
	baseIOPS := 25000.0
	switch driver {
	case "kafka":
		baseIOPS = 80000
	case "rocksdb":
		baseIOPS = 150000
	case "postgresql":
		baseIOPS = 45000
	}
	iops := baseIOPS * (0.9 + float64(threads%5)*0.02)
	out := make([]schema.IntervalSample, 0, durationSec)
	for i := 1; i <= durationSec; i++ {
		out = append(out, schema.IntervalSample{
			Seq:            i,
			ElapsedSec:     1,
			IOPS:           iops,
			ThroughputMBps: iops * 0.001,
			AvgLatencyUS:   800,
		})
	}
	return out
}
