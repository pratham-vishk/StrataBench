package lab

import (
	"context"
	"fmt"
	"strings"
)

// CheckReport holds lab readiness.
type CheckReport struct {
	Nodes   []NodeStatus `json:"nodes"`
	Ready   bool         `json:"ready"`
	Details []string     `json:"details"`
}

// Check validates agents, tools, and S3 endpoints.
func Check(ctx context.Context, cfg Config) (*CheckReport, error) {
	r := Runner{SSHUser: cfg.SSH.User, SSHKey: cfg.SSH.Key}
	allHosts := uniqueHosts(cfg.ClientHosts(), cfg.ServerHosts())
	extra := []string{}
	if cfg.Tools.InstallVdbench {
		extra = append(extra, "vdbench")
	}
	statuses, err := DiscoverHosts(ctx, r, allHosts, cfg.AgentPort, 9000, extra)
	if err != nil {
		return nil, err
	}
	rep := &CheckReport{Nodes: statuses, Ready: true}
	clientSet := map[string]bool{}
	for _, c := range cfg.Clients {
		clientSet[c.Host] = true
	}
	serverSet := map[string]bool{}
	for _, s := range cfg.Servers {
		serverSet[s.Host] = true
	}
	for _, st := range statuses {
		if clientSet[st.Host] {
			if !st.AgentOK {
				rep.Ready = false
				rep.Details = append(rep.Details, fmt.Sprintf("%s: agent not healthy on :%d", st.Host, cfg.AgentPort))
			}
			for _, m := range st.Missing {
				if m == "fio" || m == "warp" {
					rep.Details = append(rep.Details, fmt.Sprintf("%s: missing %s", st.Host, m))
				}
			}
		}
		if serverSet[st.Host] && cfg.NeedsMinIO() {
			if !st.S3OK {
				rep.Details = append(rep.Details, fmt.Sprintf("%s: S3/MinIO not reachable on :9000", st.Host))
			}
		}
	}
	if len(rep.Details) > 0 && rep.Ready {
		// warnings only
		for _, d := range rep.Details {
			if strings.Contains(d, "missing") {
				rep.Ready = false
			}
		}
	}
	return rep, nil
}

func PrintCheckReport(rep *CheckReport, cfg Config) {
	fmt.Println("Lab check:")
	fmt.Printf("  clients: %s\n", cfg.ClientCSV())
	fmt.Printf("  servers: %s\n", cfg.ServerCSV())
	for _, n := range rep.Nodes {
		fmt.Printf("  %s agent=%v s3=%v tools=%v missing=%v\n",
			n.Host, n.AgentOK, n.S3OK, n.Tools, n.Missing)
	}
	for _, d := range rep.Details {
		fmt.Printf("  ! %s\n", d)
	}
	if rep.Ready {
		fmt.Println("lab-check: PASS")
	} else {
		fmt.Println("lab-check: FAIL — run: stratabench lab bootstrap")
	}
}
