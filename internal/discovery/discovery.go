package discovery

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// Snapshot collects host metadata used for honest validation and result records.
func Snapshot() schema.HardwareSnapshot {
	snap := schema.HardwareSnapshot{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Hostname: hostname(),
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
	snap.CPUModel = cpuModel()
	snap.RDMACapable = rdmaCapable()
	if runtime.GOOS == "linux" {
		snap.BlockDevices = listBlockDevices()
		snap.NVMe = listNVMeDevices()
	}
	return snap
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func cpuModel() string {
	if runtime.GOOS != "linux" {
		return "unknown"
	}
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "unknown"
}

func rdmaCapable() bool {
	if _, err := exec.LookPath("rdma"); err != nil {
		return false
	}
	out, err := exec.Command("rdma", "link", "show").Output()
	return err == nil && len(out) > 0
}

func listBlockDevices() []schema.BlockDevice {
	entries, err := os.ReadDir("/sys/block")
	if err != nil {
		return nil
	}
	var devices []schema.BlockDevice
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") {
			continue
		}
		dev := schema.BlockDevice{Name: name}
		if b, err := os.ReadFile(filepath.Join("/sys/block", name, "size")); err == nil {
			if sectors, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64); err == nil {
				dev.SizeBytes = sectors * 512
			}
		}
		if b, err := os.ReadFile(filepath.Join("/sys/block", name, "queue", "rotational")); err == nil {
			dev.Rotational = strings.TrimSpace(string(b)) == "1"
		}
		if b, err := os.ReadFile(filepath.Join("/sys/block", name, "device", "model")); err == nil {
			dev.Model = strings.TrimSpace(string(b))
		}
		devices = append(devices, dev)
	}
	return devices
}

func listNVMeDevices() []schema.NVMEDevice {
	if _, err := exec.LookPath("nvme"); err != nil {
		return nil
	}
	out, err := exec.Command("nvme", "list", "-o", "json").Output()
	if err != nil {
		return nil
	}
	var payload struct {
		Devices []struct {
			DevicePath   string `json:"DevicePath"`
			ModelNumber  string `json:"ModelNumber"`
			Firmware     string `json:"Firmware"`
			SerialNumber string `json:"SerialNumber"`
		} `json:"Devices"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil
	}
	var devices []schema.NVMEDevice
	for _, d := range payload.Devices {
		devices = append(devices, schema.NVMEDevice{
			Device:   d.DevicePath,
			Model:    d.ModelNumber,
			Firmware: d.Firmware,
			Serial:   d.SerialNumber,
		})
	}
	return devices
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
	return 32 << 30
}
