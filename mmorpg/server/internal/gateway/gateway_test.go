package gateway

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
	"github.com/assienindylan/terence-/mmorpg/server/internal/zone"
)

// fakeAuth valide un unique token « bon » vers un compte fixe.
type fakeAuth struct{ good string }

func (f fakeAuth) Authenticate(_ context.Context, token string) (int64, error) {
	if token == f.good {
		return 42, nil
	}
	return 0, errors.New("invalid")
}

// fakeCharacters renvoie un personnage déterministe pour le compte.
type fakeCharacters struct{}

func (fakeCharacters) GetOrCreateForAccount(_ context.Context, accountID int64) (domain.Character, error) {
	return domain.Character{
		ID: "char-42", AccountID: accountID, Name: "Testeur",
		FactionID: 1, Level: 1, HP: 100, MaxHP: 100, ZoneID: 7, X: 0, Y: 0,
	}, nil
}
func (fakeCharacters) TouchLastPlayed(context.Context, string) error { return nil }

func newTestServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	world := zone.NewManager(12)
	t.Cleanup(world.Close)
	gw := New(fakeAuth{good: "valid-token"}, fakeCharacters{}, world, log)
	srv := httptest.NewServer(gw)
	t.Cleanup(srv.Close)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	return srv, wsURL
}

func readEnvelope(t *testing.T, ws *websocket.Conn) protocol.Envelope {
	t.Helper()
	_ = ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("lecture message: %v", err)
	}
	env, err := protocol.Decode(raw)
	if err != nil {
		t.Fatalf("décodage enveloppe: %v", err)
	}
	return env
}

func TestHandshakeRejectsMissingToken(t *testing.T) {
	_, wsURL := newTestServer(t)
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("connexion sans token: attendu un échec")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("attendu 401, obtenu %v", resp)
	}
}

func TestHandshakeRejectsInvalidToken(t *testing.T) {
	_, wsURL := newTestServer(t)
	_, resp, err := websocket.DefaultDialer.Dial(wsURL+"?token=wrong", nil)
	if err == nil {
		t.Fatal("connexion avec mauvais token: attendu un échec")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("attendu 401, obtenu %v", resp)
	}
}

// readHandshake consomme les deux messages d'entrée en jeu (auth.ok puis
// zone.snapshot) et retourne le snapshot décodé.
func readHandshake(t *testing.T, ws *websocket.Conn) zone.SnapshotData {
	t.Helper()
	if env := readEnvelope(t, ws); env.Type != protocol.TypeAuthOK {
		t.Fatalf("1er message: attendu %q, obtenu %q", protocol.TypeAuthOK, env.Type)
	}
	env := readEnvelope(t, ws)
	if env.Type != protocol.TypeZoneSnapshot {
		t.Fatalf("2e message: attendu %q, obtenu %q", protocol.TypeZoneSnapshot, env.Type)
	}
	var snap zone.SnapshotData
	if err := env.DecodeData(&snap); err != nil {
		t.Fatalf("décodage zone.snapshot: %v", err)
	}
	return snap
}

func TestHandshakeAcceptsValidTokenViaQuery(t *testing.T) {
	_, wsURL := newTestServer(t)
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL+"?token=valid-token", nil)
	if err != nil {
		t.Fatalf("connexion avec token valide: %v (resp=%v)", err, resp)
	}
	defer ws.Close()

	// Premier message : auth.ok avec l'identifiant de compte.
	env := readEnvelope(t, ws)
	if env.Type != protocol.TypeAuthOK {
		t.Fatalf("premier message: attendu %q, obtenu %q", protocol.TypeAuthOK, env.Type)
	}
	var payload struct {
		AccountID int64 `json:"account_id"`
	}
	if err := env.DecodeData(&payload); err != nil {
		t.Fatalf("décodage auth.ok: %v", err)
	}
	if payload.AccountID != 42 {
		t.Fatalf("account_id: attendu 42, obtenu %d", payload.AccountID)
	}
}

