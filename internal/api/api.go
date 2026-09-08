// Package api exposes the HTTP interface.
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dovlahaha/deploytrack/internal/metrics"
	"github.com/dovlahaha/deploytrack/internal/store"
)

// Server holds handler dependencies.
type Server struct{ store *store.Store }

// New builds a Server.
func New(s *store.Store) *Server { return &Server{store: s} }

// Routes returns the configured mux.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /deployments", s.createDeployment)
	mux.HandleFunc("GET /deployments", s.listDeployments)
	mux.HandleFunc("GET /metrics", s.serviceMetrics)
	return logRequests(mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type createRequest struct {
	ServiceName string     `json:"service_name"`
	Version     string     `json:"version"`
	Environment string     `json:"environment"`
	Status      string     `json:"status"`
	DeployedAt  *time.Time `json:"deployed_at"`
}

func (s *Server) createDeployment(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}

	if req.ServiceName == "" || req.Version == "" || req.Environment == "" {
		writeError(w, http.StatusBadRequest, "service_name, version and environment are required")
		return
	}
	if req.Status == "" {
		req.Status = metrics.StatusSuccess
	}
	switch req.Status {
	case metrics.StatusSuccess, metrics.StatusFailed, metrics.StatusRolledBack:
	default:
		writeError(w, http.StatusBadRequest, "status must be success, failed or rolled_back")
		return
	}

	d := &store.Deployment{
		ServiceName: req.ServiceName,
		Version:     req.Version,
		Environment: req.Environment,
		Status:      req.Status,
	}
	if req.DeployedAt != nil {
		d.DeployedAt = *req.DeployedAt
	}

	if err := s.store.Create(d); err != nil {
		log.Printf("create deployment: %v", err)
		writeError(w, http.StatusInternalServerError, "could not record the deployment")
		return
	}
	writeJSON(w, http.StatusCreated, d)
}

func (s *Server) listDeployments(w http.ResponseWriter, r *http.Request) {
	ds, err := s.store.List(store.ListFilter{
		ServiceName: r.URL.Query().Get("service"),
		Environment: r.URL.Query().Get("environment"),
	})
	if err != nil {
		log.Printf("list deployments: %v", err)
		writeError(w, http.StatusInternalServerError, "could not read deployments")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(ds), "deployments": ds})
}

func (s *Server) serviceMetrics(w http.ResponseWriter, r *http.Request) {
	days := 30
	if raw := r.URL.Query().Get("days"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, "days must be a positive whole number")
			return
		}
		days = n
	}

	window := time.Duration(days) * 24 * time.Hour
	ds, err := s.store.List(store.ListFilter{
		ServiceName: r.URL.Query().Get("service"),
		Environment: r.URL.Query().Get("environment"),
		Since:       time.Now().UTC().Add(-window),
	})
	if err != nil {
		log.Printf("metrics query: %v", err)
		writeError(w, http.StatusInternalServerError, "could not calculate metrics")
		return
	}
	writeJSON(w, http.StatusOK, metrics.Calculate(store.Events(ds), window))
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
