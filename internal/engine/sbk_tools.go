package engine

import (
	"os/exec"
)

// SBKDriverProbe reports whether a native SBK driver binary is on PATH.
type SBKDriverProbe struct {
	Driver    string `json:"driver"`
	Tool      string `json:"tool"`
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
}

// SBKToolReport summarizes native SBK driver availability on the local host.
type SBKToolReport struct {
	Drivers      []SBKDriverProbe `json:"drivers"`
	AllAvailable bool             `json:"all_available"`
}

var sbkDriverTools = []struct {
	driver string
	tools  []string
}{
	{driver: "postgresql", tools: []string{"pgbench"}},
	{driver: "rocksdb", tools: []string{"db_bench"}},
	{driver: "kafka", tools: []string{"kafka-producer-perf-test.sh", "kafka-producer-perf-test"}},
}

// ProbeSBKDrivers checks PATH for pgbench, db_bench, and kafka-producer-perf-test.
func ProbeSBKDrivers() SBKToolReport {
	rep := SBKToolReport{}
	for _, spec := range sbkDriverTools {
		probe := SBKDriverProbe{Driver: spec.driver, Tool: spec.tools[0]}
		for _, tool := range spec.tools {
			if path, err := exec.LookPath(tool); err == nil {
				probe.Available = true
				probe.Path = path
				probe.Tool = tool
				break
			}
		}
		rep.Drivers = append(rep.Drivers, probe)
		if !probe.Available {
			rep.AllAvailable = false
		}
	}
	if len(rep.Drivers) > 0 {
		rep.AllAvailable = true
		for _, d := range rep.Drivers {
			if !d.Available {
				rep.AllAvailable = false
				break
			}
		}
	}
	return rep
}

func rocksDBBenchmark(profilePattern string) string {
	switch profilePattern {
	case "write", "randwrite":
		return "fillrandom"
	case "read", "randread", "readrandom":
		return "readrandom"
	default:
		return "readrandom"
	}
}
