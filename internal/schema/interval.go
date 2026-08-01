package schema

import "time"

// IntervalSample is one time-bucket measurement (sbk-charts R-sheet row).
type IntervalSample struct {
	Seq                  int       `json:"seq,omitempty"`
	Timestamp            time.Time `json:"timestamp,omitempty"`
	ElapsedSec           float64   `json:"elapsed_sec,omitempty"`
	IOPS                 float64   `json:"iops,omitempty"`
	ReadIOPS             float64   `json:"read_iops,omitempty"`
	WriteIOPS            float64   `json:"write_iops,omitempty"`
	ThroughputMBps       float64   `json:"throughput_mbps,omitempty"`
	ReadMBps             float64   `json:"read_mbps,omitempty"`
	WriteMBps            float64   `json:"write_mbps,omitempty"`
	AvgLatencyUS         float64   `json:"avg_latency_us,omitempty"`
	MinLatencyUS         float64   `json:"min_latency_us,omitempty"`
	MaxLatencyUS         float64   `json:"max_latency_us,omitempty"`
	WriteTimeoutEvents   int64     `json:"write_timeout_events,omitempty"`
	ReadTimeoutEvents    int64     `json:"read_timeout_events,omitempty"`
	WriteTimeoutPerSec   float64            `json:"write_timeout_per_sec,omitempty"`
	ReadTimeoutPerSec    float64            `json:"read_timeout_per_sec,omitempty"`
	Percentiles          map[string]float64 `json:"percentiles,omitempty"`
}
