package profile_test

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/profile"
)

func TestApplyOverridesDurationAlias(t *testing.T) {
	p := &profile.Profile{Params: map[string]any{"runtime": 600}}
	p.ApplyOverrides(map[string]any{"duration_sec": 3600})
	if p.ParamInt("runtime", 0) != 3600 {
		t.Fatalf("runtime=%d", p.ParamInt("runtime", 0))
	}
	if p.ParamInt("duration_sec", 0) != 3600 {
		t.Fatalf("duration_sec=%d", p.ParamInt("duration_sec", 0))
	}
}

func TestApplyOverridesObjectSizeRange(t *testing.T) {
	p := &profile.Profile{Params: map[string]any{}}
	p.ApplyOverrides(map[string]any{
		"object_size_min": "3KiB",
		"object_size_max": "100KiB",
	})
	if p.ParamString("object_size", "") != "3KiB-100KiB" {
		t.Fatalf("object_size=%s", p.ParamString("object_size", ""))
	}
}

func TestCloneIndependent(t *testing.T) {
	p := &profile.Profile{Params: map[string]any{"runtime": 60}}
	cp := p.Clone()
	cp.ApplyOverrides(map[string]any{"runtime": 120})
	if p.ParamInt("runtime", 0) != 60 {
		t.Fatal("original mutated")
	}
}
