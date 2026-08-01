package validator

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestRequirementsForNVMe(t *testing.T) {
	p := &profile.Profile{Name: "nvme-random-oltp", Layer: "block", Engine: "fio"}
	req := RequirementsFor(p)
	if !req.RequiresNVMe {
		t.Fatal("expected RequiresNVMe")
	}
}

func TestRequirementsForAFA(t *testing.T) {
	p := &profile.Profile{Name: "afa-multi-lun", Layer: "block", Engine: "vdbench"}
	req := RequirementsFor(p)
	if req.MinBlockDevs < 2 {
		t.Fatal("expected MinBlockDevs >= 2")
	}
}

func TestValidateHardwareMockSkips(t *testing.T) {
	p := &profile.Profile{Name: "nvme-random-oltp", Layer: "block", Engine: "fio"}
	res := ValidateHardware(p, schema.HardwareSnapshot{}, "/dev/nvme0n1", true)
	if !res.Passed {
		t.Fatal("mock should pass")
	}
}

func TestValidateHardwareRDMA(t *testing.T) {
	p := &profile.Profile{Name: "s3-cluster-rdma", Layer: "object", Engine: "warp"}
	hw := schema.HardwareSnapshot{RDMACapable: false}
	res := ValidateHardware(p, hw, "10.0.1.10:9000", false)
	found := false
	for _, w := range res.Warnings {
		if w.Rule == "rdma" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected rdma warning")
	}
}

func TestMergeValidation(t *testing.T) {
	a := schema.ValidationResult{Passed: true, RulesChecked: []string{"a"}}
	b := schema.ValidationResult{Passed: false, Errors: []string{"fail"}, RulesChecked: []string{"b"}}
	m := MergeValidation(a, b)
	if m.Passed || len(m.Errors) != 1 || len(m.RulesChecked) != 2 {
		t.Fatalf("unexpected merge: %+v", m)
	}
}
