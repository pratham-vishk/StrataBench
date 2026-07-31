package analyst

import (
	"fmt"
	"math"

	"github.com/pratham-vishk/stratabench/internal/baseline"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

type Insight struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

const tailLatencyRatioThreshold = 10.0
const clientVariancePctThreshold = 30.0

func Analyze(run *schema.RunResult, regressionAlerts []baseline.Alert) []Insight {
	var insights []Insight

	if run.Mock {
		insights = append(insights, Insight{
			Type:     "mock",
			Severity: "info",
			Message:  "Mock engine — results are synthetic, not real storage I/O.",
		})
	}

	for _, a := range regressionAlerts {
		insights = append(insights, Insight{
			Type:     "regression",
			Severity: "warning",
			Message:  a.Message,
		})
	}

	if run.Results.LatencyUS.P50 > 0 && run.Results.LatencyUS.P99 > 0 {
		ratio := run.Results.LatencyUS.P99 / run.Results.LatencyUS.P50
		if ratio >= tailLatencyRatioThreshold {
			insights = append(insights, Insight{
				Type:     "anomaly",
				Severity: "warning",
				Message: fmt.Sprintf(
					"Tail latency spike: p99 is %.1fx p50 (%.0fµs vs %.0fµs) — investigate queue depth, GC, or array cache effects",
					ratio, run.Results.LatencyUS.P99, run.Results.LatencyUS.P50,
				),
			})
		}
	}

	if len(run.Clients) >= 2 {
		if msg, ok := clientVarianceInsight(run.Clients); ok {
			insights = append(insights, Insight{
				Type:     "variance",
				Severity: "warning",
				Message:  msg,
			})
		}
	}

	if !run.Validation.Passed {
		insights = append(insights, Insight{
			Type:     "validation",
			Severity: "critical",
			Message:  "Validation failed — results may not reflect honest storage performance.",
		})
	}

	if len(insights) == 0 {
		insights = append(insights, Insight{
			Type:     "ok",
			Severity: "info",
			Message:  fmt.Sprintf("No anomalies detected. %.0f IOPS, p99 %.0fµs.", run.Results.IOPS, run.Results.LatencyUS.P99),
		})
	}
	return insights
}

func clientVarianceInsight(clients []schema.ClientResult) (string, bool) {
	var iops []float64
	for _, c := range clients {
		if c.Results.IOPS > 0 {
			iops = append(iops, c.Results.IOPS)
		}
	}
	if len(iops) < 2 {
		return "", false
	}
	min, max := iops[0], iops[0]
	for _, v := range iops[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if min == 0 {
		return "", false
	}
	spread := ((max - min) / min) * 100
	if spread >= clientVariancePctThreshold {
		return fmt.Sprintf(
			"Client IOPS variance %.0f%% across %d nodes (%.0f–%.0f) — check network, load balance, or device health",
			spread, len(iops), min, max,
		), true
	}
	return "", false
}

func PrintInsights(insights []Insight) {
	fmt.Println("Analyst insights:")
	for _, ins := range insights {
		fmt.Printf("  [%s/%s] %s\n", ins.Severity, ins.Type, ins.Message)
	}
}

func HasCritical(insights []Insight) bool {
	for _, ins := range insights {
		if ins.Severity == "critical" {
			return true
		}
	}
	return false
}

func SummaryText(run *schema.RunResult, insights []Insight) string {
	critical := 0
	warnings := 0
	for _, ins := range insights {
		switch ins.Severity {
		case "critical":
			critical++
		case "warning":
			warnings++
		}
	}
	status := "healthy"
	if critical > 0 {
		status = "needs attention"
	} else if warnings > 0 {
		status = "review recommended"
	}
	return fmt.Sprintf(
		"%s profile on %s: %.0f IOPS, %.2f MB/s, p99 %.0fµs — %s (%d warnings, %d critical)",
		run.Profile, run.Target.Device, run.Results.IOPS, run.Results.ThroughputMBps,
		run.Results.LatencyUS.P99, status, warnings, critical,
	)
}

// PctSpread returns percentage spread between min and max values.
func PctSpread(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	min, max := values[0], values[0]
	for _, v := range values[1:] {
		min = math.Min(min, v)
		max = math.Max(max, v)
	}
	if min == 0 {
		return 0
	}
	return ((max - min) / min) * 100
}
