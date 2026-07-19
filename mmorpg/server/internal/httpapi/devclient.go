package httpapi

import (
	_ "embed"
	"net/http"
)

// devClientHTML est une page de test servie à la racine pour exercer le serveur
// à la main depuis un navigateur (créer un compte, se connecter, entrer en jeu
// et visualiser le zone.snapshot). C'est un harnais de développement du
// prototype v0.1 — pas le client de jeu final.
//
//go:embed devclient.html
var devClientHTML []byte

// handleDevClient sert la page de test. Servie par le serveur lui-même, elle
// partage son origine : ni CORS ni préflight à gérer.
func (s *Server) handleDevClient(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(devClientHTML)
}
