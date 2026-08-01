package engine

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

var reVdbInterval = regexp.MustCompile(`(?i)interval\s+(\d+)\s*,\s*([0-9][0-9.,]*)`)
var reVdbLiveRate = regexp.MustCompile(`(?i)(?:rate|iops)[^\d]*([0-9][0-9.,]*)`)

func scanVdbenchStream(ctx context.Context, r io.Reader, onInterval func(schema.IntervalSample), acc *bytes.Buffer) {
	if r == nil {
		return
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if acc != nil {
			acc.WriteString(line)
			acc.WriteByte('\n')
		}
		if onInterval != nil {
			if sample, ok := parseVdbenchLiveLine(line); ok {
				sample.Timestamp = time.Now().UTC()
				onInterval(sample)
			}
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func parseVdbenchLiveLine(line string) (schema.IntervalSample, bool) {
	if m := reVdbInterval.FindStringSubmatch(line); len(m) >= 3 {
		seq, _ := strconv.Atoi(m[1])
		iops, _ := strconv.ParseFloat(strings.ReplaceAll(m[2], ",", ""), 64)
		if iops > 0 {
			return schema.IntervalSample{
				Seq:            seq,
				ElapsedSec:     1,
				IOPS:           iops,
				ThroughputMBps: iops * 4096 / (1024 * 1024),
			}, true
		}
	}
	if m := reVdbLiveRate.FindStringSubmatch(line); len(m) >= 2 {
		iops, _ := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64)
		if iops > 0 {
			return schema.IntervalSample{
				ElapsedSec:     1,
				IOPS:           iops,
				ThroughputMBps: iops * 4096 / (1024 * 1024),
			}, true
		}
	}
	return schema.IntervalSample{}, false
}
