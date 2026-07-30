package validator_test

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/validator"
)

func TestValidateDatasetGtCacheFails(t *testing.T) {
	p := &profile.Profile{
		Name:  "test",
		Layer: "block",
		Validation: profile.ValidationRules{
			DatasetVsCache: "gt",
		},
		Params: map[string]any{
			"size":    "10g",
			"runtime": 600,
			"direct":  1,
		},
	}
	res := validator.Validate(p, validator.Options{CacheBytes: 32 << 30})
	if res.Passed {
		t.Fatal("expected validation to fail when dataset < cache")
	}
}

func TestValidateDatasetGtCachePasses(t *testing.T) {
	p := &profile.Profile{
		Name:  "test",
		Layer: "block",
		Validation: profile.ValidationRules{
			DatasetVsCache: "gt",
		},
		Params: map[string]any{
			"size":    "500g",
			"runtime": 600,
			"direct":  1,
		},
	}
	res := validator.Validate(p, validator.Options{CacheBytes: 32 << 30})
	if !res.Passed {
		t.Fatalf("expected pass, got errors: %v", res.Errors)
	}
}

func TestValidateMinRuntime(t *testing.T) {
	p := &profile.Profile{
		Name:  "test",
		Layer: "block",
		Validation: profile.ValidationRules{
			MinRuntimeSec: 600,
		},
		Params: map[string]any{
			"runtime": 120,
			"size":    "500g",
			"direct":  1,
		},
	}
	res := validator.Validate(p, validator.Options{})
	if res.Passed {
		t.Fatal("expected min runtime failure")
	}
}

func TestParseSize(t *testing.T) {
	cases := map[string]int64{
		"200g":  200 * 1e9,
		"16k":   16 * 1e3,
		"1m":    1e6,
		"500g":  500 * 1e9,
	}
	for in, want := range cases {
		got, err := validator.ParseSize(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if got != want {
			t.Fatalf("%s: got %d want %d", in, got, want)
		}
	}
}
