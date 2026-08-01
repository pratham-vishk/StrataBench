package inventory

import (
	"encoding/json"
	"fmt"

	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/schema"
	"github.com/pratham-vishk/stratabench/internal/store"
)

type Record struct {
	HostID      string
	Snapshot    schema.HardwareSnapshot
	CollectedAt string
}

func HostID(snap schema.HardwareSnapshot) string {
	if snap.Hostname != "" {
		return snap.Hostname
	}
	return snap.OS + "-" + snap.Arch
}

func Collect() schema.HardwareSnapshot {
	return discovery.Snapshot()
}

func Save(st *store.Store, snap schema.HardwareSnapshot) error {
	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	return st.SaveHardware(HostID(snap), string(data))
}

func List(st *store.Store) ([]Record, error) {
	recs, err := st.ListHardware()
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, r := range recs {
		var snap schema.HardwareSnapshot
		if err := json.Unmarshal([]byte(r.SnapshotJSON), &snap); err != nil {
			return nil, err
		}
		out = append(out, Record{
			HostID:      r.HostID,
			Snapshot:    snap,
			CollectedAt: r.CollectedAt,
		})
	}
	return out, nil
}

func Print(recs []Record) {
	if len(recs) == 0 {
		fmt.Println("No hardware inventory records.")
		return
	}
	fmt.Printf("%-20s %-10s %-6s %-8s %-6s %s\n", "HOST", "OS", "CORES", "MEM_GB", "NVMe", "CPU")
	for _, r := range recs {
		s := r.Snapshot
		fmt.Printf("%-20s %-10s %-6d %-8d %-6d %s\n",
			r.HostID, s.OS, s.CPUCores, s.MemoryBytes/(1<<30), len(s.NVMe), truncate(s.CPUModel, 40))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
