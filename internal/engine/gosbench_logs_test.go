package engine

import "testing"

func TestParseGosbenchLiveLine(t *testing.T) {
	ops, mbps, ok := parseGosbenchLiveLine("ops/s: 1250.5")
	if !ok || ops != 1250.5 {
		t.Fatalf("ops=%v ok=%v", ops, ok)
	}
	_, mbps, ok = parseGosbenchLiveLine("bandwidth: 48.2 MB/s")
	if !ok || mbps != 48.2 {
		t.Fatalf("mbps=%v ok=%v", mbps, ok)
	}
}

func TestGosbenchMockIntervals(t *testing.T) {
	iv := gosbenchMockIntervals(3, 2)
	if len(iv) != 3 || iv[0].IOPS != 500 {
		t.Fatalf("%+v", iv)
	}
}
