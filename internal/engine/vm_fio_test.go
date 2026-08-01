package engine

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/profile"
)

func TestParseVMTarget(t *testing.T) {
	host, dev := parseVMTarget("root@10.0.1.5:/dev/vdb", "")
	if host != "root@10.0.1.5" || dev != "/dev/vdb" {
		t.Fatalf("got %q %q", host, dev)
	}
	host, dev = parseVMTarget("root@10.0.1.5", "/dev/sdb")
	if host != "root@10.0.1.5" || dev != "/dev/sdb" {
		t.Fatalf("got %q %q", host, dev)
	}
}

func TestIsVMSSH(t *testing.T) {
	in := RunInput{
		Profile: &profile.Profile{Layer: "vm-block"},
		Target:  "user@vm1",
	}
	if !isVMSSH(in) {
		t.Fatal("expected vm ssh")
	}
}
