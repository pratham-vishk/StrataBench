package remote_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pratham-vishk/stratabench/internal/agentapi"
	"github.com/pratham-vishk/stratabench/internal/agentauth"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/remote"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func TestClientHealthAndAuth(t *testing.T) {
	t.Setenv("STRATABENCH_AGENT_TOKEN", "test-secret")
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		if !agentauth.Authorized(r, "test-secret") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(agentapi.HealthResponse{Status: "ok"})
	})
	srv := httptest.NewServer(agentauth.Middleware(mux))
	defer srv.Close()

	client := remote.NewClient(srv.URL[7:]) // strip http://
	_, err := client.Health(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestClientRejectsBadToken(t *testing.T) {
	t.Setenv("STRATABENCH_AGENT_TOKEN", "good")
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		if !agentauth.Authorized(r, "good") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(agentauth.Middleware(mux))
	defer srv.Close()

	t.Setenv("STRATABENCH_AGENT_TOKEN", "bad")
	client := remote.NewClient(srv.URL[7:])
	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected auth error")
	}
}

func TestClientRun(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/run", func(w http.ResponseWriter, r *http.Request) {
		var req agentapi.RunRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		run := &schema.RunResult{RunID: "x", Profile: "p", Status: "completed", Results: schema.Results{IOPS: 1}}
		_ = json.NewEncoder(w).Encode(agentapi.RunResponse{OK: true, Run: run})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := &profile.Profile{Name: "nvme-random-oltp", Engine: "mock"}
	client := remote.NewClient(srv.URL[7:])
	run, err := client.Run(context.Background(), p, "/dev/null", true, true, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if run.RunID != "x" {
		t.Fatalf("id=%s", run.RunID)
	}
}
