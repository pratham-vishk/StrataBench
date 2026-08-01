package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func runVMElbencho(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	if _, err := exec.LookPath("ssh"); err != nil {
		return nil, nil, fmt.Errorf("ssh not found in PATH (required for vm-file profiles)")
	}
	if _, err := exec.LookPath("scp"); err != nil {
		return nil, nil, fmt.Errorf("scp not found in PATH (required for vm-file profiles)")
	}

	sshHost, mountPath := parseVMTarget(in.Target, in.Profile.ParamString("guest_mount", "/mnt/data"))
	if mountPath == "" {
		mountPath = "/mnt/data"
	}

	threads := in.Profile.ParamInt("threads", 4)
	duration := in.Profile.ParamInt("duration_sec", 60)
	bs := in.Profile.ParamString("block_size", "4k")
	pattern := in.Profile.ParamString("pattern", "randread")

	args := []string{
		"-t", strconv.Itoa(threads),
		"-b", bs,
		"--timelimit", strconv.Itoa(duration),
	}
	switch pattern {
	case "randread", "read":
		args = append(args, "-r")
	case "randwrite", "write":
		args = append(args, "-w")
	default:
		args = append(args, "-r", "-w")
	}
	if in.Profile.ParamBool("rand", true) {
		args = append(args, "--rand")
	}
	if in.OnInterval != nil {
		args = append(args, "--livecsv", "stdout", "--liveint", "1000")
	}
	args = append(args, mountPath)

	remoteScript := "/tmp/stratabench-elbencho.sh"
	script := "#!/bin/bash\nset -e\nelbencho " + strings.Join(args, " ") + "\n"
	localScript := filepath.Join(in.WorkDir, "elbencho-remote.sh")
	if err := os.WriteFile(localScript, []byte(script), 0o755); err != nil {
		return nil, nil, err
	}

	scp := exec.CommandContext(ctx, "scp", localScript, sshHost+":"+remoteScript)
	if out, err := scp.CombinedOutput(); err != nil {
		return nil, nil, fmt.Errorf("scp elbencho script to guest failed: %w\n%s", err, string(out))
	}

	cmd := exec.CommandContext(ctx, "ssh", sshHost, "bash "+remoteScript)
	logPath := filepath.Join(in.WorkDir, "vm-elbencho-output.txt")

	var out []byte
	var err error
	if in.OnInterval != nil {
		out, err = runStreamedCommand(ctx, cmd, in.OnInterval, scanElbenchoStream)
		writeCommandOutput(logPath, out)
		if err != nil {
			return nil, nil, fmt.Errorf("guest elbencho failed: %w\n%s", err, string(out))
		}
	} else {
		out, err = cmd.CombinedOutput()
		writeCommandOutput(logPath, out)
		if err != nil {
			return nil, nil, fmt.Errorf("guest elbencho failed: %w\n%s", err, string(out))
		}
	}

	res, parseErr := parseElbenchoOutput(string(out))
	if parseErr != nil {
		return nil, nil, parseErr
	}
	return res, &schema.RawEngineOutput{Path: logPath, Format: "elbencho-text"}, nil
}
