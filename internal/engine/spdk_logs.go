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

var reSPDKLiveIOPS = regexp.MustCompile(`(?i)IOPS\s*[:=]\s*([0-9.]+)`)
var reSPDKLiveMB = regexp.MustCompile(`(?i)MiB/s\s*[:=]\s*([0-9.]+)`)

func scanSPDKStream(ctx context.Context, r io.Reader, onInterval func(schema.IntervalSample), acc *bytes.Buffer) {
	if r == nil {
		return
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	seq := 0
	var pendingIOPS, pendingMB float64
	flush := func() {
		if pendingIOPS == 0 && pendingMB == 0 {
			return
		}
		seq++
		onInterval(schema.IntervalSample{
			Seq:            seq,
			Timestamp:      time.Now().UTC(),
			IOPS:           pendingIOPS,
			ThroughputMBps: pendingMB,
			ElapsedSec:     1,
		})
		pendingIOPS, pendingMB = 0, 0
	}
	for sc.Scan() {
		line := sc.Text()
		if acc != nil {
			acc.WriteString(line)
			acc.WriteByte('\n')
		}
		if onInterval != nil {
			if iops, mbps, ok := parseSPDKLiveLine(line); ok {
				if iops > 0 {
					pendingIOPS = iops
				}
				if mbps > 0 {
					pendingMB = mbps
				}
				if pendingIOPS > 0 || pendingMB > 0 {
					flush()
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func parseSPDKLiveLine(line string) (iops, mbps float64, ok bool) {
	if m := reSPDKLiveIOPS.FindStringSubmatch(line); len(m) >= 2 {
		iops, _ = strconv.ParseFloat(m[1], 64)
		ok = true
	}
	if m := reSPDKLiveMB.FindStringSubmatch(line); len(m) >= 2 {
		mbps, _ = strconv.ParseFloat(m[1], 64)
		ok = true
	}
	return iops, mbps, ok
}
