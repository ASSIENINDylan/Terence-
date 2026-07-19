package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/assienindylan/terence-/mmorpg/server/internal/auth"
	"github.com/assienindylan/terence-/mmorpg/server/internal/store"
)

// credentials est le corps attendu pour register et login.
type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleRegister crée un compte. 201 en cas de succès, sans jamais renvoyer le
// hash du mot de passe.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var body credentials
	if !decodeJSON(w, r, &body) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	acc, err := s.auth.Register(ctx, body.Email, body.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"account_id": acc.ID,
		"email":      acc.Email,
	})
}

// handleLogin vérifie les identifiants et renvoie un token de session.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body credentials
	if !decodeJSON(w, r, &body) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	token, acc, err := s.auth.Login(ctx, body.Email, body.Password)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"token_type": "Bearer",
		"expires_in": int(s.auth.SessionTTL().Seconds()),
		"account_id": acc.ID,
	})
}

// handleLogout révoque le token présenté. Idempotent : toujours 204.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.auth.Logout(ctx, token); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleMe valide le token de session et renvoie l'identité associée. Sert de
// vérification de session réutilisée par le handshake WebSocket (T2).
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	accountID, err := s.auth.Authenticate(ctx, token)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"account_id": accountID})
}

// bearerToken extrait le token de l'en-tête « Authorization: Bearer <token> ».
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

// decodeJSON lit le corps JSON de la requête. En cas d'échec, écrit une réponse
// 400 et retourne false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)) // borne à 1 MiB
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "corps JSON invalide")
		return false
	}
	return true
}

// writeAuthError traduit les erreurs métier d'authentification en réponses HTTP.
// Les identifiants invalides et token invalides ne divulguent aucun détail.
func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email_taken", "cet email est déjà utilisé")
	case errors.Is(err, auth.ErrInvalidEmail):
		writeError(w, http.StatusBadRequest, "invalid_email", "email invalide")
	case errors.Is(err, auth.ErrWeakPassword):
		writeError(w, http.StatusBadRequest, "weak_password", "mot de passe trop court (8 caractères minimum)")
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "email ou mot de passe incorrect")
	case errors.Is(err, auth.ErrAccountNotActive):
		writeError(w, http.StatusForbidden, "account_not_active", "compte suspendu ou en attente")
	case errors.Is(err, auth.ErrInvalidToken):
		writeError(w, http.StatusUnauthorized, "invalid_token", "session invalide ou expirée")
	case errors.Is(err, store.ErrAccountNotFound):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "email ou mot de passe incorrect")
	default:
		writeError(w, http.StatusInternalServerError, "internal", "erreur interne")
	}
}

// writeError écrit une réponse d'erreur JSON structurée et stable pour le client.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error":   code,
		"message": message,
	})
}
