package discovery_test

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/discovery"
)

func TestParseSMARTTemperature(t *testing.T) {
	json := `{"temperature":{"current":42}}`
	if v := discovery.ParseSMARTTemperature(json); v != 42 {
		t.Fatalf("temp=%d", v)
	}
}
