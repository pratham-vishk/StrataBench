package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/compare"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/report"
)

type validateRequest struct {
	Profile       string `json:"profile"`
	Target        string `json:"target"`
	Mock          bool   `json:"mock"`
	CheckHardware bool   `json:"check_hardware"`
	CacheBytes    int64  `json:"cache_bytes"`
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req validateRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Profile == "" {
		http.Error(w, "profile required", http.StatusBadRequest)
		return
	}
	p, err := profile.LoadByName(paths.ProfilesDir(), req.Profile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result := s.Svc.Validate(orchestrator.RunOptions{
		Profile:       p,
		Target:        req.Target,
		Mock:          req.Mock,
		CheckHardware: req.CheckHardware,
		CacheBytes:    req.CacheBytes,
	})
	writeJSON(w, result)
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	runID := strings.TrimPrefix(r.URL.Path, "/api/v1/report/")
	runID = strings.Trim(runID, "/")
	if runID == "" {
		http.Error(w, "run id required", http.StatusBadRequest)
		return
	}
	run, err := s.Svc.Store.Get(runID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		regression := s.Svc.CheckRegression(run)
		insights := analyst.Analyze(run, regression)
		outPath := filepath.Join(paths.ReportsDir(), runID+".html")
		if err := report.WriteHTMLWithInsights(run, insights, analyst.SummaryText(run, insights), outPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if strings.Contains(r.Header.Get("Accept"), "text/html") {
			http.ServeFile(w, r, outPath)
			return
		}
		writeJSON(w, map[string]string{"run_id": runID, "html_path": outPath})
	case http.MethodPost:
		outPath := filepath.Join(paths.ReportsDir(), runID+".html")
		if err := report.WriteHTML(run, outPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"run_id": runID, "html_path": outPath})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type compareRequest struct {
	RunIDA string `json:"run_id"`
	RunIDB string `json:"run_id_b"`
}

func (s *Server) handleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req compareRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.RunIDA == "" || req.RunIDB == "" {
		http.Error(w, "run_id and run_id_b required", http.StatusBadRequest)
		return
	}
	a, err := s.Svc.Store.Get(req.RunIDA)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	b, err := s.Svc.Store.Get(req.RunIDB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{
		"run_id":   req.RunIDA,
		"run_id_b": req.RunIDB,
		"diff":     compare.Diff(a, b),
	})
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
