// Package httpapi expose l'API HTTP du serveur. En v0.1 elle porte les
// endpoints de santé (T0) et l'authentification (T1 : register, login, logout,
// me) ; l'upgrade WebSocket (T2) viendra s'y greffer.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/assienindylan/terence-/mmorpg/server/internal/auth"
)

// Pinger est une dépendance dont on peut vérifier la disponibilité.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Server assemble les handlers HTTP autour des dépendances (DB, cache, auth,
// gateway WebSocket).
type Server struct {
	db      Pinger
	cache   Pinger
	auth    *auth.Service
	gateway http.Handler
}

// New construit le serveur HTTP avec ses dépendances. gateway est le handler
// d'upgrade WebSocket (T2) ; il peut être nil (aucune route /ws n'est alors
// exposée).
func New(db, cache Pinger, authSvc *auth.Service, gateway http.Handler) *Server {
	return &Server{db: db, cache: cache, auth: authSvc, gateway: gateway}
}

// Handler retourne le routeur HTTP racine.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	// Liveness : le process répond-il ? (aucune dépendance externe testée)
	mux.HandleFunc("GET /healthz", s.handleLiveness)
	// Readiness : DB et Redis répondent-ils ? Critère de validation de T0.
	mux.HandleFunc("GET /readyz", s.handleReadiness)
	// Authentification (T1).
	mux.HandleFunc("POST /auth/register", s.handleRegister)
	mux.HandleFunc("POST /auth/login", s.handleLogin)
	mux.HandleFunc("POST /auth/logout", s.handleLogout)
	mux.HandleFunc("GET /auth/me", s.handleMe)
	// Gateway temps réel (T2) : upgrade WebSocket authentifié.
	if s.gateway != nil {
		mux.Handle("GET /ws", s.gateway)
	}
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
