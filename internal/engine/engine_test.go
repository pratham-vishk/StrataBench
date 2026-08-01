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

func TestGOSBenchRequiresServer(t *testing.T) {
	p := &profile.Profile{Name: "x", Engine: "gosbench"}
	r := ForProfile(p, false)
	_, _, err := r.Run(context.Background(), RunInput{Profile: p, WorkDir: t.TempDir(), Target: "127.0.0.1:9000"})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "gosbench") {
		t.Fatalf("err=%v", err)
	}
}

func TestForProfileStratabenchRequiresBinary(t *testing.T) {
	p := &profile.Profile{Name: "x", Engine: "stratabench"}
	r := ForProfile(p, false)
	_, _, err := r.Run(context.Background(), RunInput{Profile: p, WorkDir: t.TempDir(), Target: "/dev/null"})
	if err == nil || !strings.Contains(err.Error(), "engine") {
		t.Fatalf("err=%v", err)
	}
	r = ForProfile(p, true)
	res, _, err := r.Run(context.Background(), RunInput{Profile: p, Mock: true, WorkDir: t.TempDir()})
	if err != nil || res.IOPS <= 0 {
		t.Fatalf("mock err=%v iops=%v", err, res.IOPS)
	}
}
