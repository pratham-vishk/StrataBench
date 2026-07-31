package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ValidationRules struct {
	RequireDirectIO     bool     `yaml:"require_direct_io"`
	MinRuntimeSec       int      `yaml:"min_runtime_sec"`
	MinRampSec          int      `yaml:"min_ramp_sec"`
	RequirePercentiles  []float64 `yaml:"require_percentiles"`
	DatasetVsCache      string   `yaml:"dataset_vs_cache"`
}

type Profile struct {
	Name        string            `yaml:"name"`
	Version     string            `yaml:"version"`
	Layer       string            `yaml:"layer"`
	Engine      string            `yaml:"engine"`
	Description string            `yaml:"description"`
	Load        string            `yaml:"load"`
	Validation  ValidationRules   `yaml:"validation"`
	Params      map[string]any    `yaml:"params"`
	Metrics     []string          `yaml:"metrics"`
	sourcePath  string
}

func Load(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read profile: %w", err)
	}
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}
	if p.Name == "" {
		return nil, fmt.Errorf("profile missing name")
	}
	p.sourcePath = path
	return &p, nil
}

func LoadByName(profilesDir, name string) (*Profile, error) {
	if !strings.HasSuffix(name, ".yaml") {
		name += ".yaml"
	}
	return Load(filepath.Join(profilesDir, name))
}

func List(profilesDir string) ([]*Profile, error) {
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil, err
	}
	var profiles []*Profile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		p, err := Load(filepath.Join(profilesDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (p *Profile) SourcePath() string { return p.sourcePath }

func (p *Profile) ParamString(key, def string) string {
	v, ok := p.Params[key]
	if !ok {
		return def
	}
	s, ok := v.(string)
	if !ok {
		return def
	}
	return s
}

func (p *Profile) ParamInt(key string, def int) int {
	v, ok := p.Params[key]
	if !ok {
		return def
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return def
	}
}

func (p *Profile) ParamBool(key string, def bool) bool {
	v, ok := p.Params[key]
	if !ok {
		return def
	}
	b, ok := v.(bool)
	if !ok {
		return def
	}
	return b
}

func (p *Profile) ParamStringSlice(key string) []string {
	v, ok := p.Params[key]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []any:
		var out []string
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}

func (p *Profile) ToWorkload() (pattern, blockSize, datasetSize string, durationSec, rampSec, qd, threads, rwMix int, directIO bool) {
	pattern = p.ParamString("rw", p.ParamString("pattern", "read"))
	if pattern == "" {
		pattern = p.ParamString("operation", "read")
	}
	blockSize = p.ParamString("bs", p.ParamString("block_size", "4k"))
	datasetSize = p.ParamString("size", p.ParamString("dataset_size", "10g"))
	durationSec = p.ParamInt("runtime", p.ParamInt("duration_sec", 60))
	rampSec = p.ParamInt("ramp_time", p.ParamInt("ramp_time_sec", 0))
	qd = p.ParamInt("iodepth", p.ParamInt("queue_depth", 1))
	threads = p.ParamInt("numjobs", p.ParamInt("threads", 1))
	rwMix = p.ParamInt("rwmixread", p.ParamInt("read_write_mix", 100))
	directIO = p.ParamInt("direct", 0) == 1 || p.ParamBool("direct_io", false)
	return
}
