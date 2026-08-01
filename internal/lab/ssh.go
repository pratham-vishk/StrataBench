package lab

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Runner executes commands locally or over SSH.
type Runner struct {
	SSHUser string
	SSHKey  string
}

func (r Runner) sshArgs(host string) []string {
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=15",
	}
	if r.SSHKey != "" {
		args = append(args, "-i", expandHome(r.SSHKey))
	}
	args = append(args, fmt.Sprintf("%s@%s", r.SSHUser, host))
	return args
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	return path
}

func (r Runner) RunRemote(ctx context.Context, host, script string) (string, error) {
	args := append(r.sshArgs(host), "bash", "-s")
	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdin = strings.NewReader(script)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("%s: %w\n%s", host, err, out.String())
	}
	return out.String(), nil
}

func (r Runner) SCP(ctx context.Context, host string, localPaths []string, remoteDir string) error {
	args := []string{"-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new"}
	if r.SSHKey != "" {
		args = append(args, "-i", expandHome(r.SSHKey))
	}
	for _, p := range localPaths {
		args = append(args, p)
	}
	args = append(args, fmt.Sprintf("%s@%s:%s", r.SSHUser, host, remoteDir))
	cmd := exec.CommandContext(ctx, "scp", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp to %s: %w\n%s", host, err, out)
	}
	return nil
}

func (r Runner) RunLocal(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

func HTTPGet(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "curl", "-sf", url).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
