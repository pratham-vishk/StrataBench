package lab

import "testing"

func TestValidationMatrix(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Clients = []Node{{Host: "10.0.1.1", Port: 7777}}
	cfg.Servers = []Node{{Host: "10.0.1.10", Port: 9000}}
	items := ValidationMatrix(cfg)
	if len(items) < 8 {
		t.Fatalf("expected validation matrix rows, got %d", len(items))
	}
	foundNative := false
	for _, it := range items {
		if it.Profile == "block-native-io_uring" {
			foundNative = true
			if it.Engine != "stratabench" {
				t.Fatalf("engine=%s", it.Engine)
			}
		}
	}
	if !foundNative {
		t.Fatal("missing block-native-io_uring in matrix")
	}
}

func TestBlockTargetDefault(t *testing.T) {
	cfg := Config{}
	if cfg.BlockTarget() != "/dev/nvme0n1" {
		t.Fatalf("default block target=%s", cfg.BlockTarget())
	}
}
