package planner

import (
	"regexp"
	"strconv"
	"strings"
)

// ParsedIntent holds structured benchmark parameters extracted from natural language.
type ParsedIntent struct {
	Target   string
	Targets  []string
	Clients  []string
	Topology string
	Params   map[string]any
}

// DeployContext values for Params["deploy_context"].
const (
	DeployPhysical = "physical"
	DeployVirtual  = "virtual"
)

var (
	reDuration     = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(hours?|hrs?|h|minutes?|mins?|m|seconds?|secs?|s)\b`)
	reSizeRange    = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(kib|kb|mib|mb|gib|gb|b)\s*(?:-|to|–)\s*(\d+(?:\.\d+)?)\s*(kib|kb|mib|mb|gib|gb|b)`)
	reSingleSize   = regexp.MustCompile(`(?i)(?:object|obj|block|bs|size)\s*(?:size)?\s*[:=]?\s*(\d+(?:\.\d+)?)\s*(kib|kb|mib|mb|gib|gb|b)\b`)
	reIodepth      = regexp.MustCompile(`(?i)(?:iodepth|queue[- ]?depth|qd)\s*[:=]?\s*(\d+)`)
	reNumjobs      = regexp.MustCompile(`(?i)(?:numjobs|threads?|jobs?|workers?|concurrent)\s*[:=]?\s*(\d+)`)
	reRWMix        = regexp.MustCompile(`(?i)(?:(\d+)\s*%?\s*read(?:/write)?|rwmix\s*[:=]?\s*(\d+)|(\d+)\s*/\s*(\d+)\s*read)`)
	reRamp         = regexp.MustCompile(`(?i)(?:ramp|warm[- ]?up)\s*[:=]?\s*(\d+)\s*(?:s|sec|seconds?)?`)
	reDataset      = regexp.MustCompile(`(?i)(?:dataset|working set)\s*[:=]?\s*(\d+(?:\.\d+)?)\s*(tib|gib|gb|mib|mb)\b`)
	reHostPort     = regexp.MustCompile(`\b(\d{1,3}(?:\.\d{1,3}){3})(?::(\d+))?\b`)
	reDevPath      = regexp.MustCompile(`(/dev/[a-zA-Z0-9_./,@:-]+)`)
	reClientHosts  = regexp.MustCompile(`(?i)client[s]?(?:\s+set|\s+nodes?)?\s*[:=]?\s*([0-9.,\s:/-]+)`)
	reServerHosts  = regexp.MustCompile(`(?i)server[s]?(?:\s+set|\s+nodes?)?\s*[:=]?\s*([0-9.,\s:/-]+)`)
	reTopologyWord = regexp.MustCompile(`(?i)\b(pool|sweep|shard|matrix|single)\b`)
	rePCIe         = regexp.MustCompile(`(?i)\b([0-9a-f]{4}:[0-9a-f]{2}\.[0-9a-f]\.[0-9])\b`)
	reWarehouses   = regexp.MustCompile(`(?i)(?:warehouses?|wh)\s*[:=]?\s*(\d+)`)
	reConnections  = regexp.MustCompile(`(?i)(?:connections?|conns?)\s*[:=]?\s*(\d+)`)
	reBucket       = regexp.MustCompile(`(?i)bucket\s*[:=]?\s*([a-zA-Z0-9_-]+)`)
	reTopic        = regexp.MustCompile(`(?i)topic\s*[:=]?\s*([a-zA-Z0-9_.-]+)`)
	rePatternWord  = regexp.MustCompile(`(?i)\b(sequential|seq(?:uent)?|random|rand|oltp|mixed)\b`)
	reColocated    = regexp.MustCompile(`(?i)\b(same\s+node|same\s+host|colocated|co-located|localhost|local\s+node)\b`)
)

