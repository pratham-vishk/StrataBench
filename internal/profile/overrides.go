package profile

import (
	"fmt"
	"strconv"
	"strings"
)

// Clone returns a deep copy of the profile (params map copied).
func (p *Profile) Clone() *Profile {
	if p == nil {
		return nil
	}
	cp := *p
	cp.Params = make(map[string]any, len(p.Params))
	for k, v := range p.Params {
		cp.Params[k] = v
	}
	return &cp
}

// ApplyOverrides merges intent-derived parameters into the profile.
func (p *Profile) ApplyOverrides(overrides map[string]any) {
	if p == nil || len(overrides) == 0 {
		return
	}
	if p.Params == nil {
		p.Params = make(map[string]any)
	}
	for k, v := range overrides {
		if v == nil {
			continue
		}
		p.Params[normalizeParamKey(k)] = v
	}
	syncParamAliases(p)
}

func normalizeParamKey(k string) string {
	switch strings.ToLower(strings.TrimSpace(k)) {
	case "duration", "duration_seconds":
		return "duration_sec"
	case "runtime_sec", "runtime_seconds":
		return "runtime"
	case "blocksize", "block_size_bytes":
		return "bs"
	case "objectsize", "obj_size":
		return "object_size"
	case "threads", "jobs":
		return "numjobs"
	case "queue_depth", "qd":
		return "iodepth"
	case "read_write_mix", "read_pct":
		return "rwmixread"
	case "ramp", "warmup", "warmup_sec":
		return "ramp_time"
	case "dataset", "dataset_size":
		return "size"
	case "obj_size_min":
		return "object_size_min"
	case "obj_size_max":
		return "object_size_max"
	default:
		return strings.ToLower(strings.TrimSpace(k))
	}
}

func syncParamAliases(p *Profile) {
	if v, ok := p.Params["duration_sec"]; ok {
		p.Params["runtime"] = v
	} else if v, ok := p.Params["runtime"]; ok {
		p.Params["duration_sec"] = v
	}
	if v, ok := p.Params["ramp_time_sec"]; ok {
		if _, has := p.Params["ramp_time"]; !has {
			p.Params["ramp_time"] = v
		}
	}
	if v, ok := p.Params["block_size"]; ok {
		if _, has := p.Params["bs"]; !has {
			p.Params["bs"] = v
		}
	}
	if v, ok := p.Params["bs"]; ok {
		if _, has := p.Params["block_size"]; !has {
			p.Params["block_size"] = v
		}
	}
	if v, ok := p.Params["threads"]; ok {
		if _, has := p.Params["numjobs"]; !has {
			p.Params["numjobs"] = v
		}
	}
	if v, ok := p.Params["numjobs"]; ok {
		if _, has := p.Params["threads"]; !has {
			p.Params["threads"] = v
		}
	}
	if v, ok := p.Params["queue_depth"]; ok {
		if _, has := p.Params["iodepth"]; !has {
			p.Params["iodepth"] = v
		}
	}

	min, hasMin := p.Params["object_size_min"]
	max, hasMax := p.Params["object_size_max"]
	switch {
	case hasMin && hasMax:
		p.Params["object_size"] = formatAny(min) + "-" + formatAny(max)
	case hasMin:
		p.Params["object_size"] = formatAny(min)
	case hasMax:
		p.Params["object_size"] = formatAny(max)
	}
}

func formatAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return fmt.Sprint(v)
	}
}
