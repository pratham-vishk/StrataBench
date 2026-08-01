package engine

import "testing"

func TestIsVMLayer(t *testing.T) {
	for _, layer := range []string{"vm-block", "vm-file", "vm-object", "vm-application"} {
		if !isVMLayer(layer) {
			t.Fatalf("expected vm layer: %s", layer)
		}
	}
	if isVMLayer("block") {
		t.Fatal("block is not vm layer")
	}
}
