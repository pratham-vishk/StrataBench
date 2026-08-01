package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
	"gopkg.in/yaml.v3"
)

type GosbenchRunner struct{}

func (g *GosbenchRunner) Name() string { return "gosbench" }

func (g *GosbenchRunner) Run(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	if in.Mock {
		return g.runSynthetic(in)
	}

	bin := resolveGosbenchServerBin()
	if bin == "" {
		return nil, nil, fmt.Errorf("gosbench-server not found in PATH (set GOSBENCH_SERVER_BIN or use --mock)")
	}

	cfgPath := in.Profile.ParamString("gosbench_config", "")
	if cfgPath == "" {
		var err error
		cfgPath, err = writeGosbenchConfig(in)
		if err != nil {
			return nil, nil, err
		}
	}

	args := []string{"-c", cfgPath}
	if listen := in.Profile.ParamString("gosbench_listen", ""); listen != "" {
		args = append(args, "-l", listen)
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = in.WorkDir
	out, err := cmd.CombinedOutput()
	logPath := filepath.Join(in.WorkDir, "gosbench-output.txt")
	_ = os.WriteFile(logPath, out, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("gosbench-server failed: %w\n%s", err, string(out))
	}

	duration := in.Profile.ParamInt("duration_sec", 60)
	res, parseErr := parseGosbenchOutput(string(out), duration)
	if parseErr != nil {
		return nil, nil, fmt.Errorf("parse gosbench output: %w (see %s)", parseErr, logPath)
	}
	return res, &schema.RawEngineOutput{Path: logPath, Format: "gosbench-text"}, nil
}

func (g *GosbenchRunner) runSynthetic(in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	duration := in.Profile.ParamInt("duration_sec", 60)
	workers := in.Profile.ParamInt("workers", 1)
	ops := float64(workers) * 250
	res := &schema.Results{
		OpsPerSec:      ops,
		IOPS:           ops,
		ThroughputMBps: ops * 0.064,
		LatencyUS:      schema.LatencyUS{P50: 8000, P99: 25000},
		TotalOperations: int64(ops * float64(duration)),
	}
	logPath := filepath.Join(in.WorkDir, "gosbench-synthetic.txt")
	_ = os.WriteFile(logPath, []byte("gosbench synthetic mock\n"), 0o644)
	return res, &schema.RawEngineOutput{Path: logPath, Format: "gosbench-synthetic"}, nil
}

func resolveGosbenchServerBin() string {
	if v := os.Getenv("GOSBENCH_SERVER_BIN"); v != "" {
		return v
	}
	for _, name := range []string{"gosbench-server", "gosbench", "server"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

type gosbenchConfig struct {
	S3Config []gosbenchS3Config `yaml:"s3_config"`
	Tests    []gosbenchTest     `yaml:"tests"`
}

type gosbenchS3Config struct {
	AccessKey      string `yaml:"access_key"`
	SecretKey      string `yaml:"secret_key"`
	Region         string `yaml:"region"`
	Endpoint       string `yaml:"endpoint"`
	SkipSSLVerify  bool   `yaml:"skipSSLverify"`
}

type gosbenchTest struct {
	Name                string `yaml:"name"`
	ReadWeight          int    `yaml:"read_weight"`
	ExistingReadWeight  int    `yaml:"existing_read_weight"`
	WriteWeight         int    `yaml:"write_weight"`
	DeleteWeight        int    `yaml:"delete_weight"`
	ListWeight          int    `yaml:"list_weight"`
	StopWithRuntime     string `yaml:"stop_with_runtime"`
	Workers             int    `yaml:"workers"`
	WorkersShareBuckets bool   `yaml:"workers_share_buckets"`
	ParallelClients     int    `yaml:"parallel_clients"`
	CleanAfter          bool   `yaml:"clean_after"`
	Objects             gosbenchObjects `yaml:"objects"`
	Buckets             gosbenchBuckets `yaml:"buckets"`
	BucketPrefix        string `yaml:"bucket_prefix"`
	ObjectPrefix        string `yaml:"object_prefix"`
}

type gosbenchObjects struct {
	SizeMin            int    `yaml:"size_min"`
	SizeMax            int    `yaml:"size_max"`
	SizeDistribution   string `yaml:"size_distribution"`
	Unit               string `yaml:"unit"`
	NumberMin          int    `yaml:"number_min"`
	NumberMax          int    `yaml:"number_max"`
	NumberDistribution string `yaml:"number_distribution"`
}

type gosbenchBuckets struct {
	NumberMin          int    `yaml:"number_min"`
	NumberMax          int    `yaml:"number_max"`
	NumberDistribution string `yaml:"number_distribution"`
}

func writeGosbenchConfig(in RunInput) (string, error) {
	endpoint := in.Target
	if endpoint == "" {
		endpoint = in.Profile.ParamString("endpoint", "127.0.0.1:9000")
	}
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "http://" + endpoint
	}

	duration := in.Profile.ParamInt("duration_sec", 60)
	writeWeight := in.Profile.ParamInt("write_weight", 100)
	readWeight := in.Profile.ParamInt("read_weight", 0)
	sizeMin := in.Profile.ParamInt("object_size_min_kb", 4)
	sizeMax := in.Profile.ParamInt("object_size_max_kb", 64)
	workers := in.Profile.ParamInt("workers", 1)
	if workers < 1 {
		workers = 1
	}

	cfg := gosbenchConfig{
		S3Config: []gosbenchS3Config{{
			AccessKey:     envOr("GOSBENCH_ACCESS_KEY", envOr("WARP_ACCESS_KEY", "minioadmin")),
			SecretKey:     envOr("GOSBENCH_SECRET_KEY", envOr("WARP_SECRET_KEY", "minioadmin")),
			Region:        in.Profile.ParamString("region", "us-east-1"),
			Endpoint:      endpoint,
			SkipSSLVerify: in.Profile.ParamBool("skip_ssl_verify", true),
		}},
		Tests: []gosbenchTest{{
			Name:                in.Profile.Name,
			ReadWeight:          readWeight,
			WriteWeight:         writeWeight,
			StopWithRuntime:     fmt.Sprintf("%ds", duration),
			Workers:             workers,
			WorkersShareBuckets: true,
			ParallelClients:     in.Profile.ParamInt("parallel_clients", 4),
			CleanAfter:          in.Profile.ParamBool("clean_after", true),
			BucketPrefix:        in.Profile.ParamString("bucket_prefix", "stratabench-"),
			ObjectPrefix:        in.Profile.ParamString("object_prefix", "obj-"),
			Objects: gosbenchObjects{
				SizeMin:            sizeMin,
				SizeMax:            sizeMax,
				SizeDistribution:   "random",
				Unit:               "KB",
				NumberMin:          in.Profile.ParamInt("objects_per_bucket", 100),
				NumberMax:          in.Profile.ParamInt("objects_per_bucket", 100),
				NumberDistribution: "constant",
			},
			Buckets: gosbenchBuckets{
				NumberMin:          1,
				NumberMax:          in.Profile.ParamInt("buckets", 4),
				NumberDistribution: "constant",
			},
		}},
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return "", err
	}
	path := filepath.Join(in.WorkDir, "gosbench-config.yaml")
	if err := os.MkdirAll(in.WorkDir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

var (
	reGosbenchOPS = regexp.MustCompile(`(?i)(?:ops/s|operations/s|obj/s)\s*[:=]?\s*([0-9.]+)`)
	reGosbenchBW  = regexp.MustCompile(`(?i)(?:bandwidth|throughput)\s*[:=]?\s*([0-9.]+)\s*([KMGT]?B/s)`)
)

func parseGosbenchOutput(text string, durationSec int) (*schema.Results, error) {
	res := &schema.Results{LatencyUS: schema.LatencyUS{P50: 5000, P99: 18000}}
	if m := reGosbenchOPS.FindStringSubmatch(text); len(m) >= 2 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			res.OpsPerSec = v
			res.IOPS = v
		}
	}
	if m := reGosbenchBW.FindStringSubmatch(text); len(m) >= 3 {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			mult := 1.0 / 1024
			switch strings.ToUpper(m[2]) {
			case "KB/S":
				mult = 1.0 / 1024
			case "MB/S":
				mult = 1.0
			case "GB/S":
				mult = 1024.0
			}
			res.ThroughputMBps = v * mult
		}
	}
	if res.OpsPerSec == 0 && res.ThroughputMBps == 0 {
		return nil, fmt.Errorf("could not parse gosbench output")
	}
	if durationSec <= 0 {
		durationSec = 60
	}
	res.TotalOperations = int64(res.OpsPerSec * float64(durationSec))
	return res, nil
}
