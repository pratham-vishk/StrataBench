package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func parseVMTarget(target string, profileDevice string) (sshHost, device string) {
	if idx := strings.LastIndex(target, ":"); idx > 0 && strings.Contains(target[:idx], "@") {
		return target[:idx], target[idx+1:]
	}
	sshHost = target
	device = profileDevice
	if device == "" {
		device = "/dev/vdb"
	}
	return sshHost, device
}

func runVMFio(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	if _, err := exec.LookPath("ssh"); err != nil {
		return nil, nil, fmt.Errorf("ssh not found in PATH (required for vm-block profiles)")
	}

	sshHost, device := parseVMTarget(in.Target, in.Profile.ParamString("guest_device", ""))
	vmIn := in
	vmIn.Target = device

	runner := &FioRunner{}
	jobPath, _, err := runner.writeJob(vmIn)
	if err != nil {
		return nil, nil, err
	}
	defer os.Remove(jobPath)

	remoteJob := "/tmp/stratabench.fio"
	scp := exec.CommandContext(ctx, "scp", jobPath, sshHost+":"+remoteJob)
	if out, err := scp.CombinedOutput(); err != nil {
		return nil, nil, fmt.Errorf("scp job to guest failed: %w\n%s", err, string(out))
	}

	sshCmd := fmt.Sprintf("fio %s --output-format=json", remoteJob)
	cmd := exec.CommandContext(ctx, "ssh", sshHost, sshCmd)
	out, err := cmd.CombinedOutput()
	logPath := filepath.Join(in.WorkDir, "vm-fio-output.json")
	_ = os.WriteFile(logPath, out, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("guest fio failed: %w\n%s", err, string(out))
	}

	res, parseErr := parseFioJSON(out)
	if parseErr != nil {
		return nil, nil, parseErr
	}
	return res, &schema.RawEngineOutput{Path: logPath, Format: "fio-json"}, nil
}
