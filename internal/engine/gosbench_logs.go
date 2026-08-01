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

var reGosbenchLiveOPS = regexp.MustCompile(`(?i)(?:ops/s|operations/s|obj/s)\s*[:=]?\s*([0-9.]+)`)
var reGosbenchLiveBW = regexp.MustCompile(`(?i)(?:bandwidth|throughput)\s*[:=]?\s*([0-9.]+)\s*([KMGT]?B/s)`)

func scanGosbenchStream(ctx context.Context, r io.Reader, onInterval func(schema.IntervalSample), acc *bytes.Buffer) {
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
			if ops, mbps, ok := parseGosbenchLiveLine(line); ok {
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

func parseGosbenchLiveLine(line string) (ops, mbps float64, ok bool) {
	if m := reGosbenchLiveOPS.FindStringSubmatch(line); len(m) >= 2 {
		ops, _ = strconv.ParseFloat(m[1], 64)
		ok = true
	}
	if m := reGosbenchLiveBW.FindStringSubmatch(line); len(m) >= 3 {
		v, _ := strconv.ParseFloat(m[1], 64)
		mult := 1.0
		switch m[2] {
		case "KB/s", "KB/S":
			mult = 1.0 / 1024
		case "GB/s", "GB/S":
			mult = 1024.0
		}
		mbps = v * mult
		ok = true
	}
	return ops, mbps, ok
}

func gosbenchMockIntervals(durationSec, workers int) []schema.IntervalSample {
	if durationSec <= 0 {
		durationSec = 5
	}
	ops := float64(workers) * 250
	out := make([]schema.IntervalSample, 0, durationSec)
	for i := 1; i <= durationSec; i++ {
		out = append(out, schema.IntervalSample{
			Seq:            i,
			ElapsedSec:     1,
			IOPS:           ops,
			ThroughputMBps: ops * 0.064,
			AvgLatencyUS:   8000,
		})
	}
	return out
}
