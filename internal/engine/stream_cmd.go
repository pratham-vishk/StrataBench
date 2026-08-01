package engine

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

type streamLineFn func(ctx context.Context, r io.Reader, onInterval func(schema.IntervalSample), acc *bytes.Buffer)

func runStreamedCommand(ctx context.Context, cmd *exec.Cmd, onInterval func(schema.IntervalSample), scan streamLineFn) ([]byte, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	tailCtx, cancel := context.WithCancel(ctx)
	var outBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		scan(tailCtx, stdout, onInterval, &outBuf)
	}()
	go func() {
		defer wg.Done()
		scan(tailCtx, stderr, onInterval, &outBuf)
	}()
	waitErr := cmd.Wait()
	cancel()
	wg.Wait()
	return outBuf.Bytes(), waitErr
}

func writeCommandOutput(path string, out []byte) {
	_ = os.WriteFile(path, out, 0o644)
}
