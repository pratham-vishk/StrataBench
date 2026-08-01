package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFioLogFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stratabench_iops.1.log")
	content := `5000, 120000, 0, 4096
10000, 118500, 0, 4096
15000, 121200, 0, 4096
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	m := parseFioLogFile(path)
	if len(m) != 3 || m[1] != 120000 {
		t.Fatalf("%v", m)
	}
}

func TestParseFioLogIntervals(t *testing.T) {
	dir := t.TempDir()
	iops := `5000, 100000, 0, 4096
10000, 110000, 0, 4096
`
	bw := `5000, 524288000, 0, 4096
10000, 576716800, 0, 4096
`
	_ = os.WriteFile(filepath.Join(dir, "stratabench_iops.1.log"), []byte(iops), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "stratabench_bw.1.log"), []byte(bw), 0o644)

	iv := parseFioLogIntervals(dir, "stratabench")
	if len(iv) != 2 {
		t.Fatalf("got %d intervals", len(iv))
	}
	if iv[0].IOPS != 100000 {
		t.Fatalf("iops=%v", iv[0].IOPS)
	}
	if iv[0].ThroughputMBps < 400 {
		t.Fatalf("mbps=%v", iv[0].ThroughputMBps)
	}
}
