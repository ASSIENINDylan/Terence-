// Package httpapi expose l'API HTTP du serveur. En v0.1 (T0) elle porte les
// endpoints de santé ; le login HTTP (T1) et l'upgrade WebSocket (T2) viendront
// s'y greffer.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Pinger est une dépendance dont on peut vérifier la disponibilité.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Server assemble les handlers HTTP autour des dépendances (DB, cache).
type Server struct {
	db    Pinger
	cache Pinger
}

// New construit le serveur HTTP avec ses dépendances.
func New(db, cache Pinger) *Server {
	return &Server{db: db, cache: cache}
}

// Handler retourne le routeur HTTP racine.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	// Liveness : le process répond-il ? (aucune dépendance externe testée)
	mux.HandleFunc("GET /healthz", s.handleLiveness)
	// Readiness : DB et Redis répondent-ils ? Critère de validation de T0.
	mux.HandleFunc("GET /readyz", s.handleReadiness)
	return mux
}

type healthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

func (s *Server) handleLiveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	checks := map[string]string{
		"postgres": statusOf(s.db.Ping(ctx)),
		"redis":    statusOf(s.cache.Ping(ctx)),
	}

	status := http.StatusOK
	overall := "ok"
	for _, v := range checks {
		if v != "ok" {
			status = http.StatusServiceUnavailable
			overall = "degraded"
		}
	}
	writeJSON(w, status, healthResponse{Status: overall, Checks: checks})
}

func statusOf(err error) string {
	if err != nil {
		return "error: " + err.Error()
	}
	return "ok"
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
