package orchestrator

import (
	"github.com/pratham-vishk/stratabench/internal/profile"
)

// applyWarpClients injects native Warp coordinator clients when the profile leaves warp_clients empty.
func applyWarpClients(p *profile.Profile, warpClients []string) *profile.Profile {
	if p == nil || p.Engine != "warp" || len(warpClients) == 0 {
		return p
	}
	if len(p.ParamStringSlice("warp_clients")) > 0 {
		return p
	}
	cp := p.Clone()
	if cp.Params == nil {
		cp.Params = map[string]any{}
	}
	cp.Params["warp_clients"] = append([]string(nil), warpClients...)
	return cp
}
