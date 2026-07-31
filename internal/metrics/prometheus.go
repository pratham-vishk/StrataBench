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
)

func init() {
	prometheus.MustRegister(runsTotal, iopsGauge, throughputGauge, latencyP99)
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

func Handler() http.Handler {
	return promhttp.Handler()
}