func TestHandshakeAcceptsBearerHeader(t *testing.T) {
	_, wsURL := newTestServer(t)
	hdr := http.Header{"Authorization": []string{"Bearer valid-token"}}
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, hdr)
	if err != nil {
		t.Fatalf("connexion via en-tête Bearer: %v", err)
	}
	defer ws.Close()
	if env := readEnvelope(t, ws); env.Type != protocol.TypeAuthOK {
		t.Fatalf("attendu auth.ok, obtenu %q", env.Type)
	}
}

// TestZoneSnapshotOnEntry vérifie qu'après le handshake le joueur reçoit un
// zone.snapshot le contenant lui-même (T3).
func TestZoneSnapshotOnEntry(t *testing.T) {
	_, wsURL := newTestServer(t)
	ws, _, err := websocket.DefaultDialer.Dial(wsURL+"?token=valid-token", nil)
	if err != nil {
		t.Fatalf("connexion: %v", err)
	}
	defer ws.Close()

	snap := readHandshake(t, ws)
	if snap.ZoneID != 7 {
		t.Fatalf("zone_id: attendu 7, obtenu %d", snap.ZoneID)
	}
	if snap.Self != "char-42" {
		t.Fatalf("self: attendu char-42, obtenu %q", snap.Self)
	}
	if len(snap.Entities) != 1 || snap.Entities[0].CharacterID != "char-42" {
		t.Fatalf("entities: attendu [char-42], obtenu %+v", snap.Entities)
	}
	if snap.Entities[0].Name != "Testeur" || snap.Entities[0].MaxHP != 100 {
		t.Fatalf("état d'entité incorrect: %+v", snap.Entities[0])
	}
}

func TestEcho(t *testing.T) {
	_, wsURL := newTestServer(t)
	ws, _, err := websocket.DefaultDialer.Dial(wsURL+"?token=valid-token", nil)
	if err != nil {
		t.Fatalf("connexion: %v", err)
	}
	defer ws.Close()
	readHandshake(t, ws) // consomme auth.ok + zone.snapshot

	// Un message quelconque doit revenir en écho, avec le même seq et la même data.
	msg, _ := protocol.Encode("chat.say", 7, map[string]string{"body": "bonjour"})
	if err := ws.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatalf("écriture: %v", err)
	}

	env := readEnvelope(t, ws)
	if env.Type != protocol.TypeEcho {
		t.Fatalf("attendu %q, obtenu %q", protocol.TypeEcho, env.Type)
	}
	if env.Seq != 7 {
		t.Fatalf("seq: attendu 7, obtenu %d", env.Seq)
	}
	var payload struct {
		Body string `json:"body"`
	}
	if err := env.DecodeData(&payload); err != nil || payload.Body != "bonjour" {
		t.Fatalf("data d'écho incorrecte: body=%q err=%v", payload.Body, err)
	}
}

func TestPingPong(t *testing.T) {
	_, wsURL := newTestServer(t)
	ws, _, err := websocket.DefaultDialer.Dial(wsURL+"?token=valid-token", nil)
	if err != nil {
		t.Fatalf("connexion: %v", err)
	}
	defer ws.Close()
	readHandshake(t, ws) // auth.ok + zone.snapshot

	ping, _ := protocol.Encode(protocol.TypePing, 3, nil)
	if err := ws.WriteMessage(websocket.TextMessage, ping); err != nil {
		t.Fatalf("écriture ping: %v", err)
	}
	env := readEnvelope(t, ws)
	if env.Type != protocol.TypePong || env.Seq != 3 {
		t.Fatalf("attendu pong seq=3, obtenu type=%q seq=%d", env.Type, env.Seq)
	}
}

func TestInvalidMessageGetsError(t *testing.T) {
	_, wsURL := newTestServer(t)
	ws, _, err := websocket.DefaultDialer.Dial(wsURL+"?token=valid-token", nil)
	if err != nil {
		t.Fatalf("connexion: %v", err)
	}
	defer ws.Close()
	readHandshake(t, ws) // auth.ok + zone.snapshot

	if err := ws.WriteMessage(websocket.TextMessage, []byte("pas du json")); err != nil {
		t.Fatalf("écriture: %v", err)
	}
	env := readEnvelope(t, ws)
	if env.Type != protocol.TypeError {
		t.Fatalf("attendu %q, obtenu %q", protocol.TypeError, env.Type)
	}
}
