package planner

import (
	"fmt"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/profile"
)

// GuideItem is a question, warning, or recommendation for the user.
type GuideItem struct {
	Kind    string   `json:"kind"` // question, warning, recommendation, info
	Topic   string   `json:"topic"`
	Message string   `json:"message"`
	Options []string `json:"options,omitempty"`
}

// Guidance helps users refine intent before running a benchmark.
type Guidance struct {
	Ready           bool           `json:"ready"`
	Profile         string         `json:"profile"`
	Engine          string         `json:"engine"`
	Layer           string         `json:"layer"`
	Summary         string           `json:"summary"`
	Questions       []GuideItem    `json:"questions"`
	Warnings        []GuideItem    `json:"warnings"`
	Recommendations []GuideItem    `json:"recommendations"`
	AppliedDefaults map[string]any `json:"applied_defaults,omitempty"`
}

// EngineParam describes a tunable parameter for an engine.
type EngineParam struct {
	Key         string
	Aliases     []string
	Description string
	Default     string
	Required    bool
}

// EngineCatalog returns parameter definitions per engine.
func EngineCatalog(engine string) []EngineParam {
	switch engine {
	case "fio":
		return []EngineParam{
			{Key: "bs", Aliases: []string{"block_size"}, Description: "Block size (4k, 16k, 1m)", Default: "4k-16k"},
			{Key: "iodepth", Aliases: []string{"queue_depth", "qd"}, Description: "Queue depth per job", Default: "32"},
			{Key: "numjobs", Aliases: []string{"threads"}, Description: "Parallel jobs", Default: "4"},
			{Key: "runtime", Aliases: []string{"duration_sec"}, Description: "Test duration (seconds)", Default: "600"},
			{Key: "ramp_time", Description: "Warm-up before measurement", Default: "60"},
			{Key: "size", Aliases: []string{"dataset_size"}, Description: "Dataset size (must exceed cache)", Default: "200g"},
			{Key: "rw", Description: "Pattern: read, write, randread, randwrite, randrw", Default: "profile-specific"},
			{Key: "rwmixread", Description: "Read percentage for mixed workloads", Default: "70"},
		}
	case "vdbench":
		return []EngineParam{
			{Key: "block_size", Aliases: []string{"bs"}, Description: "Transfer size", Default: "4k"},
			{Key: "threads", Description: "Worker threads", Default: "16"},
			{Key: "duration_sec", Description: "Elapsed time (seconds)", Default: "300"},
			{Key: "warmup_sec", Aliases: []string{"ramp_time"}, Description: "Warm-up interval", Default: "60"},
			{Key: "pattern", Description: "randread, randwrite, seqread", Default: "randread"},
			{Key: "luns", Description: "Comma-separated LUN devices (2+ for AFA)", Required: true},
		}
	case "spdk":
		return []EngineParam{
			{Key: "transport", Description: "PCIe BDF e.g. trtype:PCIe traddr:0000:01:00.0", Required: true},
			{Key: "queue_depth", Aliases: []string{"iodepth"}, Description: "Submission queue depth", Default: "128"},
			{Key: "block_size_bytes", Description: "I/O size in bytes", Default: "4096"},
			{Key: "duration_sec", Description: "Run time (seconds)", Default: "300"},
			{Key: "pattern", Description: "randread, randwrite, read, write", Default: "randread"},
		}
	case "elbencho":
		return []EngineParam{
			{Key: "block_size", Description: "File chunk size", Default: "1m"},
			{Key: "threads", Description: "Parallel threads", Default: "16"},
			{Key: "duration_sec", Description: "Time limit (seconds)", Default: "300"},
			{Key: "pattern", Description: "read, write, randread", Default: "read"},
		}
	case "warp":
		return []EngineParam{
			{Key: "operation", Description: "put, get, mixed, delete", Default: "put"},
			{Key: "object_size", Description: "Object size or range (3KiB-100KiB)", Default: "4MiB"},
			{Key: "concurrent", Aliases: []string{"threads"}, Description: "Concurrent workers", Default: "32"},
			{Key: "duration_sec", Description: "Benchmark duration", Default: "300"},
			{Key: "bucket", Description: "S3 bucket name", Default: "stratabench-test"},
			{Key: "rdma", Description: "RDMA mode (cpu) for cluster RDMA profiles", Default: "off"},
		}
	case "sbk":
		return []EngineParam{
			{Key: "driver", Description: "postgresql, kafka, rocksdb", Required: true},
			{Key: "duration_sec", Description: "Test duration", Default: "600"},
			{Key: "connections", Aliases: []string{"threads"}, Description: "Client connections", Default: "32"},
			{Key: "dsn", Description: "PostgreSQL DSN (app-postgres)", Default: ""},
			{Key: "brokers", Description: "Kafka bootstrap servers", Default: ""},
			{Key: "topic", Description: "Kafka topic", Default: "stratabench-test"},
			{Key: "record_size_bytes", Description: "Kafka message size", Default: "4096"},
			{Key: "warehouses", Description: "TPC-C warehouses (postgres)", Default: "10"},
		}
	default:
		return nil
	}
}

