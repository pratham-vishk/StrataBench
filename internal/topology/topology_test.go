package topology

import "testing"

func TestInferMode(t *testing.T) {
	if inferMode(0, 1) != ModeSingle {
		t.Fatal("0 clients 1 target")
	}
	if inferMode(1, 1) != ModeSingle {
		t.Fatal("1 client 1 target")
	}
	if inferMode(0, 3) != ModeSweep {
		t.Fatal("0 clients N targets")
	}
	if inferMode(3, 1) != ModePool {
		t.Fatal("N clients 1 target")
	}
	if inferMode(3, 2) != ModeShard {
		t.Fatal("N clients M targets")
	}
}

func TestPlanPool(t *testing.T) {
	p, err := Build(ModePool, []string{"10.0.0.1:7777", "10.0.0.2:7777"}, []string{"/dev/nvme0n1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Assignments) != 2 {
		t.Fatalf("got %d assignments", len(p.Assignments))
	}
	for _, a := range p.Assignments {
		if a.Target != "/dev/nvme0n1" {
			t.Fatalf("target: %s", a.Target)
		}
	}
}

func TestPlanSweep(t *testing.T) {
	p, err := Build(ModeSweep, nil, []string{"10.0.1.10:9000", "10.0.1.11:9000"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Assignments) != 2 {
		t.Fatalf("got %d", len(p.Assignments))
	}
}

func TestPlanShard(t *testing.T) {
	p, err := Build(ModeShard, []string{"c1", "c2", "c3"}, []string{"s1", "s2"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"s1", "s2", "s1"}
	for i, a := range p.Assignments {
		if a.Target != want[i] {
			t.Fatalf("[%d] target %s want %s", i, a.Target, want[i])
		}
	}
}

func TestPlanMatrix(t *testing.T) {
	p, err := Build(ModeMatrix, []string{"c1", "c2"}, []string{"s1", "s2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Assignments) != 4 {
		t.Fatalf("got %d", len(p.Assignments))
	}
}

func TestPlanAutoMultiClientOneServer(t *testing.T) {
	p, err := Build(ModeAuto, []string{"c1", "c2"}, []string{"/dev/nvme0n1"})
	if err != nil || p.Mode != ModePool {
		t.Fatalf("mode=%s err=%v", p.Mode, err)
	}
}

func TestPlanAutoOneClientMultiServer(t *testing.T) {
	p, err := Build(ModeAuto, nil, []string{"s1", "s2", "s3"})
	if err != nil || p.Mode != ModeSweep {
		t.Fatalf("mode=%s err=%v", p.Mode, err)
	}
}
