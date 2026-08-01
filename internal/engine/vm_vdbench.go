package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func runVMVdbench(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	if _, err := exec.LookPath("ssh"); err != nil {
		return nil, nil, fmt.Errorf("ssh not found in PATH (required for vm-afa profiles)")
	}
	if _, err := exec.LookPath("scp"); err != nil {
		return nil, nil, fmt.Errorf("scp not found in PATH (required for vm-afa profiles)")
	}

	sshHost, devices := parseVMTarget(in.Target, in.Profile.ParamString("guest_devices", ""))
	if devices == "" {
		devices = "/dev/sdb,/dev/sdc,/dev/sdd"
	}

	vmIn := in
	vmIn.Target = devices

	runner := &VdbenchRunner{}
	parmPath, _, err := runner.writeParmfile(vmIn)
	if err != nil {
		return nil, nil, err
	}
	defer os.Remove(parmPath)

	remoteParm := "/tmp/stratabench-vdbench.parm"
	remoteOut := "/tmp/stratabench-vdbench-out"

	scp := exec.CommandContext(ctx, "scp", parmPath, sshHost+":"+remoteParm)
	if out, err := scp.CombinedOutput(); err != nil {
		return nil, nil, fmt.Errorf("scp vdbench parmfile to guest failed: %w\n%s", err, string(out))
	}

	sshCmd := fmt.Sprintf("mkdir -p %s && vdbench -f %s -o %s", remoteOut, remoteParm, remoteOut)
	cmd := exec.CommandContext(ctx, "ssh", sshHost, sshCmd)
	out, err := cmd.CombinedOutput()
	logPath := filepath.Join(in.WorkDir, "vm-vdbench-output.txt")
	_ = os.WriteFile(logPath, out, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("guest vdbench failed: %w\n%s", err, string(out))
	}

	res, parseErr := parseVdbenchOutput(string(out), "")
	if parseErr != nil {
		return nil, nil, parseErr
	}
	return res, &schema.RawEngineOutput{Path: logPath, Format: "vdbench-text"}, nil
}