// Guide analyzes plan + intent and returns discussion items before proceeding.
func Guide(plan PlanResult, intent string, p *profile.Profile) Guidance {
	g := Guidance{
		Profile: plan.Profile,
		Engine:  p.Engine,
		Layer:   p.Layer,
	}
	lower := strings.ToLower(intent)
	hasTarget := plan.Target != "" || len(plan.Targets) > 0
	hasClients := len(plan.Clients) > 0
	_, hasObjSize := plan.Params["object_size"]
	_, hasObjRange := plan.Params["object_size_min"]
	mentionsNVMe := strings.Contains(lower, "nvme") || strings.Contains(lower, "ssd")
	mentionsObject := strings.Contains(lower, "object") || strings.Contains(lower, "s3") || strings.Contains(lower, "minio")
	mentionsHDD := strings.Contains(lower, "hdd") || strings.Contains(lower, "rotational")
	mentionsRDMA := strings.Contains(lower, "rdma")
	mentionsVM := reWordVM.MatchString(lower) || strings.Contains(lower, "guest") || strings.Contains(lower, "virtual")
	mentionsApp := strings.Contains(lower, "postgres") || strings.Contains(lower, "kafka") || strings.Contains(lower, "rocksdb") || strings.Contains(lower, "database")
	deployCtx := deployContextFromPlan(plan, lower)
	mentionsPhysical := deployCtx == DeployPhysical
	mentionsVirtual := deployCtx == DeployVirtual || mentionsVM
	_, colocated := plan.Params["colocated"]

	// Physical vs virtual deployment (recommendation — not blocking unless conflicting)
	if !mentionsVirtual && !mentionsPhysical && !strings.HasPrefix(p.Layer, "application") {
		if p.Layer == "block" || p.Layer == "object" || p.Layer == "file" {
			g.Recommendations = append(g.Recommendations, GuideItem{
				Kind:    "info",
				Topic:   "deploy",
				Message: "Clarify physical host vs VM guest if it matters — say physical/bare metal or virtual/vm/guest in intent. Default profiles assume physical unless vm-* is selected.",
				Options: []string{"physical", "virtual"},
			})
		}
	}
	if mentionsPhysical && strings.HasPrefix(p.Layer, "vm-") {
		g.Recommendations = append(g.Recommendations, GuideItem{
			Kind:    "recommendation",
			Topic:   "deploy",
			Message: "Physical deployment requested — use block/object profiles on host paths (e.g. /dev/nvme0n1, host:9000), not vm-* SSH profiles.",
		})
	}
	if mentionsVirtual && !strings.HasPrefix(p.Layer, "vm-") && !strings.HasPrefix(p.Layer, "application") {
		g.Recommendations = append(g.Recommendations, GuideItem{
			Kind:    "recommendation",
			Topic:   "deploy",
			Message: "Virtual/guest workload — use vm-* profiles. Target is SSH form root@host:/dev/vdb for block, or guest IP:9000 for S3.",
			Options: []string{"vm-disk-random", "vm-nvme-oltp", "vm-s3-rdma", "vm-afa-multi-lun"},
		})
	}

	// Colocated client + server on same node
	if colocated && hasClients && len(plan.Targets) > 0 {
		g.Warnings = append(g.Warnings, GuideItem{
			Kind:    "info",
			Topic:   "topology",
			Message: "Client and server share the same host — using topology single (one node runs both warp/agent and endpoint).",
		})
		if plan.Topology == "" || plan.Topology == "shard" {
			if g.AppliedDefaults == nil {
				g.AppliedDefaults = map[string]any{}
			}
			g.AppliedDefaults["topology"] = "single"
		}
	}
	if mentionsNVMe && (hasObjSize || hasObjRange) && p.Layer != "object" && p.Layer != "vm-object" {
		g.Questions = append(g.Questions, GuideItem{
			Kind:    "question",
			Topic:   "layer",
			Message: "You mentioned NVMe and object size. Block NVMe uses block size (bs); object storage uses object_size. Which layer do you want?",
			Options: []string{"block NVMe (fio) — use bs/iodepth", "object/S3 (warp) — use object_size", "both — run cross-layer profiles"},
		})
	}
	if mentionsObject && p.Engine == "fio" {
		g.Recommendations = append(g.Recommendations, GuideItem{
			Kind:    "recommendation",
			Topic:   "profile",
			Message: "Object/S3 workload detected — consider s3-put-throughput, s3-cluster-put-get, or s3-cluster-rdma instead of block fio.",
			Options: []string{"s3-put-throughput", "s3-cluster-put-get", "s3-cluster-rdma"},
		})
	}
	if mentionsHDD && strings.Contains(plan.Profile, "nvme") {
		g.Warnings = append(g.Warnings, GuideItem{
			Kind:    "warning",
			Topic:   "profile",
			Message: "HDD mentioned but profile looks NVMe-oriented. Consider hdd-sequential-read or vm-hdd-sequential.",
		})
	}

	// Target requirements per engine
	switch p.Engine {
	case "fio":
		if !hasTarget {
			g.Questions = append(g.Questions, GuideItem{
				Kind:    "question",
				Topic:   "target",
				Message: "fio needs a block device path (e.g. /dev/nvme0n1).",
			})
		}
	case "spdk":
		if !hasTarget && plan.Params["transport"] == nil {
			g.Questions = append(g.Questions, GuideItem{
				Kind:    "question",
				Topic:   "target",
				Message: "SPDK needs PCIe BDF as target (0000:01:00.0) or transport param.",
			})
		}
	case "vdbench":
		if !hasTarget {
			g.Questions = append(g.Questions, GuideItem{
				Kind:    "question",
				Topic:   "target",
				Message: "vdbench needs block device(s) as target (comma-separated for multi-LUN).",
			})
		}
		luns := countDevices(plan.Target, plan.Targets)
		if luns < 2 && strings.Contains(plan.Profile, "afa") {
			g.Questions = append(g.Questions, GuideItem{
				Kind:    "question",
				Topic:   "luns",
				Message: "AFA multi-LUN needs 2+ devices. Provide comma-separated targets (e.g. /dev/sdb,/dev/sdc,/dev/sdd).",
			})
		}
	case "warp":
		if !hasTarget && len(plan.Targets) == 0 {
			g.Questions = append(g.Questions, GuideItem{
				Kind:    "question",
				Topic:   "target",
				Message: "S3/warp needs an endpoint (host:9000). Set target or servers in intent.",
			})
		}
		if mentionsRDMA && !strings.Contains(plan.Profile, "rdma") {
			g.Recommendations = append(g.Recommendations, GuideItem{
				Kind:    "recommendation",
				Topic:   "profile",
				Message: "RDMA mentioned — use s3-cluster-rdma or vm-s3-rdma profile.",
			})
		}
		if p.Engine == "warp" {
			g.Warnings = append(g.Warnings, GuideItem{
				Kind:    "info",
				Topic:   "credentials",
				Message: "Set WARP_ACCESS_KEY and WARP_SECRET_KEY for real S3 runs (default minioadmin for lab).",
			})
		}
	case "elbencho":
		if !hasTarget {
			g.Questions = append(g.Questions, GuideItem{
				Kind:    "question",
				Topic:   "target",
				Message: "File workload needs a mount path (e.g. /mnt/nfs/share) or SSH guest path root@host:/mnt/data.",
			})
		}
	case "sbk":
		driver := p.ParamString("driver", "")
		if driver == "postgresql" && plan.Target == "" && plan.Params["dsn"] == nil {
			g.Questions = append(g.Questions, GuideItem{
				Kind:    "question",
				Topic:   "dsn",
				Message: "PostgreSQL benchmark needs a DSN target (postgres://user@host/db) or set dsn param.",
			})
		}
		if driver == "kafka" && plan.Target == "" && plan.Params["brokers"] == nil {
			g.Questions = append(g.Questions, GuideItem{
				Kind:    "question",
				Topic:   "brokers",
				Message: "Kafka benchmark needs broker address (host:9092) as target or brokers param.",
			})
		}
	}

	// VM layer SSH format
	if strings.HasPrefix(p.Layer, "vm-") && !mentionsVM && plan.Target != "" && !strings.Contains(plan.Target, "@") {
		g.Warnings = append(g.Warnings, GuideItem{
			Kind:    "warning",
			Topic:   "vm",
			Message: "VM profile selected — target should be SSH form root@host:/dev/vdb unless using agents on guest.",
		})
	}
	if mentionsVM && !strings.HasPrefix(p.Layer, "vm-") && !strings.HasPrefix(p.Layer, "application") && !mentionsPhysical {
		g.Recommendations = append(g.Recommendations, GuideItem{
			Kind:    "recommendation",
			Topic:   "profile",
			Message: "Virtual/guest workload — consider vm-disk-random, vm-nvme-oltp, vm-s3-*, or vm-afa-multi-lun.",
		})
	}

	// Distributed topology
	if hasClients && len(plan.Targets) == 0 && plan.Target == "" {
		g.Questions = append(g.Questions, GuideItem{
			Kind:    "question",
			Topic:   "topology",
			Message: "Clients specified but no server targets. Which storage endpoints should clients hit?",
		})
	}
	if hasClients && len(plan.Targets) > 0 && plan.Topology == "" {
		g.AppliedDefaults = map[string]any{"topology": "shard"}
		g.Recommendations = append(g.Recommendations, GuideItem{
			Kind:    "recommendation",
			Topic:   "topology",
			Message: "Multiple clients and servers — defaulting topology to shard (N clients → M servers). Use pool/sweep/matrix if different.",
			Options: []string{"pool", "sweep", "shard", "matrix"},
		})
	}

	// Duration vs profile minimum
	if dur, ok := plan.Params["duration_sec"]; ok {
		if sec, ok := dur.(int); ok && sec < p.Validation.MinRuntimeSec && p.Validation.MinRuntimeSec > 0 {
			g.Warnings = append(g.Warnings, GuideItem{
				Kind:    "warning",
				Topic:   "duration",
				Message: fmt.Sprintf("Requested %ds is below profile minimum %ds for steady-state — validator may fail.", sec, p.Validation.MinRuntimeSec),
			})
		}
	}

	// Engine-specific defaults applied when missing
	g.AppliedDefaults = mergeDefaults(g.AppliedDefaults, suggestEngineDefaults(p, plan))

	// App layer detection
	if mentionsApp && p.Engine != "sbk" {
		g.Recommendations = append(g.Recommendations, GuideItem{
			Kind:    "recommendation",
			Topic:   "profile",
			Message: "Application workload — consider app-postgres-tpc-c, app-kafka-producer, or app-rocksdb-read.",
		})
	}

	g.Ready = len(g.Questions) == 0
	g.Summary = buildSummary(g, plan)
	return g
}

