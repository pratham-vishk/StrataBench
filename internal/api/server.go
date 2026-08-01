package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/agentloop"
	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/inventory"
	"github.com/pratham-vishk/stratabench/internal/llm"
	"github.com/pratham-vishk/stratabench/internal/metrics"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/planner"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/remote"
	"github.com/pratham-vishk/stratabench/internal/runstate"
)

type Server struct {
	Svc *orchestrator.Service
}

type runRequest struct {
	Profile       string   `json:"profile"`
	Target        string   `json:"target"`
	Targets       []string `json:"targets"`
	Clients       []string `json:"clients"`
	Topology      string   `json:"topology"`
	Mock          bool     `json:"mock"`
	Async         bool     `json:"async"`
	SkipValidate  bool     `json:"skip_validate"`
	CheckHardware bool     `json:"check_hardware"`
	CacheBytes    int64    `json:"cache_bytes"`
}

type agentRequest struct {
	Intent        string `json:"intent"`
	Target        string `json:"target"`
	Mock          bool   `json:"mock"`
	UseLLM        bool   `json:"use_llm"`
	UseOllama     bool   `json:"use_ollama"`
	CheckBaseline bool   `json:"check_baseline"`
	CheckHardware bool   `json:"check_hardware"`
}

type planRequest struct {
	Intent   string `json:"intent"`
	UseLLM   bool   `json:"use_llm"`
	Provider string `json:"llm_provider"`
	Model    string `json:"model"`
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
	mux.HandleFunc("/api/v1/plan", s.handlePlan)
	mux.HandleFunc("/api/v1/agent", s.handleAgent)
	mux.HandleFunc("/api/v1/validate", s.handleValidate)
	mux.HandleFunc("/api/v1/compare", s.handleCompare)
	mux.HandleFunc("/api/v1/report/", s.handleReport)
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
		UseLLM:        req.UseLLM || req.UseOllama,
		UseOllama:     req.UseOllama,
		LLM:           llm.FromEnv(),
		CheckBaseline: req.CheckBaseline,
		CheckHardware: req.CheckHardware,
		DataDir:       paths.DataDir(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, result)
}

func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req planRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Intent == "" {
		http.Error(w, "intent required", http.StatusBadRequest)
		return
	}
	profiles, err := profile.List(paths.ProfilesDir())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cfg := llm.FromEnv()
	if req.Provider != "" {
		cfg.Provider = req.Provider
	}
	if req.Model != "" {
		cfg.Model = req.Model
	}
	res := planner.Plan(r.Context(), planner.PlanOptions{
		Intent:   req.Intent,
		Profiles: profiles,
		Hardware: discovery.Snapshot(),
		UseLLM:   req.UseLLM,
		LLM:      cfg,
	})
	writeJSON(w, res)
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
		opts := orchestrator.RunOptions{
			Profile:       p,
			Target:        req.Target,
			Targets:       req.Targets,
			Clients:       req.Clients,
			Topology:      req.Topology,
			Mock:          req.Mock,
			SkipValidate:  req.SkipValidate,
			CheckHardware: req.CheckHardware,
			CacheBytes:    req.CacheBytes,
			DataDir:       paths.DataDir(),
		}
		if req.Async {
			runID, err := s.Svc.StartAsyncRun(r.Context(), opts)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusAccepted)
			writeJSON(w, map[string]string{"run_id": runID, "status": "running"})
			return
		}
		run, err := s.Svc.Run(r.Context(), opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, run)
	case path != "" && strings.HasSuffix(path, "/stream") && r.Method == http.MethodGet:
		runID := strings.TrimSuffix(path, "/stream")
		runID = strings.Trim(runID, "/")
		s.handleRunStream(w, r, runID)
	case path != "" && strings.HasSuffix(path, "/progress") && r.Method == http.MethodGet:
		runID := strings.TrimSuffix(path, "/progress")
		runID = strings.Trim(runID, "/")
		progress, ok := runstate.Get(runID)
		if !ok {
			http.Error(w, "no active progress for run", http.StatusNotFound)
			return
		}
		writeJSON(w, progress)
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