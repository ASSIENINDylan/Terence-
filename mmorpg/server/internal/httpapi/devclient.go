package httpapi

import (
	_ "embed"
	"net/http"
)

// devClientHTML est le client de jeu jouable servi à la racine : création de
// compte, entrée en jeu, déplacement au clavier, rendu de la zone (joueurs,
// mobs, butin, portails), combat au tour par tour et inventaire. Client v0.1 du
// prototype, embarqué dans le binaire (aucune dépendance externe).
//
//go:embed devclient.html
var devClientHTML []byte

// threeJS est la bibliothèque de rendu 3D (three.js, build UMD), embarquée dans
// le binaire et servie sous /libs/ : le client 3D fonctionne sans internet.
//
//go:embed three.min.js
var threeJS []byte

// handleThreeJS sert la bibliothèque 3D du client.
func (s *Server) handleThreeJS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(threeJS)
}

// handleDevClient sert le client. Servi par le serveur lui-même, il partage son
// origine : ni CORS ni préflight à gérer.
func (s *Server) handleDevClient(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(devClientHTML)
}