// ParseIntent extracts benchmark parameters from natural language (no LLM required).
func ParseIntent(text string) ParsedIntent {
	out := ParsedIntent{Params: map[string]any{}}
	lower := strings.ToLower(text)

	if sec := parseDurationSec(text); sec > 0 {
		out.Params["duration_sec"] = sec
		out.Params["runtime"] = sec
	}

	if min, max, ok := parseObjectSizeRange(text); ok {
		out.Params["object_size_min"] = min
		out.Params["object_size_max"] = max
	} else if sz := parseSingleObjectSize(text, lower); sz != "" {
		out.Params["object_size"] = sz
		if strings.Contains(lower, "block") || strings.Contains(lower, "nvme") || strings.Contains(lower, "fio") {
			out.Params["bs"] = sz
		}
	}

	if m := reIodepth.FindStringSubmatch(text); len(m) >= 2 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			out.Params["iodepth"] = n
		}
	}
	if m := reNumjobs.FindStringSubmatch(text); len(m) >= 2 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			out.Params["numjobs"] = n
			out.Params["concurrent"] = n
			out.Params["threads"] = n
		}
	}
	if mix := parseRWMix(text); mix >= 0 {
		out.Params["rwmixread"] = mix
	}
	if m := reRamp.FindStringSubmatch(text); len(m) >= 2 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			out.Params["ramp_time"] = n
		}
	}
	if m := reDataset.FindStringSubmatch(text); len(m) >= 3 {
		out.Params["size"] = strings.ToLower(m[1] + m[2])
	}

	if m := reWarehouses.FindStringSubmatch(text); len(m) >= 2 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			out.Params["warehouses"] = n
		}
	}
	if m := reConnections.FindStringSubmatch(text); len(m) >= 2 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			out.Params["connections"] = n
		}
	}
	if m := reBucket.FindStringSubmatch(text); len(m) >= 2 {
		out.Params["bucket"] = m[1]
	}
	if m := reTopic.FindStringSubmatch(text); len(m) >= 2 {
		out.Params["topic"] = m[1]
	}
	if strings.Contains(lower, "rdma") {
		out.Params["rdma"] = "cpu"
	}
	if m := rePCIe.FindStringSubmatch(text); len(m) >= 2 {
		out.Params["transport"] = "trtype:PCIe traddr:" + m[1]
		if out.Target == "" {
			out.Target = m[1]
		}
	}
	parseDriverIntent(lower, &out)
	parsePatternIntent(lower, &out)
	parseDeployContext(lower, &out)

	if strings.Contains(lower, "put") && strings.Contains(lower, "get") {
		out.Params["operation"] = "mixed"
	} else if strings.Contains(lower, "put") || strings.Contains(lower, "upload") || strings.Contains(lower, "write") {
		out.Params["operation"] = "put"
	} else if strings.Contains(lower, "get") || strings.Contains(lower, "download") || strings.Contains(lower, "read") {
		if strings.Contains(lower, "object") || strings.Contains(lower, "s3") {
			out.Params["operation"] = "get"
		}
	}

	out.Clients = parseHostsFromLabel(reClientHosts, text)
	out.Targets = parseHostsFromLabel(reServerHosts, text)

	if reColocated.MatchString(text) {
		out.Params["colocated"] = true
	}
	if m := reTopologyWord.FindStringSubmatch(text); len(m) >= 2 {
		out.Topology = strings.ToLower(m[1])
	} else if len(out.Clients) > 0 && len(out.Targets) > 0 {
		out.Topology = "shard"
	} else if len(out.Clients) > 1 {
		out.Topology = "pool"
	} else if len(out.Targets) > 1 {
		out.Topology = "sweep"
	}
	inferColocatedTopology(&out)

	if dev := reDevPath.FindString(text); dev != "" {
		out.Target = dev
	}
	if out.Target == "" && len(out.Targets) == 1 {
		out.Target = out.Targets[0]
	}
	if out.Target == "" {
		// bare host:port as target when not classified as client/server
		hosts := extractHostPorts(text)
		if len(hosts) == 1 && len(out.Clients) == 0 && len(out.Targets) == 0 {
			out.Target = hosts[0]
		}
	}

	return out
}

func parseDurationSec(text string) int {
	matches := reDuration.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return 0
	}
	total := 0
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		val, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			continue
		}
		unit := strings.ToLower(m[2])
		switch {
		case strings.HasPrefix(unit, "h"):
			total += int(val * 3600)
		case unit == "m" || strings.HasPrefix(unit, "min"):
			total += int(val * 60)
		default:
			total += int(val)
		}
	}
	return total
}

func parseObjectSizeRange(text string) (min, max string, ok bool) {
	m := reSizeRange.FindStringSubmatch(text)
	if len(m) < 5 {
		return "", "", false
	}
	return normalizeSize(m[1], m[2]), normalizeSize(m[3], m[4]), true
}

func parseSingleObjectSize(text, lower string) string {
	if strings.Contains(lower, "object") || strings.Contains(lower, "obj") || strings.Contains(lower, "s3") {
		m := reSingleSize.FindStringSubmatch(text)
		if len(m) >= 3 {
			return normalizeSize(m[1], m[2])
		}
	}
	return ""
}

func normalizeSize(num, unit string) string {
	u := strings.ToUpper(strings.TrimSpace(unit))
	switch u {
	case "KB", "KIB":
		return num + "KiB"
	case "MB", "MIB":
		return num + "MiB"
	case "GB", "GIB":
		return num + "GiB"
	case "B":
		return num + "B"
	default:
		return num + u
	}
}

func parseRWMix(text string) int {
	m := reRWMix.FindStringSubmatch(text)
	if len(m) == 0 {
		return -1
	}
	for i := 1; i < len(m); i++ {
		if m[i] != "" {
			if n, err := strconv.Atoi(m[i]); err == nil {
				return n
			}
		}
	}
	return -1
}

func parseHostsFromLabel(re *regexp.Regexp, text string) []string {
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return nil
	}
	blob := strings.TrimSpace(m[1])
	if idx := strings.Index(strings.ToLower(blob), "server"); idx > 0 {
		blob = strings.TrimSpace(blob[:idx])
	}
	if idx := strings.Index(strings.ToLower(blob), "client"); idx > 0 {
		blob = strings.TrimSpace(blob[:idx])
	}
	return extractHostPorts(blob)
}

