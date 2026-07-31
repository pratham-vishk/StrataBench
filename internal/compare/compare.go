package compare

import (
	"fmt"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func Print(a, b *schema.RunResult) {
	fmt.Printf("Compare runs\n")
	fmt.Printf("  A: %s  profile=%s  engine=%s\n", a.RunID, a.Profile, a.Engine)
	fmt.Printf("  B: %s  profile=%s  engine=%s\n", b.RunID, b.Profile, b.Engine)
	fmt.Println()
	fmt.Printf("%-20s %12s %12s %12s\n", "Metric", "A", "B", "Delta%")
	printRow("IOPS", a.Results.IOPS, b.Results.IOPS)
	printRow("Throughput MB/s", a.Results.ThroughputMBps, b.Results.ThroughputMBps)
	printRow("p99 µs", a.Results.LatencyUS.P99, b.Results.LatencyUS.P99)
}

func printRow(name string, av, bv float64) {
	delta := pctDelta(av, bv)
	fmt.Printf("%-20s %12.0f %12.0f %11.1f%%\n", name, av, bv, delta)
}

func pctDelta(a, b float64) float64 {
	if a == 0 {
		return 0
	}
	return ((b - a) / a) * 100
}
