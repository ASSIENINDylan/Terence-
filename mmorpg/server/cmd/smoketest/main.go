// Commande smoketest : vérifie de bout en bout un serveur MMORPG déjà démarré,
// sans navigateur. Elle rejoue le parcours T0→T3 et affiche un rapport ✓/✗.
//
// Usage :
//
//	go run ./cmd/smoketest                 # cible http://localhost:8080
//	go run ./cmd/smoketest http://host:8080
//
// Elle NE démarre PAS le serveur : lance-le d'abord (docker compose up, ou
// go run ./cmd/server avec PostgreSQL + Redis joignables).
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
)

func main() {
	base := "http://localhost:8080"
	if len(os.Args) > 1 {
		base = strings.TrimRight(os.Args[1], "/")
	}
	fmt.Printf("Cible : %s\n\n", base)

	r := &runner{base: base, http: &http.Client{Timeout: 10 * time.Second}}

	r.step("Le serveur est prêt (GET /readyz)", r.checkReady)
	r.step("Créer un compte (POST /auth/register)", r.register)
	r.step("Se connecter (POST /auth/login)", r.login)
	r.step("Vérifier la session (GET /auth/me)", r.me)
	r.step("Entrer en jeu en WebSocket + recevoir auth.ok et zone.snapshot", r.enterGame)
	r.step("Ping applicatif → pong", r.pingPong)
	r.step("Écho d'un message", r.echo)

	fmt.Println()
	if r.failed > 0 {
		fmt.Printf("❌ %d/%d étapes en échec.\n", r.failed, r.total)
		os.Exit(1)
	}
	fmt.Printf("✅ Tout fonctionne : %d/%d étapes OK.\n", r.total, r.total)
}

type runner struct {
	base   string
	http   *http.Client
	total  int
	failed int

	// état partagé entre les étapes
	email string
	pass  string
	token string
	accID float64
	ws    *websocket.Conn
}

func (r *runner) step(name string, fn func() error) {
	r.total++
	if err := fn(); err != nil {
		r.failed++
		fmt.Printf("  ✗ %s\n      → %v\n", name, err)
		return
	}
	fmt.Printf("  ✓ %s\n", name)
}

func (r *runner) checkReady() error {
	resp, err := r.http.Get(r.base + "/readyz")
	if err != nil {
		return fmt.Errorf("serveur injoignable (est-il démarré ? port correct ?) : %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("readyz a répondu %d : %s", resp.StatusCode, body)
	}
	return nil
}

func (r *runner) register() error {
	r.email = fmt.Sprintf("smoke+%d@test.local", time.Now().UnixNano())
	r.pass = "passphrase-smoke"
	var out map[string]any
	code, err := r.postJSON("/auth/register", map[string]string{"email": r.email, "password": r.pass}, &out)
	if err != nil {
		return err
	}
	if code != 201 {
		return fmt.Errorf("attendu 201, obtenu %d (%v)", code, out)
	}
	return nil
}

func (r *runner) login() error {
	var out map[string]any
	code, err := r.postJSON("/auth/login", map[string]string{"email": r.email, "password": r.pass}, &out)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("attendu 200, obtenu %d (%v)", code, out)
	}
	tok, _ := out["token"].(string)
	if tok == "" {
		return fmt.Errorf("aucun token dans la réponse : %v", out)
	}
	r.token = tok
	r.accID, _ = out["account_id"].(float64)
	return nil
}

func (r *runner) me() error {
	req, _ := http.NewRequest("GET", r.base+"/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+r.token)
	resp, err := r.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("attendu 200, obtenu %d", resp.StatusCode)
	}
	return nil
}

func (r *runner) enterGame() error {
	wsURL := "ws" + strings.TrimPrefix(r.base, "http") + "/ws?token=" + r.token
	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("handshake WebSocket refusé (HTTP %d) : %w", resp.StatusCode, err)
		}
		return fmt.Errorf("connexion WebSocket impossible : %w", err)
	}
	r.ws = ws

	// 1er message attendu : auth.ok
	env, err := r.readEnvelope()
	if err != nil {
		return err
	}
	if env.Type != protocol.TypeAuthOK {
		return fmt.Errorf("1er message : attendu %q, obtenu %q", protocol.TypeAuthOK, env.Type)
	}

	// 2e message attendu : zone.snapshot
	env, err = r.readEnvelope()
	if err != nil {
		return err
	}
	if env.Type != protocol.TypeZoneSnapshot {
		return fmt.Errorf("2e message : attendu %q, obtenu %q", protocol.TypeZoneSnapshot, env.Type)
	}
	var snap struct {
		ZoneID   int    `json:"zone_id"`
		Self     string `json:"self"`
		Entities []struct {
			Name string `json:"name"`
		} `json:"entities"`
	}
	if err := env.DecodeData(&snap); err != nil {
		return err
	}
	if snap.Self == "" || len(snap.Entities) == 0 {
		return fmt.Errorf("snapshot incohérent : %+v", snap)
	}
	fmt.Printf("      (zone %d, personnage « %s », %d entité(s) présente(s))\n",
		snap.ZoneID, snap.Entities[0].Name, len(snap.Entities))
	return nil
}

func (r *runner) pingPong() error {
	if err := r.sendEnvelope(protocol.TypePing, 111, nil); err != nil {
		return err
	}
	env, err := r.readEnvelope()
	if err != nil {
		return err
	}
	if env.Type != protocol.TypePong || env.Seq != 111 {
		return fmt.Errorf("attendu pong seq=111, obtenu type=%q seq=%d", env.Type, env.Seq)
	}
	return nil
}

func (r *runner) echo() error {
	if err := r.sendEnvelope("chat.say", 222, map[string]string{"body": "coucou"}); err != nil {
		return err
	}
	env, err := r.readEnvelope()
	if err != nil {
		return err
	}
	if env.Type != protocol.TypeEcho || env.Seq != 222 {
		return fmt.Errorf("attendu echo seq=222, obtenu type=%q seq=%d", env.Type, env.Seq)
	}
	_ = r.ws.Close()
	return nil
}

// ── helpers ─────────────────────────────────────────────────────────────────

func (r *runner) postJSON(path string, in, out any) (int, error) {
	b, _ := json.Marshal(in)
	resp, err := r.http.Post(r.base+path, "application/json", bytes.NewReader(b))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if len(body) > 0 {
		_ = json.Unmarshal(body, out)
	}
	return resp.StatusCode, nil
}

func (r *runner) readEnvelope() (protocol.Envelope, error) {
	_ = r.ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, err := r.ws.ReadMessage()
	if err != nil {
		return protocol.Envelope{}, fmt.Errorf("lecture WebSocket : %w", err)
	}
	return protocol.Decode(raw)
}

func (r *runner) sendEnvelope(t string, seq uint64, data any) error {
	raw, err := protocol.Encode(t, seq, data)
	if err != nil {
		return err
	}
	return r.ws.WriteMessage(websocket.TextMessage, raw)
}
