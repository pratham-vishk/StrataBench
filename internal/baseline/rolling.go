package baseline

import (
	"fmt"
	"time"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

const DefaultRollingDays = 30

type ReferenceStats struct {
	IOPS   float64
	P99    float64
	Source string
	RunID  string
}

// ReferenceFromHistory computes rolling reference from historical runs (best IOPS, best p99).
func ReferenceFromHistory(history []*schema.RunResult, profile, targetKey string, excludeRunID string) *ReferenceStats {
	var bestIOPS, bestP99 float64
	var iopsRunID string
	count := 0
	for _, h := range history {
		if h.RunID == excludeRunID {
			continue
		}
		if h.Profile != profile || TargetKey(h) != targetKey {
			continue
		}
		if h.Results.IOPS <= 0 {
			continue
		}
		count++
		if h.Results.IOPS > bestIOPS {
			bestIOPS = h.Results.IOPS
			iopsRunID = h.RunID
		}
		if h.Results.LatencyUS.P99 > 0 && (bestP99 == 0 || h.Results.LatencyUS.P99 < bestP99) {
			bestP99 = h.Results.LatencyUS.P99
		}
	}
	if count == 0 {
		return nil
	}
	return &ReferenceStats{
		IOPS:   bestIOPS,
		P99:    bestP99,
		Source: fmt.Sprintf("rolling_%dd", DefaultRollingDays),
		RunID:  iopsRunID,
	}
}

func CheckAgainstReference(current *schema.RunResult, ref *ReferenceStats, iopsThreshold, latencyThreshold float64) []Alert {
	if ref == nil {
		return nil
	}
	synthetic := &schema.RunResult{
		RunID: ref.RunID,
		Results: schema.Results{
			IOPS:      ref.IOPS,
			LatencyUS: schema.LatencyUS{P99: ref.P99},
		},
	}
	alerts := Check(current, synthetic, iopsThreshold, latencyThreshold)
	for i := range alerts {
		alerts[i].Message = "[rolling] " + alerts[i].Message
	}
	return alerts
}

func RollingSince(days int) time.Time {
	if days <= 0 {
		days = DefaultRollingDays
	}
	return time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
}