func deployContextFromPlan(plan PlanResult, lower string) string {
	if v, ok := plan.Params["deploy_context"].(string); ok && v != "" {
		return v
	}
	if strings.Contains(lower, "physical") || strings.Contains(lower, "bare metal") || strings.Contains(lower, "baremetal") {
		return DeployPhysical
	}
	if strings.Contains(lower, "virtual") || reWordVM.MatchString(lower) || strings.Contains(lower, "guest") {
		return DeployVirtual
	}
	return ""
}

func suggestEngineDefaults(p *profile.Profile, plan PlanResult) map[string]any {
	def := map[string]any{}
	for _, param := range EngineCatalog(p.Engine) {
		if param.Required {
			continue
		}
		if hasParam(plan.Params, param.Key) || hasAnyAlias(plan.Params, param.Aliases) {
			continue
		}
		// copy profile default if present
		if v, ok := p.Params[param.Key]; ok {
			def[param.Key] = v
		}
	}
	return def
}

func hasParam(params map[string]any, key string) bool {
	_, ok := params[key]
	return ok
}

func hasAnyAlias(params map[string]any, aliases []string) bool {
	for _, a := range aliases {
		if _, ok := params[a]; ok {
			return true
		}
	}
	return false
}

func countDevices(target string, targets []string) int {
	if len(targets) > 0 {
		return len(targets)
	}
	if target == "" {
		return 0
	}
	return len(strings.Split(target, ","))
}

