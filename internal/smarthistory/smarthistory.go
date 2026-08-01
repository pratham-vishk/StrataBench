package smarthistory

import (
	"encoding/json"
	"fmt"

	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/inventory"
	"github.com/pratham-vishk/stratabench/internal/schema"
	"github.com/pratham-vishk/stratabench/internal/store"
)

type Record struct {
	HostID      string
	Device      string
	Reading     schema.SMARTReading
	CollectedAt string
}

func CollectAndSave(st *store.Store) (int, error) {
	snap := discovery.Snapshot()
	hostID := inventory.HostID(snap)
	readings := discovery.CollectSMART()
	for _, r := range readings {
		data, err := json.Marshal(r)
		if err != nil {
			return 0, err
		}
		if err := st.SaveSMART(hostID, r.Device, string(data)); err != nil {
			return 0, err
		}
	}
	return len(readings), nil
}

func List(st *store.Store, limit int) ([]Record, error) {
	recs, err := st.ListSMART(limit)
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, r := range recs {
		var reading schema.SMARTReading
		if err := json.Unmarshal([]byte(r.ReadingJSON), &reading); err != nil {
			return nil, err
		}
		out = append(out, Record{
			HostID:      r.HostID,
			Device:      r.Device,
			Reading:     reading,
			CollectedAt: r.CollectedAt,
		})
	}
	return out, nil
}

func Print(recs []Record) {
	if len(recs) == 0 {
		fmt.Println("No SMART history records.")
		return
	}
	fmt.Printf("%-16s %-14s %-20s %-5s %-6s %-6s %s\n", "HOST", "DEVICE", "MODEL", "TEMP", "WEAR%", "POH", "HEALTH")
	for _, r := range recs {
		health := "OK"
		if !r.Reading.HealthPassed {
			health = "FAIL"
		}
		fmt.Printf("%-16s %-14s %-20s %-5d %-6d %-6d %s\n",
			r.HostID, r.Device, truncate(r.Reading.Model, 20),
			r.Reading.TemperatureC, r.Reading.WearPercent, r.Reading.PowerOnHours, health)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
