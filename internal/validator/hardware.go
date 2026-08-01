package validator

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

// HardwareRequirement describes tools and host capabilities needed for a profile.
type HardwareRequirement struct {
	Tools          []string
	MinBlockDevs   int
	RequiresNVMe   bool
	RequiresHDD    bool
	RequiresRDMA   bool
	RequiresSSH    bool
	MinMemoryGB    int
	Notes          string
}

// RequirementsFor returns expected hardware/tooling for a workload profile.
func RequirementsFor(p *profile.Profile) HardwareRequirement {
	req := HardwareRequirement{MinMemoryGB: 4, Notes: p.Description}

	switch p.Engine {
	case "fio":
		req.Tools = append(req.Tools, "fio")
	case "vdbench":
		req.Tools = append(req.Tools, "vdbench")
		req.MinBlockDevs = 2
	case "spdk":
		req.Tools = append(req.Tools, "perf", "spdk_nvme_perf")
		req.RequiresNVMe = true
	case "elbencho":
		req.Tools = append(req.Tools, "elbencho")
	case "warp":
		req.Tools = append(req.Tools, "warp")
	case "sbk":
		switch p.ParamString("driver", "") {
		case "postgresql":
			req.Tools = append(req.Tools, "pgbench")
		case "kafka":
			req.Tools = append(req.Tools, "kafka-producer-perf-test")
		case "rocksdb":
			req.Tools = append(req.Tools, "db_bench")
		}
	}

	switch p.Layer {
	case "block":
		if strings.Contains(p.Name, "hdd") {
			req.RequiresHDD = true
		}
		if strings.Contains(p.Name, "nvme") || strings.Contains(p.Name, "spdk") {
			req.RequiresNVMe = true
		}
		if strings.Contains(p.Name, "afa") {
			req.MinBlockDevs = 2
		}
	case "vm-block", "vm-file", "vm-afa":
		req.RequiresSSH = true
		if p.Layer == "vm-afa" {
			req.Tools = append(req.Tools, "vdbench")
			req.MinBlockDevs = 2
		}
		if p.Layer == "vm-file" {
			req.Tools = append(req.Tools, "elbencho")
		}
		if strings.Contains(p.Name, "nvme") {
			req.Notes += " (guest: NVMe PCIe passthrough recommended)"
		}
	case "vm-object", "object":
		if strings.Contains(p.Name, "rdma") {
			req.RequiresRDMA = true
			req.Notes += " (RDMA-capable NIC required)"
		}
	case "application", "vm-application":
		req.MinMemoryGB = 8
	}

	return req
}

// ValidateHardware checks host capabilities and tools for the given profile.
func ValidateHardware(p *profile.Profile, hw schema.HardwareSnapshot, target string, mock bool) schema.ValidationResult {
	result := schema.ValidationResult{Passed: true, RulesChecked: []string{}}
	if mock {
		result.Warnings = append(result.Warnings, schema.Warning{
			Rule: "hardware_skipped", Message: "mock mode — hardware checks skipped", Severity: "info",
		})
		return result
	}

	req := RequirementsFor(p)
	check := func(name string, fn func() error, severity string) {
		result.RulesChecked = append(result.RulesChecked, name)
		if err := fn(); err != nil {
			if severity == "error" {
				result.Passed = false
				result.Errors = append(result.Errors, err.Error())
			} else {
				result.Warnings = append(result.Warnings, schema.Warning{Rule: name, Message: err.Error(), Severity: severity})
			}
		}
	}

	for _, tool := range req.Tools {
		tool := tool
		check("tool:"+tool, func() error {
			if _, err := exec.LookPath(tool); err != nil {
				return fmt.Errorf("%s not found in PATH — install or use --mock", tool)
			}
			return nil
		}, "warning")
	}

	check("memory", func() error {
		minBytes := int64(req.MinMemoryGB) << 30
		if hw.MemoryBytes > 0 && hw.MemoryBytes < minBytes {
			return fmt.Errorf("host memory %d GB below recommended %d GB for %s", hw.MemoryBytes>>30, req.MinMemoryGB, p.Name)
		}
		return nil
	}, "warning")

	check("nvme_present", func() error {
		if !req.RequiresNVMe {
			return nil
		}
		if len(hw.NVMe) == 0 && !hasNonRotational(hw) {
			return fmt.Errorf("profile %s expects NVMe or SSD — none detected (run inventory collect)", p.Name)
		}
		return nil
	}, "warning")

	check("hdd_present", func() error {
		if !req.RequiresHDD {
			return nil
		}
		if !hasRotational(hw) {
			return fmt.Errorf("profile %s expects rotational HDD — none detected", p.Name)
		}
		return nil
	}, "warning")

	check("block_device_count", func() error {
		if req.MinBlockDevs <= 1 {
			return nil
		}
		if len(hw.BlockDevices) < req.MinBlockDevs {
			return fmt.Errorf("profile %s needs %d+ block devices, found %d", p.Name, req.MinBlockDevs, len(hw.BlockDevices))
		}
		return nil
	}, "warning")

	check("rdma", func() error {
		if !req.RequiresRDMA {
			return nil
		}
		if !hw.RDMACapable {
			return fmt.Errorf("profile %s requires RDMA — no RDMA links detected (rdma link show)", p.Name)
		}
		return nil
	}, "warning")

	check("ssh", func() error {
		if !req.RequiresSSH {
			return nil
		}
		if _, err := exec.LookPath("ssh"); err != nil {
			return fmt.Errorf("vm profile requires ssh in PATH for guest execution")
		}
		return nil
	}, "error")

	if target != "" && !strings.Contains(target, "@") && strings.HasPrefix(target, "/dev/") {
		check("target_device", func() error {
			dev := strings.TrimPrefix(target, "/dev/")
			if idx := strings.Index(dev, ","); idx > 0 {
				dev = dev[:idx]
			}
			for _, d := range hw.BlockDevices {
				if d.Name == dev || strings.HasPrefix(dev, d.Name) {
					return nil
				}
			}
			for _, n := range hw.NVMe {
				if strings.Contains(n.Device, dev) {
					return nil
				}
			}
			return fmt.Errorf("target %s not found in host inventory — verify device path", target)
		}, "warning")
	}

	return result
}

func hasRotational(hw schema.HardwareSnapshot) bool {
	for _, d := range hw.BlockDevices {
		if d.Rotational {
			return true
		}
	}
	return false
}

func hasNonRotational(hw schema.HardwareSnapshot) bool {
	for _, d := range hw.BlockDevices {
		if !d.Rotational {
			return true
		}
	}
	return len(hw.NVMe) > 0
}

// MergeValidation combines workload and hardware validation results.
func MergeValidation(a, b schema.ValidationResult) schema.ValidationResult {
	out := a
	out.RulesChecked = append(out.RulesChecked, b.RulesChecked...)
	out.Warnings = append(out.Warnings, b.Warnings...)
	out.Errors = append(out.Errors, b.Errors...)
	if !b.Passed {
		out.Passed = false
	}
	return out
}
