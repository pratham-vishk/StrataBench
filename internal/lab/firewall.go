package lab

import (
	"context"
	"fmt"
	"strings"
)

// FirewallScript returns ufw commands for lab ports.
func FirewallScript(ports []int) string {
	var b strings.Builder
	b.WriteString("# StrataBench lab firewall (ufw)\n")
	for _, p := range ports {
		fmt.Fprintf(&b, "sudo ufw allow %d/tcp\n", p)
	}
	b.WriteString("sudo ufw reload\n")
	return b.String()
}

// ApplyFirewall opens ports on a host (best-effort; requires ufw).
func ApplyFirewall(ctx context.Context, r Runner, host string, ports []int) error {
	if len(ports) == 0 {
		return nil
	}
	script := FirewallScript(ports) + "\necho firewall_ok"
	_, err := r.RunRemote(ctx, host, script)
	return err
}
