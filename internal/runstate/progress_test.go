package runstate

import "testing"

func TestProgressLifecycle(t *testing.T) {
	id := "run-1"
	Set(Progress{RunID: id, Phase: "running", TotalAssignments: 2})
	p, ok := Get(id)
	if !ok || p.TotalAssignments != 2 {
		t.Fatalf("%+v ok=%v", p, ok)
	}
	IncrementDone(id)
	p, _ = Get(id)
	if p.CompletedAssignments != 1 {
		t.Fatalf("%+v", p)
	}
	Clear(id)
	if _, ok := Get(id); ok {
		t.Fatal("expected cleared")
	}
}
