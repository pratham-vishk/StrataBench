package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/pratham-vishk/stratabench/internal/agentapi"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/profile"
)

func main() {
	listen := envOr("STRATABENCH_AGENT_LISTEN", ":7777")
	dataDir := envOr("STRATABENCH_AGENT_DATA", filepath.Join(paths.DataDir(), "agent"))

	svc, err := orchestrator.NewService(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	defer svc.Close()

	host, _ := os.Hostname()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, agentapi.HealthResponse{Status: "ok", Version: agentapi.Version, Host: host})
	})
	mux.HandleFunc("/v1/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var req agentapi.RunRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var p profile.Profile
		if err := yaml.Unmarshal([]byte(req.ProfileYAML), &p); err != nil {
			writeJSON(w, agentapi.RunResponse{OK: false, Error: err.Error()})
			return
		}
		workDir := req.WorkDir
		if workDir == "" {
			workDir = filepath.Join(dataDir, "work")
		}
		run, err := svc.Run(r.Context(), orchestrator.RunOptions{
			Profile:      &p,
			Target:       req.Target,
			Mock:         req.Mock,
			SkipValidate: req.SkipValidate,
			CacheBytes:   req.CacheBytes,
			WorkDir:      workDir,
			DataDir:      dataDir,
		})
		if err != nil {
			writeJSON(w, agentapi.RunResponse{OK: false, Error: err.Error()})
			return
		}
		writeJSON(w, agentapi.RunResponse{OK: true, Run: run})
	})

	log.Printf("stratabench-agent listening on %s", listen)
	log.Fatal(http.ListenAndServe(listen, mux))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
