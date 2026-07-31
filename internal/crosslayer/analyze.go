package crosslayer

import (
	"fmt"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

type Insight struct {
	Type    string  `json:"type"`
	Message string  `json:"message"`
	Metric  string  `json:"metric"`
	Ratio   float64 `json:"ratio,omitempty"`
}

func Analyze(runs []*schema.RunResult) []Insight {
	if len(runs) < 2 {
		return nil
	}
	var insights []Insight
	byLayer := map[string]*schema.RunResult{}
	for _, r := range runs {
		byLayer[r.Layer] = r
	}
	if block, ok := byLayer["block"]; ok {
		if obj, ok2 := byLayer["object"]; ok2 && block.Results.LatencyUS.P99 > 0 {
			ratio := obj.Results.LatencyUS.P99 / block.Results.LatencyUS.P99
			if ratio > 10 {
				insights = append(insights, Insight{
					Type:    "bottleneck",
					Metric:  "latency_p99",
					Ratio:   ratio,
					Message: fmt.Sprintf("Object layer p99 is %.0fx slower than block (%.0fµs vs %.0fµs) — likely gateway/protocol overhead", ratio, obj.Results.LatencyUS.P99, block.Results.LatencyUS.P99),
				})
			}
		}
		if block.Results.IOPS > 0 {
			for layer, r := range byLayer {
				if layer == "block" || r.Results.IOPS == 0 {
					continue
				}
				pct := (r.Results.IOPS / block.Results.IOPS) * 100
				if pct < 5 {
					insights = append(insights, Insight{
						Type:    "throughput_gap",
						Metric:  "iops",
						Message: fmt.Sprintf("%s layer achieves only %.1f%% of block IOPS (%.0f vs %.0f)", layer, pct, r.Results.IOPS, block.Results.IOPS),
					})
				}
			}
		}
	}
	return insights
}

func PrintInsights(insights []Insight) {
	if len(insights) == 0 {
		fmt.Println("No cross-layer insights (need 2+ layers with results).")
		return
	}
	fmt.Println("Cross-layer insights:")
	for _, ins := range insights {
		fmt.Printf("  [%s] %s\n", ins.Type, ins.Message)
	}
}

func ParseProfilesCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
