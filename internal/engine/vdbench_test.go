package engine

import (
	"os"
	"strings"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/profile"
)

func TestVdbenchPatternMapping(t *testing.T) {
	rd, seek := vdbenchPattern("randread", -1)
	if rd != 100 || seek != 100 {
		t.Fatalf("randread: rd=%d seek=%d", rd, seek)
	}
	rd, seek = vdbenchPattern("randwrite", -1)
	if rd != 0 || seek != 100 {
		t.Fatalf("randwrite: rd=%d seek=%d", rd, seek)
	}
}

func TestVdbenchParmfileLUNs(t *testing.T) {
	p := &profile.Profile{
		Name:   "afa-multi-lun",
		Params: map[string]any{"pattern": "randread", "threads": 4},
	}
	in := RunInput{
		Profile: p,
		Target:  "/dev/sdb,/dev/sdc",
		WorkDir: t.TempDir(),
	}
	runner := &VdbenchRunner{}
	parmPath, _, err := runner.writeParmfile(in)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(parmPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "lun=/dev/sdb") || !strings.Contains(text, "lun=/dev/sdc") {
		t.Fatalf("missing luns in parmfile:\n%s", text)
	}
}

func TestParseVdbenchOutput(t *testing.T) {
	text := "avg  12345.6  0.250  some columns"
	res, err := parseVdbenchOutput(text, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.IOPS != 12345.6 {
		t.Fatalf("iops=%v", res.IOPS)
	}
}
