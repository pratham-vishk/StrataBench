package topology

import (
	"fmt"
	"strings"
)

const (
	ModeAuto   = "auto"
	ModeSingle = "single" // 1 client → 1 server
	ModePool   = "pool"   // N clients → 1 server
	ModeSweep  = "sweep"  // 1 client → N servers
	ModeShard  = "shard"  // N clients → M servers (round-robin pairs)
	ModeMatrix = "matrix" // N clients × M servers (cartesian)
)

// Assignment is one client→target benchmark job.
type Assignment struct {
	Client string // empty = local coordinator
	Target string
}

// Schedule describes how a run is distributed.
type Schedule struct {
	Mode        string
	Assignments []Assignment
}

// Build builds assignments for the given topology mode.
func Build(mode string, clients, targets []string) (Schedule, error) {
	targets = NormalizeList(targets)
	clients = NormalizeList(clients)
	if len(targets) == 0 {
		return Schedule{}, fmt.Errorf("at least one target is required")
	}
	if mode == "" {
		mode = ModeAuto
	}
	if mode == ModeAuto {
		mode = inferMode(len(clients), len(targets))
	}

	switch mode {
	case ModeSingle:
		if len(targets) != 1 {
			return Schedule{}, fmt.Errorf("topology single requires exactly one target")
		}
		if len(clients) > 1 {
			return Schedule{}, fmt.Errorf("topology single allows at most one client")
		}
		client := ""
		if len(clients) == 1 {
			client = clients[0]
		}
		return Schedule{Mode: ModeSingle, Assignments: []Assignment{{Client: client, Target: targets[0]}}}, nil

	case ModePool:
		if len(targets) != 1 {
			return Schedule{}, fmt.Errorf("topology pool requires exactly one target")
		}
		if len(clients) == 0 {
			return Schedule{}, fmt.Errorf("topology pool requires at least one client")
		}
		var as []Assignment
		for _, c := range clients {
			as = append(as, Assignment{Client: c, Target: targets[0]})
		}
		return Schedule{Mode: ModePool, Assignments: as}, nil

	case ModeSweep:
		if len(clients) > 1 {
			return Schedule{}, fmt.Errorf("topology sweep allows at most one client (use shard or matrix for multi-client)")
		}
		client := ""
		if len(clients) == 1 {
			client = clients[0]
		}
		var as []Assignment
		for _, t := range targets {
			as = append(as, Assignment{Client: client, Target: t})
		}
		return Schedule{Mode: ModeSweep, Assignments: as}, nil

	case ModeShard:
		if len(clients) == 0 {
			return Schedule{}, fmt.Errorf("topology shard requires at least one client")
		}
		var as []Assignment
		for i, c := range clients {
			as = append(as, Assignment{Client: c, Target: targets[i%len(targets)]})
		}
		return Schedule{Mode: ModeShard, Assignments: as}, nil

	case ModeMatrix:
		if len(clients) == 0 {
			return Schedule{}, fmt.Errorf("topology matrix requires at least one client")
		}
		var as []Assignment
		for _, c := range clients {
			for _, t := range targets {
				as = append(as, Assignment{Client: c, Target: t})
			}
		}
		return Schedule{Mode: ModeMatrix, Assignments: as}, nil

	default:
		return Schedule{}, fmt.Errorf("unknown topology %q (use auto, single, pool, sweep, shard, matrix)", mode)
	}
}

func inferMode(nClients, nTargets int) string {
	switch {
	case nClients <= 1 && nTargets <= 1:
		return ModeSingle
	case nClients <= 1 && nTargets > 1:
		return ModeSweep
	case nClients > 1 && nTargets <= 1:
		return ModePool
	default:
		return ModeShard
	}
}

// NormalizeList trims and deduplicates while preserving order.
func NormalizeList(items []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

// ParseCSV splits comma-separated hosts or targets.
func ParseCSV(csv string) []string {
	if csv == "" {
		return nil
	}
	return NormalizeList(strings.Split(csv, ","))
}

// MergeTargets returns targets from list, or single target as one-element slice.
func MergeTargets(single string, list []string) []string {
	if len(list) > 0 {
		return NormalizeList(list)
	}
	if strings.TrimSpace(single) != "" {
		return []string{strings.TrimSpace(single)}
	}
	return nil
}
