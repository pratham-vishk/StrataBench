// Reference implementation of the native StrataBench engine contract.
// Production builds may replace this with a Rust binary using the same CLI.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
)

type engineConfig struct {
	Target       string         `json:"target"`
	Profile      string         `json:"profile"`
	Layer        string         `json:"layer"`
	Pattern      string         `json:"pattern"`
	BlockSize    string         `json:"block_size"`
	DatasetSize  string         `json:"dataset_size"`
	DurationSec  int            `json:"duration_sec"`
	RampSec      int            `json:"ramp_sec"`
	QueueDepth   int            `json:"queue_depth"`
	Threads      int            `json:"threads"`
	ReadWriteMix int            `json:"read_write_mix"`
	DirectIO     bool           `json:"direct_io"`
	Params       map[string]any `json:"params,omitempty"`
}

type engineResults struct {
	IOPS           float64            `json:"iops"`
	ReadIOPS       float64            `json:"read_iops,omitempty"`
	WriteIOPS      float64            `json:"write_iops,omitempty"`
	ThroughputMBps float64            `json:"throughput_mbps"`
	OpsPerSec      float64            `json:"ops_per_sec"`
	LatencyUS      map[string]float64 `json:"latency_us"`
	TotalOperations int64             `json:"total_operations,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: stratabench-engine run --config <file> --output <file>")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "run":
		if err := runBenchmark(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "version":
		fmt.Println("stratabench-engine 0.1.0-stub (go reference)")
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
}

func runBenchmark(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "input config JSON")
	outPath := fs.String("output", "", "output results JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *cfgPath == "" || *outPath == "" {
		return fmt.Errorf("--config and --output are required")
	}
	raw, err := os.ReadFile(*cfgPath)
	if err != nil {
		return err
	}
	var cfg engineConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}
	res := synthesize(cfg)
	out, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(*outPath, out, 0o644)
}

func synthesize(cfg engineConfig) engineResults {
	threads := cfg.Threads
	if threads <= 0 {
		threads = 1
	}
	qd := cfg.QueueDepth
	if qd <= 0 {
		qd = 32
	}
	duration := cfg.DurationSec
	if duration <= 0 {
		duration = 60
	}

	bs := parseBlockBytes(cfg.BlockSize)
	base := 40000.0 * float64(threads) * math.Sqrt(float64(qd))
	if cfg.Layer == "object" {
		base = 4000 * float64(threads)
	}
	if strings.Contains(strings.ToLower(cfg.Pattern), "seq") {
		base *= 1.8
	}
	iops := base
	read := iops * 0.7
	write := iops * 0.3
	if cfg.ReadWriteMix > 0 {
		read = iops * float64(cfg.ReadWriteMix) / 100
		write = iops - read
	}
	mbps := iops * bs / (1024 * 1024)
	p50 := 80.0 + float64(qd)*2
	p99 := p50 * 4

	return engineResults{
		IOPS:            iops,
		ReadIOPS:        read,
		WriteIOPS:       write,
		ThroughputMBps:  mbps,
		OpsPerSec:       iops,
		TotalOperations: int64(iops * float64(duration)),
		LatencyUS: map[string]float64{
			"p50": p50,
			"p95": p50 * 2.5,
			"p99": p99,
		},
	}
}

func parseBlockBytes(s string) float64 {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 4096
	}
	mult := 1.0
	switch {
	case strings.HasSuffix(s, "kib"), strings.HasSuffix(s, "kb"):
		mult = 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "kib"), "kb")
	case strings.HasSuffix(s, "mib"), strings.HasSuffix(s, "mb"):
		mult = 1024 * 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "mib"), "mb")
	}
	var n float64
	fmt.Sscanf(s, "%f", &n)
	if n <= 0 {
		return 4096
	}
	return n * mult
}
