package discovery

import (
	"encoding/json"
	"os/exec"
	"runtime"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// CollectSMART reads SMART data via smartctl on Linux block/NVMe devices.
func CollectSMART() []schema.SMARTReading {
	if runtime.GOOS != "linux" {
		return nil
	}
	if _, err := exec.LookPath("smartctl"); err != nil {
		return nil
	}

	var devices []string
	for _, d := range listBlockDevices() {
		devices = append(devices, "/dev/"+d.Name)
	}
	for _, d := range listNVMeDevices() {
		if d.Device != "" {
			devices = append(devices, d.Device)
		}
	}

	var readings []schema.SMARTReading
	seen := map[string]bool{}
	for _, dev := range devices {
		if seen[dev] {
			continue
		}
		seen[dev] = true
		if r, ok := readSMART(dev); ok {
			readings = append(readings, r)
		}
	}
	return readings
}

func readSMART(device string) (schema.SMARTReading, bool) {
	out, err := exec.Command("smartctl", "-j", "-a", device).Output()
	if err != nil {
		return schema.SMARTReading{}, false
	}

	var payload struct {
		ModelName    string `json:"model_name"`
		SerialNumber string `json:"serial_number"`
		SmartStatus  struct {
			Passed bool `json:"passed"`
		} `json:"smart_status"`
		Temperature struct {
			Current int `json:"current"`
		} `json:"temperature"`
		PowerOnTime struct {
			Hours int `json:"hours"`
		} `json:"power_on_time"`
		NVMeLog struct {
			PercentageUsed int `json:"percentage_used"`
		} `json:"nvme_smart_health_information_log"`
		ATA struct {
			Table []struct {
				NameStr string `json:"name"`
				Raw     struct {
					Value int `json:"value"`
				} `json:"raw"`
			} `json:"table"`
		} `json:"ata_smart_attributes"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return schema.SMARTReading{}, false
	}

	r := schema.SMARTReading{
		Device:       device,
		Model:        payload.ModelName,
		Serial:       payload.SerialNumber,
		TemperatureC: payload.Temperature.Current,
		PowerOnHours: payload.PowerOnTime.Hours,
		WearPercent:  payload.NVMeLog.PercentageUsed,
		HealthPassed: payload.SmartStatus.Passed,
	}
	for _, attr := range payload.ATA.Table {
		if strings.Contains(strings.ToLower(attr.NameStr), "reallocated") {
			r.Reallocated = attr.Raw.Value
		}
	}
	return r, true
}

// ParseSMARTTemperature extracts temperature from smartctl JSON for tests.
func ParseSMARTTemperature(jsonText string) int {
	var payload struct {
		Temperature struct {
			Current int `json:"current"`
		} `json:"temperature"`
	}
	if err := json.Unmarshal([]byte(jsonText), &payload); err != nil {
		return 0
	}
	return payload.Temperature.Current
}
