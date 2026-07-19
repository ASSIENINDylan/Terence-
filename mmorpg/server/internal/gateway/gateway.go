// Package gateway porte la terminaison WebSocket du serveur (§3, §5, T2 du plan
// de build).
//
// Rôle : accepter les connexions temps réel, authentifier le handshake avec un
// token de session (émis par le package auth en T1), puis router les messages.
// En T2, le routage se limite à un écho ; le mouvement, le combat et le chat
// (T3+) viendront s'y brancher via Conn.
package gateway

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
	"github.com/assienindylan/terence-/mmorpg/server/internal/zone"
)

// Authenticator valide un token de session et renvoie le compte associé.
// auth.Service satisfait cette interface (méthode Authenticate).
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (accountID int64, err error)
}

// CharacterStore charge (ou crée) le personnage persistant d'un compte.
// store.Store satisfait cette interface.
type CharacterStore interface {
	GetOrCreateForAccount(ctx context.Context, accountID int64) (domain.Character, error)
	TouchLastPlayed(ctx context.Context, characterID string) error
}

// World place les personnages dans leur zone et en produit les snapshots.
// zone.Manager satisfait cette interface.
type World interface {
	Enter(char domain.Character, sender zone.Sender) zone.SnapshotData
	Leave(char domain.Character)
}

// Gateway gère l'upgrade HTTP→WebSocket et le cycle de vie des connexions.
type Gateway struct {
	auth       Authenticator
	characters CharacterStore
	world      World
	log        *slog.Logger
	upgrader   websocket.Upgrader
}

// New construit la gateway. La vérification d'origine est permissive en v0.1
// (dev) ; elle sera restreinte à la liste des origines autorisées en durcissement.
func New(auth Authenticator, characters CharacterStore, world World, log *slog.Logger) *Gateway {
	return &Gateway{
		auth:       auth,
		characters: characters,
		world:      world,
		log:        log,
		upgrader: websocket.Upgrader{
			HandshakeTimeout: 10 * time.Second,
			ReadBufferSize:   1024,
			WriteBufferSize:  1024,
			CheckOrigin:      func(_ *http.Request) bool { return true },
		},
	}
}

// ServeHTTP authentifie puis met à niveau la requête en connexion WebSocket.
// L'authentification a lieu AVANT l'upgrade : un token absent ou invalide reçoit
// une réponse HTTP 401 classique, sans jamais ouvrir de socket.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing_token", "token de session requis")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	accountID, err := g.auth.Authenticate(ctx, token)
	cancel()
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid_token", "session invalide ou expirée")
		return
	}

	ws, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade écrit déjà la réponse d'erreur HTTP appropriée.
		g.log.Warn("échec de l'upgrade WebSocket", "err", err)
		return
	}

	conn := newConn(accountID, ws, g.log)
	g.log.Info("connexion WebSocket établie", "account_id", accountID)

	// Handshake applicatif : on confirme l'authentification au client.
	conn.SendEnvelope(protocol.TypeAuthOK, 0, map[string]any{"account_id": accountID})

	// Entrée en jeu (T3) : charger le personnage persistant, l'entrer dans sa
	// zone, et lui envoyer l'état initial. Un échec ici ferme proprement la
	// connexion plutôt que de laisser le joueur dans un état incohérent.
	char, ok := g.enterWorld(r.Context(), conn, accountID)
	if !ok {
		_ = ws.Close()
		return
	}
	defer g.world.Leave(char)

	// run bloque jusqu'à la fermeture de la connexion (lecture/écriture).
	conn.run()
	g.log.Info("connexion WebSocket fermée", "account_id", accountID, "character_id", char.ID)
}

// enterWorld charge le personnage du compte, le place dans sa zone et lui envoie
// le zone.snapshot. Retourne false si l'entrée échoue (le personnage n'est alors
// pas dans le monde).
func (g *Gateway) enterWorld(reqCtx context.Context, conn *Conn, accountID int64) (domain.Character, bool) {
	ctx, cancel := context.WithTimeout(reqCtx, 5*time.Second)
	defer cancel()

	char, err := g.characters.GetOrCreateForAccount(ctx, accountID)
	if err != nil {
		g.log.Error("chargement du personnage échoué", "account_id", accountID, "err", err)
		conn.SendEnvelope(protocol.TypeError, 0, protocol.ErrorData{
			Code:    "character_load_failed",
			Message: "impossible de charger le personnage",
		})
		return domain.Character{}, false
	}

	// Trace la session ; non bloquant pour l'entrée en jeu.
	_ = g.characters.TouchLastPlayed(ctx, char.ID)

	snap := g.world.Enter(char, conn)
	conn.SendEnvelope(protocol.TypeZoneSnapshot, 0, snap)
	g.log.Info("entrée en zone", "character_id", char.ID, "zone_id", char.ZoneID, "présents", len(snap.Entities))
	return char, true
}

// extractToken lit le token depuis l'en-tête « Authorization: Bearer <token> »
// (clients natifs) ou, à défaut, le paramètre de requête ?token= (navigateurs,
// qui ne peuvent pas fixer d'en-tête sur un WebSocket).
func extractToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) > len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return strings.TrimSpace(r.URL.Query().Get("token"))
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + code + `","message":"` + message + `"}`))
}
