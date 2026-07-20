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
	r.step("Rejoindre une zone verte par le portail + y voir 3 mobs", r.transitionToGreen)
	r.step("Combat PvE : approcher un mob, l'IA joue, le vaincre, gagner de l'or", r.fightMob)
	r.step("Réapparition : le mob tué revient (la zone garde ses 3 mobs)", r.checkRespawn)
	_ = r.ws.Close()

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

	// état de jeu suivi au fil des messages (snapshot + deltas)
	selfID  string
	zoneID  int
	tier    string
	portals []portalInfo
	pos     map[string]xy
	mobs    map[string]bool
}

type xy struct{ X, Y int }

type portalInfo struct {
	LinkID int    `json:"link_id"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Kind   string `json:"kind"`
	ToZone string `json:"to_zone"`
}

type snapshotMsg struct {
	ZoneID   int    `json:"zone_id"`
	Tier     string `json:"tier"`
	Self     string `json:"self"`
	Entities []struct {
		CharacterID string `json:"character_id"`
		Name        string `json:"name"`
		X           int    `json:"x"`
		Y           int    `json:"y"`
		Mob         bool   `json:"mob"`
	} `json:"entities"`
	Portals []portalInfo `json:"portals"`
}

type deltaMsg struct {
	Joined []struct {
		CharacterID string `json:"character_id"`
		X           int    `json:"x"`
		Y           int    `json:"y"`
		Mob         bool   `json:"mob"`
	} `json:"joined"`
	Moved []struct {
		CharacterID string `json:"character_id"`
		X           int    `json:"x"`
		Y           int    `json:"y"`
	} `json:"moved"`
	Left []string `json:"left"`
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

	// Ensuite : la fiche du personnage (char.update) peut précéder le
	// zone.snapshot. On lit jusqu'au snapshot en ignorant les char.update.
	for {
		env, err = r.readEnvelope()
		if err != nil {
			return err
		}
		if env.Type == protocol.TypeCharUpdate {
			continue
		}
		if env.Type != protocol.TypeZoneSnapshot {
			return fmt.Errorf("attendu %q (ou char.update), obtenu %q", protocol.TypeZoneSnapshot, env.Type)
		}
		break
	}
	var snap snapshotMsg
	if err := env.DecodeData(&snap); err != nil {
		return err
	}
	if snap.Self == "" || len(snap.Entities) == 0 {
		return fmt.Errorf("snapshot incohérent : %+v", snap)
	}
	r.applySnapshot(snap)
	fmt.Printf("      (zone %d, palier %q, %d entité(s) présente(s))\n",
		snap.ZoneID, snap.Tier, len(snap.Entities))
	return nil
}

// applySnapshot réinitialise l'état de jeu suivi à partir d'un zone.snapshot.
func (r *runner) applySnapshot(snap snapshotMsg) {
	r.selfID = snap.Self
	r.zoneID = snap.ZoneID
	r.tier = snap.Tier
	r.portals = snap.Portals
	r.pos = make(map[string]xy, len(snap.Entities))
	r.mobs = make(map[string]bool)
	for _, e := range snap.Entities {
		r.pos[e.CharacterID] = xy{e.X, e.Y}
		if e.Mob {
			r.mobs[e.CharacterID] = true
		}
	}
}

// applyDelta met à jour les positions suivies à partir d'un zone.delta.
func (r *runner) applyDelta(d deltaMsg) {
	for _, j := range d.Joined {
		r.pos[j.CharacterID] = xy{j.X, j.Y}
		if j.Mob {
			r.mobs[j.CharacterID] = true
		}
	}
	for _, mv := range d.Moved {
		r.pos[mv.CharacterID] = xy{mv.X, mv.Y}
	}
	for _, id := range d.Left {
		delete(r.pos, id)
		delete(r.mobs, id)
	}
}

func (r *runner) pingPong() error {
	if err := r.sendEnvelope(protocol.TypePing, 111, nil); err != nil {
		return err
	}
	if err := r.awaitReply(protocol.TypePong, 111); err != nil {
		return err
	}
	return nil
}

func (r *runner) echo() error {
	if err := r.sendEnvelope("chat.say", 222, map[string]string{"body": "coucou"}); err != nil {
		return err
	}
	return r.awaitReply(protocol.TypeEcho, 222)
}

// awaitReply lit jusqu'à la réponse attendue (type + seq), en ignorant le trafic
// de fond (zone.delta/snapshot, char.update) que la boucle de tick peut émettre.
func (r *runner) awaitReply(wantType string, wantSeq uint64) error {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		env, err := r.readEnvelope()
		if err != nil {
			return err
		}
		switch env.Type {
		case protocol.TypeZoneDelta, protocol.TypeZoneSnapshot, protocol.TypeCharUpdate:
			continue // trafic de fond : on ignore
		}
		if env.Type != wantType || env.Seq != wantSeq {
			return fmt.Errorf("attendu %s seq=%d, obtenu type=%q seq=%d", wantType, wantSeq, env.Type, env.Seq)
		}
		return nil
	}
	return fmt.Errorf("aucune réponse %s seq=%d reçue dans le délai imparti", wantType, wantSeq)
}

// ── PvE : transition + combat contre un mob ─────────────────────────────────

// transitionToGreen navigue jusqu'au portail du village puis traverse vers la
// zone verte, et vérifie qu'elle contient bien 3 mobs.
func (r *runner) transitionToGreen() error {
	// Choisir un portail (les villages n'exposent qu'un portail vers le vert).
	var portal *portalInfo
	for i := range r.portals {
		if r.portals[i].Kind == "portal" {
			portal = &r.portals[i]
			break
		}
	}
	if portal == nil {
		return fmt.Errorf("aucun portail exposé dans le snapshot du village (palier %q)", r.tier)
	}

	// S'approcher du portail (rayon de traversée = 45).
	if err := r.navigate(xy{portal.X, portal.Y}, 40, nil); err != nil {
		return fmt.Errorf("navigation vers le portail : %w", err)
	}
	// Stopper le mouvement avant de traverser.
	_ = r.sendEnvelope(protocol.TypeMoveIntent, 0, map[string]int{"dx": 0, "dy": 0})

	// Traverser : le serveur répond char.update puis zone.snapshot.
	if err := r.sendEnvelope(protocol.TypeZoneTransition, 0, map[string]int{"link_id": portal.LinkID}); err != nil {
		return err
	}
	snap, err := r.awaitSnapshot(6 * time.Second)
	if err != nil {
		return fmt.Errorf("aucun zone.snapshot après la traversée : %w", err)
	}
	r.applySnapshot(snap)
	if r.tier != "green" {
		return fmt.Errorf("attendu une zone verte, obtenu palier %q (zone %d)", r.tier, r.zoneID)
	}
	if n := len(r.mobs); n != 3 {
		return fmt.Errorf("la zone verte devrait contenir 3 mobs, obtenu %d", n)
	}
	fmt.Printf("      (zone verte %d, %d mobs présents)\n", r.zoneID, len(r.mobs))
	return nil
}

// fightMob approche un mob jusqu'à déclencher le combat, laisse l'IA du mob
// jouer ses tours, attaque jusqu'à la victoire et vérifie le gain d'or. Un mob
// qui fuit (comportement d'IA valide) est réengagé : on retente jusqu'à un kill.
func (r *runner) fightMob() error {
	sawMobAI, sawFlee := false, false
	for attempt := 0; attempt < 6; attempt++ {
		res, err := r.oneCombat()
		if err != nil {
			return err
		}
		if res.mobActed {
			sawMobAI = true
		}
		if res.fled {
			sawFlee = true
			continue // le mob a fui : on le repoursuit
		}
		if !res.killed {
			return fmt.Errorf("le joueur aurait dû vaincre le mob (vainqueur=%q, raison=%q)", res.winner, res.reason)
		}
		if !sawMobAI {
			return fmt.Errorf("le mob n'a jamais joué son tour : l'IA n'a pas agi")
		}
		if res.gold <= 0 {
			return fmt.Errorf("le joueur aurait dû gagner de l'or (or reçu : %d)", res.gold)
		}
		note := ""
		if sawFlee {
			note = " (après une fuite du mob : IA de fuite observée)"
		}
		fmt.Printf("      (mob vaincu ; l'IA a joué ses tours%s ; or gagné : %d)\n", note, res.gold)
		return nil
	}
	return fmt.Errorf("le mob a fui trop de fois d'affilée sans être vaincu")
}

// checkRespawn vérifie qu'après la mort d'un mob, un nouveau mob réapparaît
// (delta « joined » avec mob=true), preuve que la zone se repeuple.
func (r *runner) checkRespawn() error {
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		_ = r.ws.SetReadDeadline(deadline)
		_, raw, err := r.ws.ReadMessage()
		if err != nil {
			return fmt.Errorf("aucune réapparition de mob observée : %w", err)
		}
		env, err := protocol.Decode(raw)
		if err != nil {
			return err
		}
		if env.Type != protocol.TypeZoneDelta {
			continue // on ignore combat.* et char.update pendant la veille
		}
		var d deltaMsg
		if err := env.DecodeData(&d); err != nil {
			continue
		}
		for _, j := range d.Joined {
			if j.Mob {
				fmt.Printf("      (mob réapparu : %s)\n", j.CharacterID)
				return nil
			}
		}
	}
	return fmt.Errorf("aucun mob n'a réapparu dans le délai imparti")
}

type combatResult struct {
	killed   bool
	fled     bool
	mobActed bool
	gold     int64
	winner   string
	reason   string
}

// oneCombat approche le mob le plus proche, engage un unique combat et le joue
// jusqu'à son terme (kill ou fuite).
func (r *runner) oneCombat() (combatResult, error) {
	var res combatResult

	var start struct {
		Turn      string `json:"turn"`
		Opponents []struct {
			CharacterID string `json:"character_id"`
			Mob         bool   `json:"mob"`
		} `json:"opponents"`
	}
	var startEnv *protocol.Envelope
	err := r.navigate(xy{}, 0, func(env protocol.Envelope) bool {
		if env.Type == protocol.TypeCombatStart {
			e := env
			startEnv = &e
			return true // stopper la navigation
		}
		return false
	})
	if err != nil {
		return res, fmt.Errorf("navigation vers le mob : %w", err)
	}
	if startEnv == nil {
		return res, fmt.Errorf("aucun combat PvE déclenché en approchant un mob")
	}
	if err := startEnv.DecodeData(&start); err != nil {
		return res, err
	}
	mobID := ""
	for _, o := range start.Opponents {
		if o.Mob {
			mobID = o.CharacterID
		}
	}
	if mobID == "" {
		return res, fmt.Errorf("le combat déclenché n'oppose pas de mob : %+v", start.Opponents)
	}

	// Attaquer à notre tour ; l'IA du mob joue seule. La récompense (char.update)
	// arrive juste avant combat.end : on la capture au vol.
	turn := start.Turn
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if turn == r.selfID {
			if err := r.sendEnvelope(protocol.TypeCombatAction, 0, map[string]string{"action": "attack"}); err != nil {
				return res, err
			}
			turn = "" // évite de renvoyer l'action avant le prochain événement
		}
		env, err := r.readEnvelope()
		if err != nil {
			return res, fmt.Errorf("lecture pendant le combat : %w", err)
		}
		switch env.Type {
		case protocol.TypeCharUpdate:
			var cs struct {
				Gold int64 `json:"gold"`
			}
			if err := env.DecodeData(&cs); err == nil {
				res.gold = cs.Gold
			}
		case protocol.TypeCombatEvent:
			var ev struct {
				Actor string `json:"actor"`
				Turn  string `json:"turn"`
			}
			if err := env.DecodeData(&ev); err != nil {
				return res, err
			}
			if ev.Actor == mobID {
				res.mobActed = true // le mob a agi de lui-même (IA)
			}
			turn = ev.Turn
		case protocol.TypeCombatEnd:
			var end struct {
				Winner string `json:"winner"`
				Reason string `json:"reason"`
			}
			if err := env.DecodeData(&end); err != nil {
				return res, err
			}
			res.winner, res.reason = end.Winner, end.Reason
			res.killed = end.Reason == "death" && end.Winner == r.selfID
			res.fled = end.Reason == "flee"
			return res, nil
		}
	}
	return res, fmt.Errorf("le combat ne s'est pas terminé dans le délai imparti")
}

// navigate déplace le joueur vers une cible en suivant les zone.delta. Si onMsg
// est fourni et renvoie true pour un message, la navigation s'arrête aussitôt
// (utilisé pour stopper dès combat.start). Sinon elle s'arrête en arrivant à
// arriveDist de la cible ; avec une cible nulle, elle vise le mob le plus proche.
func (r *runner) navigate(target xy, arriveDist int, onMsg func(protocol.Envelope) bool) error {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		dst := target
		if onMsg != nil { // mode « viser le mob le plus proche »
			m, ok := r.nearestMob()
			if !ok {
				return fmt.Errorf("plus aucun mob à cibler")
			}
			dst = m
		}
		self := r.pos[r.selfID]
		if onMsg == nil && dist2(self, dst) <= arriveDist*arriveDist {
			return nil // arrivé
		}
		_ = r.sendEnvelope(protocol.TypeMoveIntent, 0, map[string]int{
			"dx": signum(dst.X - self.X), "dy": signum(dst.Y - self.Y),
		})
		env, err := r.readEnvelope()
		if err != nil {
			return err
		}
		switch env.Type {
		case protocol.TypeZoneDelta:
			var d deltaMsg
			if err := env.DecodeData(&d); err == nil {
				r.applyDelta(d)
			}
		}
		if onMsg != nil && onMsg(env) {
			return nil
		}
	}
	return fmt.Errorf("cible non atteinte dans le délai imparti")
}

func (r *runner) nearestMob() (xy, bool) {
	self := r.pos[r.selfID]
	best, found := xy{}, false
	bestD := 1 << 30
	for id := range r.mobs {
		p, ok := r.pos[id]
		if !ok {
			continue
		}
		if d := dist2(self, p); d < bestD {
			bestD, best, found = d, p, true
		}
	}
	return best, found
}

// awaitSnapshot lit jusqu'au prochain zone.snapshot (en ignorant char.update et
// deltas), dans le délai imparti.
func (r *runner) awaitSnapshot(timeout time.Duration) (snapshotMsg, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		env, err := r.readEnvelope()
		if err != nil {
			return snapshotMsg{}, err
		}
		if env.Type == protocol.TypeZoneSnapshot {
			var s snapshotMsg
			if err := env.DecodeData(&s); err != nil {
				return snapshotMsg{}, err
			}
			return s, nil
		}
	}
	return snapshotMsg{}, fmt.Errorf("délai dépassé")
}

func dist2(a, b xy) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx + dy*dy
}

func signum(v int) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
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
