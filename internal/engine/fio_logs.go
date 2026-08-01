package engine

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// parseFioLogIntervals reads fio write_iops_log / write_bw_log output into interval samples.
func parseFioLogIntervals(workDir, prefix string) []schema.IntervalSample {
	iopsBySeq := parseFioLogFile(filepath.Join(workDir, prefix+"_iops.1.log"))
	bwBySeq := parseFioLogFile(filepath.Join(workDir, prefix+"_bw.1.log"))
	if len(iopsBySeq) == 0 && len(bwBySeq) == 0 {
		// glob fallback for multi-job or alternate naming
		matches, _ := filepath.Glob(filepath.Join(workDir, prefix+"_iops.*.log"))
		for _, p := range matches {
			for k, v := range parseFioLogFile(p) {
				iopsBySeq[k] += v
			}
		}
		matches, _ = filepath.Glob(filepath.Join(workDir, prefix+"_bw.*.log"))
		for _, p := range matches {
			for k, v := range parseFioLogFile(p) {
				bwBySeq[k] += v
			}
		}
	}
	if len(iopsBySeq) == 0 && len(bwBySeq) == 0 {
		return nil
	}

	seqs := mapKeys(iopsBySeq, bwBySeq)
	sort.Ints(seqs)
	out := make([]schema.IntervalSample, 0, len(seqs))
	for i, seq := range seqs {
		iops := iopsBySeq[seq]
		mbps := bwBySeq[seq] / (1024 * 1024)
		out = append(out, schema.IntervalSample{
			Seq:            i + 1,
			ElapsedSec:     5,
			IOPS:           iops,
			ThroughputMBps: mbps,
		})
	}
	return out
}

// parseFioLogFile parses fio log lines: time_ms, value, rw, bs.
func parseFioLogFile(path string) map[int]float64 {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	out := map[int]float64{}
	sc := bufio.NewScanner(f)
	seq := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, ", ")
		if len(parts) < 2 {
			parts = strings.Split(line, ",")
		}
		if len(parts) < 2 {
			continue
		}
		val, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			continue
		}
		seq++
		out[seq] += val
	}
	return out
}

func mapKeys(maps ...map[int]float64) []int {
	seen := map[int]struct{}{}
	for _, m := range maps {
		for k := range m {
			seen[k] = struct{}{}
		}
	}
	out := make([]int, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

// attachFioIntervals merges log-derived intervals into results when present.
func attachFioIntervals(res *schema.Results, workDir, prefix string) {
	if res == nil {
		return
	}
	iv := parseFioLogIntervals(workDir, prefix)
	if len(iv) > 0 {
		res.Intervals = iv
	}
}

// watchFioLogIntervals polls fio write_iops_log / write_bw_log files and invokes
// onInterval for each new time bucket until ctx is cancelled (then flushes once).
func watchFioLogIntervals(ctx context.Context, workDir, prefix string, onInterval func(schema.IntervalSample)) {
	if onInterval == nil {
		return
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	emitted := 0
	flush := func() {
		iv := parseFioLogIntervals(workDir, prefix)
		for i := emitted; i < len(iv); i++ {
			sample := iv[i]
			sample.Timestamp = time.Now().UTC()
			onInterval(sample)
		}
		emitted = len(iv)
	}
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


func fioLogPrefixBase(workDir string) string {
	return "stratabench"
}

func fioLogAvgMSec(runtime int) int {
	if runtime >= 300 {
		return 5000
	}
	if runtime >= 60 {
		return 2000
	}
	return 1000
}

func fioJobLogSection(workDir string, runtime int) string {
	base := fioLogPrefixBase(workDir)
	ms := fioLogAvgMSec(runtime)
	return fmt.Sprintf(`write_iops_log=%s
write_bw_log=%s
log_avg_msec=%d
`, base, base, ms)
}
