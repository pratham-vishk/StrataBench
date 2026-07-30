package engine

import (
	"context"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

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

	p50 := 120.0
	if in.Profile.Load == "heavy" {
		p50 = 80
	}
	if in.Profile.Layer == "object" {
		p50 = 4500
	}

	res := &schema.Results{
		IOPS:           iops,
		OpsPerSec:      iops,
		ThroughputMBps: throughput,
		LatencyUS: schema.LatencyUS{
			Min:  p50 * 0.4,
			Mean: p50 * 1.1,
			P50:  p50,
			P95:  p50 * 2.2,
			P99:  p50 * 3.5,
			P999: p50 * 8,
		},
		CPUPercent:      35 + rand.Float64()*20,
		TotalOperations: int64(iops * float64(durationSec)),
	}

	return res, &schema.RawEngineOutput{Path: "mock", Format: "mock"}, nil
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
