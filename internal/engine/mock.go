package engine

import (
	"context"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/pratham-vishk/stratabench/internal/metrics"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

type MockRunner struct{}

func (m *MockRunner) Name() string { return "mock" }

func (m *MockRunner) Run(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	_, blockSize, _, durationSec, _, qd, threads, _, _ := in.Profile.ToWorkload()
	if durationSec <= 0 {
		durationSec = 5
	}

	simDuration := time.Duration(durationSec) * time.Second
	if simDuration > 3*time.Second {
		simDuration = 3 * time.Second
	}
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-time.After(simDuration):
	}

	bsBytes := float64(parseBlockSizeBytes(blockSize))
	baseIOPS := 50000.0 * float64(threads) * math.Sqrt(float64(qd))
	if in.Profile.Layer == "object" {
		baseIOPS = 5000 * float64(threads)
	}
	jitter := 0.85 + rand.Float64()*0.3
	iops := baseIOPS * jitter
	throughput := iops * bsBytes / (1024 * 1024)
	readIOPS := iops * 0.7
	writeIOPS := iops * 0.3
	if in.Profile.Layer == "object" {
		op := strings.ToLower(in.Profile.ParamString("operation", "put"))
		switch op {
		case "get", "read":
			readIOPS, writeIOPS = iops, 0
		case "put", "write":
			readIOPS, writeIOPS = 0, iops
		case "mixed":
			readIOPS, writeIOPS = iops*0.45, iops*0.55
		default:
			writeIOPS, readIOPS = iops, 0
		}
	}

	p50 := 120.0
	if in.Profile.Load == "heavy" {
		p50 = 80
	}
	if in.Profile.Layer == "object" {
		p50 = 4500
	}

	percentiles := mockPercentiles(p50)
	counts := mockPercentileCounts(int64(iops * float64(durationSec)))
	intervals := mockIntervals(durationSec, baseIOPS, throughput, p50)

	res := &schema.Results{
		IOPS:             iops,
		ReadIOPS:         readIOPS,
		WriteIOPS:        writeIOPS,
		OpsPerSec:        iops,
		ThroughputMBps:   throughput,
		Percentiles:      percentiles,
		PercentileCounts: counts,
		Intervals:        intervals,
		LatencyUS: schema.LatencyUS{
			Min:  p50 * 0.4,
			Mean: p50 * 1.1,
			P50:  p50,
			P75:  p50 * 1.4,
			P90:  p50 * 1.8,
			P95:  p50 * 2.2,
			P99:  p50 * 3.5,
			P999: p50 * 8,
			P9999: p50 * 12,
			Max:  p50 * 15,
		},
		CPUPercent:      35 + rand.Float64()*20,
		TotalOperations: int64(iops * float64(durationSec)),
		Totals: schema.TotalStats{
			TotalMB:             throughput * float64(durationSec),
			TotalRecords:        int64(iops * float64(durationSec)),
			WriteRequestMB:      throughput * 0.3 * float64(durationSec),
			WriteRequestRecords: int64(writeIOPS * float64(durationSec)),
			ReadRequestMB:       throughput * 0.7 * float64(durationSec),
			ReadRequestRecords:  int64(readIOPS * float64(durationSec)),
		},
	}
	metrics.PopulateLatencyUS(&res.LatencyUS, percentiles)

	return res, &schema.RawEngineOutput{Path: "mock", Format: "mock"}, nil
}

func mockPercentiles(p50 float64) map[string]float64 {
	m := map[string]float64{}
	for _, l := range metrics.StandardPercentileLabels {
		switch {
		case l == "p50":
			m[l] = p50
		case strings.HasPrefix(l, "p99"):
			m[l] = p50 * (3 + rand.Float64()*2)
		case strings.HasPrefix(l, "p9"):
			m[l] = p50 * (2 + rand.Float64())
		default:
			m[l] = p50 * (0.8 + rand.Float64()*0.5)
		}
	}
	return m
}

func mockPercentileCounts(totalOps int64) map[string]int64 {
	m := map[string]int64{}
	for _, l := range metrics.StandardPercentileLabels {
		m[l] = totalOps / 100
	}
	return m
}

func mockIntervals(durationSec int, baseIOPS, throughput, p50 float64) []schema.IntervalSample {
	buckets := durationSec / 5
	if buckets < 3 {
		buckets = 3
	}
	if buckets > 12 {
		buckets = 12
	}
	start := time.Now().UTC().Add(-time.Duration(durationSec) * time.Second)
	var out []schema.IntervalSample
	for i := 0; i < buckets; i++ {
		j := 0.9 + rand.Float64()*0.2
		out = append(out, schema.IntervalSample{
			Seq:            i + 1,
			Timestamp:      start.Add(time.Duration(i*5) * time.Second),
			ElapsedSec:     5,
			IOPS:           baseIOPS * j,
			ReadIOPS:       baseIOPS * j * 0.7,
			WriteIOPS:      baseIOPS * j * 0.3,
			ThroughputMBps: throughput * j,
			ReadMBps:       throughput * j * 0.7,
			WriteMBps:      throughput * j * 0.3,
			AvgLatencyUS:   p50 * (0.9 + rand.Float64()*0.3),
			MinLatencyUS:   p50 * 0.4,
			MaxLatencyUS:   p50 * (8 + rand.Float64()*4),
		})
	}
	return out
}

func parseBlockSizeBytes(bs string) int64 {
	bs = strings.TrimSpace(strings.ToLower(bs))
	mult := int64(1)
	switch {
	case strings.HasSuffix(bs, "k"):
		mult = 1024
		bs = strings.TrimSuffix(bs, "k")
	case strings.HasSuffix(bs, "m"):
		mult = 1024 * 1024
		bs = strings.TrimSuffix(bs, "m")
	}
	n, err := strconv.ParseInt(bs, 10, 64)
	if err != nil || n <= 0 {
		return 4096
	}
	return n * mult
}
