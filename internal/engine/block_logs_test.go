package engine

import (
	"testing"
)

func TestParseElbenchoLiveLine(t *testing.T) {
	iops, mbps, ok := parseElbenchoLiveLine("IOPS: 12500.5")
	if !ok || iops != 12500.5 {
		t.Fatalf("iops=%v ok=%v", iops, ok)
	}
	_, mbps, ok = parseElbenchoLiveLine("48.2 MiB/s")
	if !ok || mbps != 48.2 {
		t.Fatalf("mbps=%v ok=%v", mbps, ok)
	}
}

func TestParseElbenchoCSVLine(t *testing.T) {
	header := "Phase,MixType,IOPS,MiB/s,Latency"
	sample, hdr, ok := parseElbenchoCSVLine(header, nil)
	if !ok || len(hdr) != 5 {
		t.Fatalf("header parse: sample=%+v hdr=%v ok=%v", sample, hdr, ok)
	}
	sample, _, ok = parseElbenchoCSVLine("READ,read,5000,19.5,1.2", hdr)
	if !ok || sample.IOPS != 5000 || sample.ThroughputMBps != 19.5 {
		t.Fatalf("%+v ok=%v", sample, ok)
	}
}

func TestParseSPDKLiveLine(t *testing.T) {
	iops, mbps, ok := parseSPDKLiveLine("  IOPS      : 1234567.89")
	if !ok || iops != 1234567.89 {
		t.Fatalf("iops=%v ok=%v", iops, ok)
	}
	_, mbps, ok = parseSPDKLiveLine("  MiB/s     : 4812.34")
	if !ok || mbps != 4812.34 {
		t.Fatalf("mbps=%v ok=%v", mbps, ok)
	}
}

func TestParseVdbenchLiveLine(t *testing.T) {
	sample, ok := parseVdbenchLiveLine("08:20:15.001 interval        1,      9927,  1.22")
	if !ok || sample.Seq != 1 || sample.IOPS != 9927 {
		t.Fatalf("%+v ok=%v", sample, ok)
	}
	sample, ok = parseVdbenchLiveLine("rate  12345.6  io/s")
	if !ok || sample.IOPS != 12345.6 {
		t.Fatalf("%+v ok=%v", sample, ok)
	}
}

func TestParseVdbenchLiveLineInterval(t *testing.T) {
	_, ok := parseVdbenchLiveLine("no data here")
	if ok {
		t.Fatal("expected no match")
	}
	sample, ok := parseVdbenchLiveLine("interval   3, 50,000.5")
	if !ok || sample.IOPS != 50000.5 {
		t.Fatalf("%+v ok=%v", sample, ok)
	}
}
