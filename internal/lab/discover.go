package lab

import (
	"context"
	"fmt"
	"strings"
)

// NodeStatus is the result of probing one host.
type NodeStatus struct {
	Host       string   `json:"host"`
	AgentOK    bool     `json:"agent_ok"`
	AgentPort  int      `json:"agent_port"`
	S3OK       bool     `json:"s3_ok"`
	S3Port     int      `json:"s3_port"`
	Tools      []string `json:"tools"`
	Missing    []string `json:"missing"`
	Suggested  string   `json:"suggested_role"` // client, s3, both
}

var defaultTools = []string{"fio", "warp"}

// DiscoverHosts probes hosts via SSH and HTTP to classify roles and tool gaps.
func DiscoverHosts(ctx context.Context, r Runner, hosts []string, agentPort, s3Port int, extraTools []string) ([]NodeStatus, error) {
	tools := append([]string{}, defaultTools...)
	tools = append(tools, extraTools...)
	var out []NodeStatus
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		st := NodeStatus{Host: host, AgentPort: agentPort, S3Port: s3Port}
		if body, err := HTTPGet(ctx, fmt.Sprintf("http://%s:%d/v1/health", host, agentPort)); err == nil && strings.Contains(body, "ok") {
			st.AgentOK = true
		}
		if _, err := HTTPGet(ctx, fmt.Sprintf("http://%s:%d/minio/health/live", host, s3Port)); err == nil {
			st.S3OK = true
		}
		probe := `for t in fio warp vdbench elbencho; do command -v $t && echo "have:$t"; done`
		if remote, err := r.RunRemote(ctx, host, probe); err == nil {
			for _, line := range strings.Split(remote, "\n") {
				if strings.HasPrefix(line, "have:") {
					st.Tools = append(st.Tools, strings.TrimPrefix(line, "have:"))
				}
			}
		}
		have := map[string]bool{}
		for _, t := range st.Tools {
			have[t] = true
		}
		for _, t := range tools {
			if !have[t] {
				st.Missing = append(st.Missing, t)
			}
		}
		switch {
		case st.AgentOK && st.S3OK:
			st.Suggested = "both"
		case st.S3OK:
			st.Suggested = "s3"
		case st.AgentOK || len(st.Missing) < len(tools):
			st.Suggested = "client"
		default:
			st.Suggested = "client"
		}
		out = append(out, st)
	}
	return out, nil
}

// ApplyDiscovery updates config clients/servers from discovery results.
func ApplyDiscovery(cfg *Config, statuses []NodeStatus) {
	cfg.Clients = nil
	cfg.Servers = nil
	for _, st := range statuses {
		switch st.Suggested {
		case "s3":
			cfg.Servers = append(cfg.Servers, Node{Host: st.Host, Port: st.S3Port, Role: "s3"})
		case "both":
			cfg.Clients = append(cfg.Clients, Node{Host: st.Host, Port: st.AgentPort, Role: "client"})
			cfg.Servers = append(cfg.Servers, Node{Host: st.Host, Port: st.S3Port, Role: "s3"})
		default:
			cfg.Clients = append(cfg.Clients, Node{Host: st.Host, Port: st.AgentPort, Role: "client"})
		}
	}
}