func mergeDefaults(a, b map[string]any) map[string]any {
	if a == nil {
		a = map[string]any{}
	}
	for k, v := range b {
		if _, ok := a[k]; !ok {
			a[k] = v
		}
	}
	return a
}

func buildSummary(g Guidance, plan PlanResult) string {
	if g.Ready {
		return fmt.Sprintf("Ready to run %s (%s/%s). Review warnings, then proceed.", plan.Profile, g.Engine, g.Layer)
	}
	return fmt.Sprintf("Need clarification before running %s (%s/%s). Answer %d question(s) or pass --yes to use recommendations.", plan.Profile, g.Engine, g.Layer, len(g.Questions))
}

// FormatGuidance returns human-readable discussion text.
func FormatGuidance(g Guidance) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Guidance: %s\n", g.Summary)
	for _, q := range g.Questions {
		fmt.Fprintf(&b, "\n? [%s] %s\n", q.Topic, q.Message)
		for _, o := range q.Options {
			fmt.Fprintf(&b, "    - %s\n", o)
		}
	}
	for _, w := range g.Warnings {
		fmt.Fprintf(&b, "\n! [%s] %s\n", w.Topic, w.Message)
	}
	for _, r := range g.Recommendations {
		fmt.Fprintf(&b, "\n→ [%s] %s\n", r.Topic, r.Message)
		for _, o := range r.Options {
			fmt.Fprintf(&b, "    - %s\n", o)
		}
	}
	if len(g.AppliedDefaults) > 0 {
		fmt.Fprintf(&b, "\nDefaults (from profile): %v\n", g.AppliedDefaults)
	}
	return b.String()
}
