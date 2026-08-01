package report

import (
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// WorkloadLabels are customer-facing metric names that adapt to block vs S3/object runs.
type WorkloadLabels struct {
	OpsRate       string // IOPS | Ops/s
	OpsUnit       string // rec/s | obj/s
	ReadOp        string // Read IOPS | GET ops/s
	WriteOp       string // Write IOPS | PUT ops/s
	TotalVolume   string // Total records | Total objects
	ThroughputRec string // records/s | objects/s
	IsObject      bool
	Operation     string // put | get | mixed | ...
}

func workloadLabels(run *schema.RunResult) WorkloadLabels {
	lbl := WorkloadLabels{
		OpsRate:       "IOPS",
		OpsUnit:       "rec/s",
		ReadOp:        "Read IOPS",
		WriteOp:       "Write IOPS",
		TotalVolume:   "Total records",
		ThroughputRec: "records/s",
	}
	if !isObjectLayer(run) {
		return lbl
	}
	lbl.IsObject = true
	lbl.OpsRate = "Ops/s"
	lbl.OpsUnit = "obj/s"
	lbl.ReadOp = "GET ops/s"
	lbl.WriteOp = "PUT ops/s"
	lbl.TotalVolume = "Total objects"
	lbl.ThroughputRec = "objects/s"
	lbl.Operation = strings.ToLower(strings.TrimSpace(run.Workload.Pattern))
	if lbl.Operation == "" {
		lbl.Operation = inferObjectOperation(run.Profile)
	}
	return lbl
}

func isObjectLayer(run *schema.RunResult) bool {
	layer := strings.ToLower(strings.TrimSpace(run.Layer))
	return layer == "object" || strings.HasPrefix(layer, "vm-object")
}

func inferObjectOperation(profile string) string {
	p := strings.ToLower(profile)
	switch {
	case strings.Contains(p, "put") && strings.Contains(p, "get"):
		return "mixed"
	case strings.Contains(p, "put"):
		return "put"
	case strings.Contains(p, "get"):
		return "get"
	case strings.Contains(p, "mixed"):
		return "mixed"
	case strings.Contains(p, "delete"):
		return "delete"
	default:
		return "put"
	}
}

func objectHasPutGet(run *schema.RunResult, lbl WorkloadLabels) bool {
	if lbl.Operation == "mixed" {
		return true
	}
	if run.Results.ReadIOPS > 0 && run.Results.WriteIOPS > 0 {
		return true
	}
	for _, c := range run.Clients {
		if c.Results.ReadIOPS > 0 && c.Results.WriteIOPS > 0 {
			return true
		}
	}
	return lbl.Operation == "mixed"
}
