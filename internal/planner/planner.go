package planner

import (
	"strings"

	"github.com/pratham-vishk/stratabench/internal/profile"
)

// SuggestProfile maps natural-language intent to a built-in profile name.
func SuggestProfile(text string, profiles []*profile.Profile) string {
	t := strings.ToLower(text)
	candidates := map[string][]string{
		"nvme-random-oltp":     {"nvme", "oltp", "database", "random", "16k"},
		"ssd-random-4k":        {"ssd", "4k", "random"},
		"hdd-sequential-read":  {"hdd", "disk", "sequential", "read"},
		"s3-put-throughput":    {"s3", "put", "upload", "object", "write"},
		"s3-get-throughput":    {"s3", "get", "download", "read", "object"},
		"s3-cluster-put-get":   {"s3", "cluster", "warp", "distributed", "object"},
		"nvme-max-stress":      {"stress", "max", "heavy", "extreme", "nvme"},
		"spdk-nvme-peak":       {"spdk", "peak", "nvme", "userspace"},
		"afa-multi-lun":        {"afa", "array", "multi", "lun", "vdbench", "flash"},
		"s3-cluster-rdma":      {"s3", "rdma", "warp", "cluster", "object"},
		"file-parallel-read":   {"file", "nfs", "lustre", "parallel", "elbencho"},
		"vm-disk-random":       {"vm", "virtual", "guest", "disk", "random"},
		"app-kafka-producer":   {"kafka", "message", "queue", "stream", "application"},
		"app-rocksdb-read":     {"rocksdb", "kv", "embedded", "database", "application"},
		"app-postgres-tpc-c":   {"postgres", "postgresql", "database", "oltp", "tpc", "application"},
	}

	best := ""
	bestScore := 0
	for name, keywords := range candidates {
		score := 0
		for _, kw := range keywords {
			if strings.Contains(t, kw) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			best = name
		}
	}
	if best != "" {
		return best
	}
	if len(profiles) > 0 {
		return profiles[0].Name
	}
	return "nvme-random-oltp"
}
