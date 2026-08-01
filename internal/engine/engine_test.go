package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/profile"
)

func TestForProfileUnknownEngineFails(t *testing.T) {
	p := &profile.Profile{Name: "custom", Engine: "nosuch"}
	r := ForProfile(p, false)
	_, _, err := r.Run(context.Background(), RunInput{Profile: p, WorkDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "nosuch") {
		t.Fatalf("err=%v", err)
	}
}

func TestGOSBenchUnsupported(t *testing.T) {
	p := &profile.Profile{Name: "x", Engine: "gosbench"}
	r := ForProfile(p, false)
	_, _, err := r.Run(context.Background(), RunInput{Profile: p, WorkDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "gosbench") {
		t.Fatalf("err=%v", err)
	}
}

func TestForProfileStratabenchRequiresMock(t *testing.T) {
	p := &profile.Profile{Name: "x", Engine: "stratabench"}
	r := ForProfile(p, false)
	_, _, err := r.Run(context.Background(), RunInput{Profile: p, WorkDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for native stratabench engine")
	}
	r = ForProfile(p, true)
	res, _, err := r.Run(context.Background(), RunInput{Profile: p, Mock: true, WorkDir: t.TempDir()})
	if err != nil || res.IOPS <= 0 {
		t.Fatalf("mock err=%v iops=%v", err, res.IOPS)
	}
}
