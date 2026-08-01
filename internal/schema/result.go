package schema

import "time"

const SchemaVersion = "1.0.0"

type ValidationResult struct {
	Passed       bool     `json:"passed"`
	RulesChecked []string `json:"rules_checked"`
	Warnings     []Warning `json:"warnings"`
	Errors       []string `json:"errors,omitempty"`
}

type Warning struct {
	Rule     string `json:"rule"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type Target struct {
	Type     string            `json:"type"`
	Device   string            `json:"device,omitempty"`
	Host     string            `json:"host,omitempty"`
	Endpoint string            `json:"endpoint,omitempty"`
	VM       *string           `json:"vm"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Workload struct {
	Pattern       string `json:"pattern,omitempty"`
	BlockSize     string `json:"block_size,omitempty"`
	ReadWriteMix  int    `json:"read_write_mix,omitempty"`
	QueueDepth    int    `json:"queue_depth,omitempty"`
	Threads       int    `json:"threads,omitempty"`
	DatasetSize   string `json:"dataset_size,omitempty"`
	DurationSec   int    `json:"duration_sec,omitempty"`
	RampTimeSec   int    `json:"ramp_time_sec,omitempty"`
	DirectIO      bool   `json:"direct_io,omitempty"`
}

type LatencyUS struct {
	Min   float64 `json:"min,omitempty"`
	Max   float64 `json:"max,omitempty"`
	Mean  float64 `json:"mean,omitempty"`
	P50   float64 `json:"p50,omitempty"`
	P75   float64 `json:"p75,omitempty"`
	P90   float64 `json:"p90,omitempty"`
	P95   float64 `json:"p95,omitempty"`
	P99   float64 `json:"p99,omitempty"`
	P999  float64 `json:"p99_9,omitempty"`
	P9999 float64 `json:"p99_99,omitempty"`
}

type Results struct {
	IOPS              float64            `json:"iops,omitempty"`
	ReadIOPS          float64            `json:"read_iops,omitempty"`
	WriteIOPS         float64            `json:"write_iops,omitempty"`
	ThroughputMBps    float64            `json:"throughput_mbps,omitempty"`
	OpsPerSec         float64            `json:"ops_per_sec,omitempty"`
	LatencyUS         LatencyUS          `json:"latency_us"`
	Percentiles       map[string]float64 `json:"percentiles,omitempty"`
	PercentileCounts  map[string]int64   `json:"percentile_counts,omitempty"`
	Intervals         []IntervalSample   `json:"intervals,omitempty"`
	CPUPercent        float64            `json:"cpu_percent,omitempty"`
	TotalBytesRead    int64              `json:"total_bytes_read,omitempty"`
	TotalBytesWritten int64              `json:"total_bytes_written,omitempty"`
	TotalOperations   int64              `json:"total_operations,omitempty"`
	Totals            TotalStats         `json:"totals,omitempty"`
}

// TotalStats holds SBK Total-row volume, pending, and reliability counters.
type TotalStats struct {
	TotalMB             float64 `json:"total_mb,omitempty"`
	TotalRecords        int64   `json:"total_records,omitempty"`
	WriteRequestMB      float64 `json:"write_request_mb,omitempty"`
	WriteRequestRecords int64   `json:"write_request_records,omitempty"`
	ReadRequestMB       float64 `json:"read_request_mb,omitempty"`
	ReadRequestRecords  int64   `json:"read_request_records,omitempty"`
	WritePendingMB      float64 `json:"write_pending_mb,omitempty"`
	WritePendingRecords int64   `json:"write_pending_records,omitempty"`
	ReadPendingMB       float64 `json:"read_pending_mb,omitempty"`
	ReadPendingRecords  int64   `json:"read_pending_records,omitempty"`
	WriteTimeoutEvents  int64   `json:"write_timeout_events,omitempty"`
	ReadTimeoutEvents   int64   `json:"read_timeout_events,omitempty"`
	InvalidLatencies    int64   `json:"invalid_latencies,omitempty"`
	LowerDiscard        int64   `json:"lower_discard,omitempty"`
	HigherDiscard       int64   `json:"higher_discard,omitempty"`
	SLC1                int64   `json:"slc1,omitempty"`
	SLC2                int64   `json:"slc2,omitempty"`
}

type HardwareSnapshot struct {
	Hostname     string         `json:"hostname,omitempty"`
	CPUModel     string         `json:"cpu_model,omitempty"`
	CPUCores     int            `json:"cpu_cores,omitempty"`
	MemoryBytes  int64          `json:"memory_bytes,omitempty"`
	CacheBytes   int64          `json:"cache_bytes,omitempty"`
	NICSpeedGbps int            `json:"nic_speed_gbps,omitempty"`
	RDMACapable  bool           `json:"rdma_capable,omitempty"`
	OS           string         `json:"os,omitempty"`
	Arch         string         `json:"arch,omitempty"`
	BlockDevices []BlockDevice  `json:"block_devices,omitempty"`
	NVMe         []NVMEDevice   `json:"nvme_devices,omitempty"`
}

type BlockDevice struct {
	Name        string `json:"name"`
	Model       string `json:"model,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	Rotational  bool   `json:"rotational,omitempty"`
}

type NVMEDevice struct {
	Device   string `json:"device"`
	Model    string `json:"model,omitempty"`
	Firmware string `json:"firmware,omitempty"`
	Serial   string `json:"serial,omitempty"`
}

type SMARTReading struct {
	Device         string `json:"device"`
	Model          string `json:"model,omitempty"`
	Serial         string `json:"serial,omitempty"`
	TemperatureC   int    `json:"temperature_c,omitempty"`
	PowerOnHours   int    `json:"power_on_hours,omitempty"`
	WearPercent    int    `json:"wear_percent,omitempty"`
	Reallocated    int    `json:"reallocated_sectors,omitempty"`
	HealthPassed   bool   `json:"health_passed"`
}

type Timestamps struct {
	StartedAt            time.Time  `json:"started_at"`
	CompletedAt          time.Time  `json:"completed_at"`
	SteadyStateReachedAt *time.Time `json:"steady_state_reached_at,omitempty"`
}

type RawEngineOutput struct {
	Path   string `json:"path,omitempty"`
	Format string `json:"format,omitempty"`
}

type RunResult struct {
	SchemaVersion string            `json:"schema_version"`
	RunID         string            `json:"run_id"`
	Profile       string            `json:"profile"`
	Layer         string            `json:"layer"`
	Engine        string            `json:"engine"`
	Status        string            `json:"status"`
	Mock          bool              `json:"mock,omitempty"`
	Provenance    Provenance        `json:"provenance,omitempty"`
	Validation    ValidationResult    `json:"validation"`
	Target        Target            `json:"target"`
	Workload      Workload          `json:"workload"`
	Results       Results           `json:"results"`
	Hardware      HardwareSnapshot  `json:"hardware_snapshot"`
	Timestamps    Timestamps        `json:"timestamps"`
	RawOutput     *RawEngineOutput  `json:"raw_engine_output,omitempty"`
	Clients       []ClientResult    `json:"clients,omitempty"`
	Targets       []TargetResult    `json:"targets,omitempty"`
	Topology      string            `json:"topology,omitempty"`
}

type ClientResult struct {
	Host    string  `json:"host"`
	Target  string  `json:"target,omitempty"`
	Results Results `json:"results"`
}

type TargetResult struct {
	Target  string  `json:"target"`
	Results Results `json:"results"`
}

// Provenance records git/build context for reproducible branch comparisons.
type Provenance struct {
	GitRepo     string `json:"git_repo,omitempty"`
	GitBranch   string `json:"git_branch,omitempty"`
	GitSHA      string `json:"git_sha,omitempty"`
	GitDirty    bool   `json:"git_dirty,omitempty"`
	BuildCmd    string `json:"build_cmd,omitempty"`
	ToolVersion string `json:"tool_version,omitempty"`
	CompareRole string `json:"compare_role,omitempty"` // base | head
}