func extractHostPorts(blob string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range reHostPort.FindAllStringSubmatch(blob, -1) {
		host := m[1]
		port := m[2]
		if port == "" {
			port = defaultPortForContext(blob)
		}
		addr := host + ":" + port
		if !seen[addr] {
			seen[addr] = true
			out = append(out, addr)
		}
	}
	return out
}

func defaultPortForContext(blob string) string {
	lower := strings.ToLower(blob)
	if strings.Contains(lower, "agent") || strings.Contains(lower, "client") {
		return "7777"
	}
	if strings.Contains(lower, "s3") || strings.Contains(lower, "minio") || strings.Contains(lower, "object") {
		return "9000"
	}
	return "7777"
}

func parseDriverIntent(lower string, out *ParsedIntent) {
	switch {
	case strings.Contains(lower, "postgres") || strings.Contains(lower, "postgresql") || strings.Contains(lower, "pgbench"):
		out.Params["driver"] = "postgresql"
	case strings.Contains(lower, "kafka"):
		out.Params["driver"] = "kafka"
	case strings.Contains(lower, "rocksdb"):
		out.Params["driver"] = "rocksdb"
	}
}

func parsePatternIntent(lower string, out *ParsedIntent) {
	if m := rePatternWord.FindStringSubmatch(lower); len(m) >= 2 {
		switch strings.ToLower(m[1]) {
		case "sequential", "seq", "sequent":
			out.Params["rw"] = "read"
			out.Params["pattern"] = "seqread"
		case "random", "rand", "oltp":
			out.Params["rw"] = "randrw"
			out.Params["pattern"] = "randread"
		case "mixed":
			out.Params["rw"] = "randrw"
			out.Params["operation"] = "mixed"
		}
	}
}

func parseDeployContext(lower string, out *ParsedIntent) {
	switch {
	case strings.Contains(lower, "virtual") || reWordVM.MatchString(lower) ||
		strings.Contains(lower, "guest") || strings.Contains(lower, "hci") ||
		strings.Contains(lower, "inside vm"):
		out.Params["deploy_context"] = DeployVirtual
	case strings.Contains(lower, "physical") || strings.Contains(lower, "bare metal") ||
		strings.Contains(lower, "baremetal") || strings.Contains(lower, "bare-metal") ||
		strings.Contains(lower, "bare metal host"):
		out.Params["deploy_context"] = DeployPhysical
	}
}

// hostOf returns the host portion of host:port (or the whole string if no port).
func hostOf(addr string) string {
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, "root@") {
		rest := strings.TrimPrefix(addr, "root@")
		if i := strings.Index(rest, ":"); i > 0 {
			return rest[:i]
		}
		if i := strings.Index(rest, "/"); i > 0 {
			return rest[:i]
		}
		return rest
	}
	if i := strings.LastIndex(addr, ":"); i > 0 {
		return addr[:i]
	}
	return addr
}

func normalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "127.0.0.1" || host == "::1" {
		return "localhost"
	}
	return host
}

func uniqueHosts(addrs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, a := range addrs {
		h := normalizeHost(hostOf(a))
		if h == "" || seen[h] {
			continue
		}
		seen[h] = true
		out = append(out, h)
	}
	return out
}

// inferColocatedTopology sets single-node topology when clients and servers share one host.
func inferColocatedTopology(out *ParsedIntent) {
	if len(out.Clients) == 0 || len(out.Targets) == 0 {
		return
	}
	all := append(append([]string{}, out.Clients...), out.Targets...)
	hosts := uniqueHosts(all)
	if len(hosts) == 1 {
		out.Topology = "single"
		out.Params["colocated"] = true
	}
}

// MergePlan combines keyword/LLM plan with parsed intent. CLI opts override when non-empty.
func MergePlan(base PlanResult, parsed ParsedIntent, cliTarget string, cliTargets, cliClients []string, cliTopology string) PlanResult {
	out := base
	if out.Params == nil {
		out.Params = map[string]any{}
	}
	for k, v := range parsed.Params {
		if _, exists := out.Params[k]; !exists {
			out.Params[k] = v
		}
	}
	if cliTarget != "" {
		out.Target = cliTarget
	} else if out.Target == "" {
		out.Target = parsed.Target
	}
	if len(cliTargets) > 0 {
		out.Targets = cliTargets
	} else if len(out.Targets) == 0 {
		out.Targets = parsed.Targets
	}
	if len(cliClients) > 0 {
		out.Clients = cliClients
	} else if len(out.Clients) == 0 {
		out.Clients = parsed.Clients
	}
	if cliTopology != "" && cliTopology != "auto" {
		out.Topology = cliTopology
	} else if out.Topology == "" {
		out.Topology = parsed.Topology
	}
	if out.Topology == "" {
		out.Topology = "auto"
	}
	return out
}
