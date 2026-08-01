package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/agentloop"
	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/inventory"
	"github.com/pratham-vishk/stratabench/internal/metrics"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/remote"
)

type Server struct {
	Svc *orchestrator.Service
}

type runRequest struct {
	Profile      string   `json:"profile"`
	Target       string   `json:"target"`
	Clients      []string `json:"clients"`
	Mock         bool     `json:"mock"`
	SkipValidate bool     `json:"skip_validate"`
	CacheBytes   int64    `json:"cache_bytes"`
}

type agentRequest struct {
	Intent        string `json:"intent"`
	Target        string `json:"target"`
	Mock          bool   `json:"mock"`
	UseOllama     bool   `json:"use_ollama"`
	CheckBaseline bool   `json:"check_baseline"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/profiles", s.handleProfiles)
	mux.HandleFunc("/api/v1/runs", s.handleRuns)
	mux.HandleFunc("/api/v1/inventory", s.handleInventory)
	mux.HandleFunc("/api/v1/analyze/", s.handleAnalyze)
	mux.HandleFunc("/api/v1/agent", s.handleAgent)
	mux.Handle("/metrics", metrics.Handler())
	return mux
}

func (s *Server) handleInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	recs, err := inventory.List(s.Svc.Store)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, recs)
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	runID := strings.TrimPrefix(r.URL.Path, "/api/v1/analyze/")
	if runID == "" {
		http.Error(w, "run id required", http.StatusBadRequest)
		return
	}
	run, err := s.Svc.Store.Get(runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	regression := s.Svc.CheckRegression(run)
	insights := analyst.Analyze(run, regression)
	writeJSON(w, map[string]any{
		"run_id":   runID,
		"insights": insights,
		"summary":  analyst.SummaryText(run, insights),
	})
}

func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req agentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Target == "" {
		http.Error(w, "target required", http.StatusBadRequest)
		return
	}
	result, err := agentloop.Run(r.Context(), agentloop.Options{
		Intent:        req.Intent,
		Target:        req.Target,
		Mock:          req.Mock,
		UseOllama:     req.UseOllama,
		CheckBaseline: req.CheckBaseline,
		DataDir:       paths.DataDir(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, result)
}

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	profiles, err := profile.List(paths.ProfilesDir())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, profiles)
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/runs")
	path = strings.Trim(path, "/")

	switch {
	case path == "" && r.Method == http.MethodGet:
		runs, err := s.Svc.Store.List(50)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, runs)
	case path == "" && r.Method == http.MethodPost:
		var req runRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		p, err := profile.LoadByName(paths.ProfilesDir(), req.Profile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		run, err := s.Svc.Run(r.Context(), orchestrator.RunOptions{
			Profile:      p,
			Target:       req.Target,
			Clients:      req.Clients,
			Mock:         req.Mock,
			SkipValidate: req.SkipValidate,
			CacheBytes:   req.CacheBytes,
			DataDir:      paths.DataDir(),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, run)
	case path != "" && r.Method == http.MethodGet:
		run, err := s.Svc.Store.Get(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, run)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// ParseClientsCSV helper for API consumers.
func ParseClientsCSV(csv string) []string { return remote.ParseHosts(csv) }