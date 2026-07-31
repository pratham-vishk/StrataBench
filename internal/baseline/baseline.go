package baseline

import (
	"fmt"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

const (
	DefaultIOPSDegradePct   = 10.0
	DefaultLatencyDegradePct = 15.0
)

type Entry struct {
	Profile   string `json:"profile"`
	TargetKey string `json:"target_key"`
	RunID     string `json:"run_id"`
	SetAt     string `json:"set_at"`
}

type Alert struct {
	Metric       string  `json:"metric"`
	Baseline     float64 `json:"baseline"`
	Current      float64 `json:"current"`
	DeltaPct     float64 `json:"delta_pct"`
	ThresholdPct float64 `json:"threshold_pct"`
	Message      string  `json:"message"`
}

// TargetKey normalizes profile+target for baseline lookup.
func TargetKey(run *schema.RunResult) string {
	if run.Target.Endpoint != "" {
		return strings.TrimSpace(run.Target.Endpoint)
	}
	if run.Target.Device != "" {
		return strings.TrimSpace(run.Target.Device)
	}
	if run.Target.Host != "" {
		return strings.TrimSpace(run.Target.Host)
	}
	return "_default"
}

func Check(current, baseline *schema.RunResult, iopsThreshold, latencyThreshold float64) []Alert {
	if iopsThreshold <= 0 {
		iopsThreshold = DefaultIOPSDegradePct
	}
	if latencyThreshold <= 0 {
		latencyThreshold = DefaultLatencyDegradePct
	}

	var alerts []Alert
	if baseline.Results.IOPS > 0 && current.Results.IOPS > 0 {
		delta := pctDelta(baseline.Results.IOPS, current.Results.IOPS)
		if delta < -iopsThreshold {
			alerts = append(alerts, Alert{
				Metric:       "iops",
				Baseline:     baseline.Results.IOPS,
				Current:      current.Results.IOPS,
				DeltaPct:     delta,
				ThresholdPct: iopsThreshold,
				Message: fmt.Sprintf(
					"IOPS regressed %.1f%% (%.0f → %.0f, threshold %.0f%%)",
					-delta, baseline.Results.IOPS, current.Results.IOPS, iopsThreshold,
				),
			})
		}
	}
	if baseline.Results.LatencyUS.P99 > 0 && current.Results.LatencyUS.P99 > 0 {
		delta := pctDelta(baseline.Results.LatencyUS.P99, current.Results.LatencyUS.P99)
		if delta > latencyThreshold {
			alerts = append(alerts, Alert{
				Metric:       "latency_p99",
				Baseline:     baseline.Results.LatencyUS.P99,
				Current:      current.Results.LatencyUS.P99,
				DeltaPct:     delta,
				ThresholdPct: latencyThreshold,
				Message: fmt.Sprintf(
					"p99 latency regressed %.1f%% (%.0fµs → %.0fµs, threshold %.0f%%)",
					delta, baseline.Results.LatencyUS.P99, current.Results.LatencyUS.P99, latencyThreshold,
				),
			})
		}
	}
	return alerts
}

func PrintAlerts(alerts []Alert) {
	if len(alerts) == 0 {
		fmt.Println("No regression vs baseline.")
		return
	}
	fmt.Println("Regression alerts:")
	for _, a := range alerts {
		fmt.Printf("  [%s] %s\n", a.Metric, a.Message)
	}
}

func pctDelta(base, current float64) float64 {
	if base == 0 {
		return 0
	}
	return ((current - base) / base) * 100
}
