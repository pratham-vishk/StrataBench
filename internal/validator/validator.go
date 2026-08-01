package validator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

type Options struct {
	CacheBytes    int64
	CheckHardware bool
	Target        string
	Mock          bool
	Hardware      schema.HardwareSnapshot
}

func Validate(p *profile.Profile, opts Options) schema.ValidationResult {
	result := schema.ValidationResult{
		Passed:       true,
		RulesChecked: []string{},
		Warnings:     []schema.Warning{},
	}

	_, _, datasetSize, durationSec, rampSec, _, _, _, directIO := p.ToWorkload()

	check := func(name string, fn func() error) {
		result.RulesChecked = append(result.RulesChecked, name)
		if err := fn(); err != nil {
			result.Passed = false
			result.Errors = append(result.Errors, err.Error())
		}
	}

	check("direct_io", func() error {
		if !p.Validation.RequireDirectIO {
			return nil
		}
		if p.Layer == "object" {
			return nil
		}
		if !directIO {
			return fmt.Errorf("profile requires direct I/O (bypass page cache) but direct_io is false")
		}
		return nil
	})

	check("min_runtime", func() error {
		min := p.Validation.MinRuntimeSec
		if min == 0 {
			return nil
		}
		if durationSec < min {
			return fmt.Errorf("runtime %ds is below minimum %ds for steady-state measurement", durationSec, min)
		}
		return nil
	})

	check("min_ramp", func() error {
		min := p.Validation.MinRampSec
		if min == 0 {
			return nil
		}
		if rampSec < min {
			return fmt.Errorf("ramp_time %ds is below minimum %ds", rampSec, min)
		}
		return nil
	})

	check("tail_latency", func() error {
		if len(p.Validation.RequirePercentiles) == 0 {
			return nil
		}
		pl := p.ParamString("percentile_list", "")
		if pl == "" && p.Engine != "fio" {
			result.Warnings = append(result.Warnings, schema.Warning{
				Rule:     "tail_latency",
				Message:  "percentile_list not set in profile params; engine may still report percentiles",
				Severity: "warning",
			})
			return nil
		}
		for _, req := range p.Validation.RequirePercentiles {
			token := fmt.Sprintf("%g", req)
			if pl != "" && !strings.Contains(pl, token) {
				return fmt.Errorf("percentile_list missing required p%s", token)
			}
		}
		return nil
	})

	check("dataset_gt_cache", func() error {
		if p.Validation.DatasetVsCache != "gt" {
			return nil
		}
		if p.Layer == "object" {
			return nil
		}
		dsBytes, err := ParseSize(datasetSize)
		if err != nil {
			return fmt.Errorf("invalid dataset size %q: %w", datasetSize, err)
		}
		cache := opts.CacheBytes
		if cache == 0 {
			cache = 32 << 30
		}
		if dsBytes <= cache {
			return fmt.Errorf("dataset %s (%d bytes) must exceed cache estimate (%d bytes) — otherwise you measure cache, not storage", datasetSize, dsBytes, cache)
		}
		return nil
	})

	check("steady_state_window", func() error {
		if durationSec > 0 && rampSec > 0 && durationSec <= rampSec+30 {
			result.Warnings = append(result.Warnings, schema.Warning{
				Rule:     "steady_state_window",
				Message:  fmt.Sprintf("runtime %ds may be too short after ramp %ds to reach steady state", durationSec, rampSec),
				Severity: "warning",
			})
		}
		return nil
	})

	if opts.CheckHardware {
		hw := opts.Hardware
		if hw.Hostname == "" {
			hw = discovery.Snapshot()
		}
		hwResult := ValidateHardware(p, hw, opts.Target, opts.Mock)
		result = MergeValidation(result, hwResult)
	}

	return result
}

func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty size")
	}
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "tib"):
		mult = 1 << 40
		s = strings.TrimSuffix(s, "tib")
	case strings.HasSuffix(s, "gib"):
		mult = 1 << 30
		s = strings.TrimSuffix(s, "gib")
	case strings.HasSuffix(s, "mib"):
		mult = 1 << 20
		s = strings.TrimSuffix(s, "mib")
	case strings.HasSuffix(s, "kib"):
		mult = 1 << 10
		s = strings.TrimSuffix(s, "kib")
	case strings.HasSuffix(s, "tb"):
		mult = 1e12
		s = strings.TrimSuffix(s, "tb")
	case strings.HasSuffix(s, "gb"), strings.HasSuffix(s, "g"):
		mult = 1e9
		s = strings.TrimSuffix(strings.TrimSuffix(s, "gb"), "g")
	case strings.HasSuffix(s, "mb"), strings.HasSuffix(s, "m"):
		mult = 1e6
		s = strings.TrimSuffix(strings.TrimSuffix(s, "mb"), "m")
	case strings.HasSuffix(s, "kb"), strings.HasSuffix(s, "k"):
		mult = 1e3
		s = strings.TrimSuffix(strings.TrimSuffix(s, "kb"), "k")
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	}
	return int64(n * float64(mult)), nil
}
