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
	IOPS             float64   `json:"iops,omitempty"`
	ReadIOPS         float64   `json:"read_iops,omitempty"`
	WriteIOPS        float64   `json:"write_iops,omitempty"`
	ThroughputMBps   float64   `json:"throughput_mbps,omitempty"`
	OpsPerSec        float64   `json:"ops_per_sec,omitempty"`
	LatencyUS        LatencyUS `json:"latency_us"`
	CPUPercent       float64   `json:"cpu_percent,omitempty"`
	TotalBytesRead   int64     `json:"total_bytes_read,omitempty"`
	TotalBytesWritten int64    `json:"total_bytes_written,omitempty"`
	TotalOperations  int64     `json:"total_operations,omitempty"`
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
	Validation    ValidationResult    `json:"validation"`
	Target        Target            `json:"target"`
	Workload      Workload          `json:"workload"`
	Results       Results           `json:"results"`
	Hardware      HardwareSnapshot  `json:"hardware_snapshot"`
	Timestamps    Timestamps        `json:"timestamps"`
	RawOutput     *RawEngineOutput  `json:"raw_engine_output,omitempty"`
	Clients       []ClientResult    `json:"clients,omitempty"`
}

type ClientResult struct {
	Host    string  `json:"host"`
	Results Results `json:"results"`
}
