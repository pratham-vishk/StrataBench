package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

var (
	runsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stratabench_runs_total",
		Help: "Total benchmark runs completed",
	}, []string{"profile", "engine", "layer", "mock"})

	iopsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "stratabench_iops",
		Help: "Latest run IOPS",
	}, []string{"run_id", "profile"})

	throughputGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "stratabench_throughput_mbps",
		Help: "Latest run throughput MB/s",
	}, []string{"run_id", "profile"})

	latencyP99 = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "stratabench_latency_p99_us",
		Help: "Latest run p99 latency microseconds",
	}, []string{"run_id", "profile"})

	runAssignmentProgress = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "stratabench_run_assignment_progress",
		Help: "Fraction of topology assignments completed for in-flight runs (0-1)",
	}, []string{"run_id", "profile", "phase"})

	runAssignmentTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "stratabench_run_assignments_total",
		Help: "Total topology assignments for in-flight runs",
	}, []string{"run_id", "profile"})
)

func init() {
	prometheus.MustRegister(runsTotal, iopsGauge, throughputGauge, latencyP99, runAssignmentProgress, runAssignmentTotal)
}

func RecordRun(run *schema.RunResult) {
	mock := "false"
	if run.Mock {
		mock = "true"
	}
	runsTotal.WithLabelValues(run.Profile, run.Engine, run.Layer, mock).Inc()
	iopsGauge.WithLabelValues(run.RunID, run.Profile).Set(run.Results.IOPS)
	throughputGauge.WithLabelValues(run.RunID, run.Profile).Set(run.Results.ThroughputMBps)
	latencyP99.WithLabelValues(run.RunID, run.Profile).Set(run.Results.LatencyUS.P99)
}

// RecordProgress updates live Prometheus gauges for an in-flight run.
func RecordProgress(runID, profile, phase string, completed, total int) {
	if total <= 0 {
		runAssignmentProgress.DeleteLabelValues(runID, profile, phase)
		runAssignmentTotal.DeleteLabelValues(runID, profile)
		return
	}
	ratio := float64(completed) / float64(total)
	runAssignmentProgress.WithLabelValues(runID, profile, phase).Set(ratio)
	runAssignmentTotal.WithLabelValues(runID, profile).Set(float64(total))
}

// ClearProgress removes live gauges when a run finishes.
func ClearProgress(runID string) {
	runAssignmentProgress.DeletePartialMatch(prometheus.Labels{"run_id": runID})
	runAssignmentTotal.DeletePartialMatch(prometheus.Labels{"run_id": runID})
}

func Handler() http.Handler {
	return promhttp.Handler()
}
