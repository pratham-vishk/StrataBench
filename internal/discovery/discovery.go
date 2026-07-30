package discovery

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// Snapshot collects host metadata used for honest validation and result records.
func Snapshot() schema.HardwareSnapshot {
	snap := schema.HardwareSnapshot{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
	if v, ok := os.LookupEnv("STRATABENCH_MOCK_CACHE_BYTES"); ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			snap.CacheBytes = n
		}
	}
	if snap.CacheBytes == 0 {
		snap.CacheBytes = estimateCacheBytes()
	}
	snap.MemoryBytes = estimateMemoryBytes()
	snap.CPUCores = runtime.NumCPU()
	snap.CPUModel = "unknown"
	return snap
}

func estimateMemoryBytes() int64 {
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return 8 << 30
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kb, err := strconv.ParseInt(fields[1], 10, 64)
					if err == nil {
						return kb * 1024
					}
				}
			}
		}
	}
	return 8 << 30
}

func estimateCacheBytes() int64 {
	// Conservative default when array cache size is unknown.
	return 32 << 30
}
