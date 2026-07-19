// Package zone gère l'état vivant des zones (§4, §9).
//
// Chaque zone est un ACTEUR : une unique goroutine possède tout son état
// (présences, positions, combats) et le fait évoluer à chaque tick — sans verrou.
//
//   - T3 : entrée en zone et snapshot.
//   - T4 : boucle de tick + mouvement autoritatif + deltas.
//   - T5 : combat au tour par tour déclenché par la rencontre (combat.go).
//   - Monde à paliers : transitions entre zones par portails/barrières, PvP et
//     pénalité d'XP à la mort selon le palier de la zone, réapparition au village.
package zone

import (
	"math/rand"
	"sync"
	"time"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
)

const (
	moveSpeedPerSec = 48
	worldBound      = 480
	mailboxSize     = 256

	encounterRadius  = 12
	turnTimeoutSec   = 15
	engageCooldown   = 3
	transitionRadius = 45 // distance max d'un portail pour l'emprunter
	combatXPReward   = 50 // XP gagnée en remportant un combat PvP
)

// Sender pousse un message à un joueur.
type Sender interface {
	SendEnvelope(msgType string, seq uint64, data any)
}

// Client est ce qu'une zone connaît d'un joueur connecté : de quoi lui envoyer
// des messages et le relocaliser (réapparition au village après la mort). La
// session de la gateway le satisfait.
type Client interface {
	Sender
	// Relocate demande le placement du joueur dans une autre zone (non bloquant
	// pour la zone appelante).
	Relocate(char domain.Character)
}

// Intent est une intention de déplacement (direction, chaque axe dans {-1,0,1}).
type Intent struct {
	DX int `json:"dx"`
	DY int `json:"dy"`
}

// Portal est un point de transition exposé au client dans le snapshot.
type Portal struct {
	LinkID int    `json:"link_id"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Kind   string `json:"kind"` // portal | barrier
	ToZone string `json:"to_zone"`
}

// ZoneMeta décrit les attributs statiques d'une zone, calculés au démarrage.
type ZoneMeta struct {
	ID          int
	Code        string
	Name        string
	Tier        string // village | green | orange | red
	PvP         bool
	DeathXPLoss float64 // fraction d'XP perdue à la mort (0, 0.5, 1)
	Portals     []Portal
}

// Link est un lien de transition entre deux zones.
type Link struct {
	ID         int
	FromZoneID int
	ToZoneID   int
	Kind       string
	FromX      int
	FromY      int
	ToX        int
	ToY        int
	MinLevel   int
}

// EntityState est la représentation réseau d'une entité (§10).
type EntityState struct {
	CharacterID string `json:"character_id"`
	Name        string `json:"name"`
	FactionID   int    `json:"faction_id"`
	Level       int    `json:"level"`
	HP          int    `json:"hp"`
	MaxHP       int    `json:"max_hp"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
}

func entityStateOf(c domain.Character) EntityState {
	return EntityState{
		CharacterID: c.ID, Name: c.Name, FactionID: c.FactionID,
		Level: c.Level, HP: c.HP, MaxHP: c.MaxHP, X: c.X, Y: c.Y,
	}
}

// SnapshotData est l'état complet d'une zone, envoyé à l'entrée (T3) et après
// chaque transition.
type SnapshotData struct {
	ZoneID   int           `json:"zone_id"`
	ZoneName string        `json:"zone_name"`
	Tier     string        `json:"tier"`
	PvP      bool          `json:"pvp"`
	Self     string        `json:"self"`
	Entities []EntityState `json:"entities"`
	Portals  []Portal      `json:"portals,omitempty"`
	Tick     uint64        `json:"tick"`
}

// MovedEntity décrit la nouvelle position d'une entité qui a bougé.
type MovedEntity struct {
	CharacterID string `json:"character_id"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
}

// DeltaData diffuse les changements d'une zone sur un tick (T4).
type DeltaData struct {
	Tick   uint64        `json:"tick"`
	Joined []EntityState `json:"joined,omitempty"`
	Moved  []MovedEntity `json:"moved,omitempty"`
	Left   []string      `json:"left,omitempty"`
}

// member est l'état vivant d'un personnage présent dans une zone.
type member struct {
	char   domain.Character
	client Client
	intent Intent

	combatID      string
	cooldownUntil uint64
}

func (m *member) inCombat() bool { return m.combatID != "" }

// Zone est un acteur : sa goroutine loop() possède tout l'état ci-dessous.
type Zone struct {
	meta ZoneMeta
	hz   int
	step int

	cmds chan func()
	quit chan struct{}

	members map[string]*member
	combats map[string]*combat
	combatN int

	joined []string
	left   []string
	dirty  map[string]bool

	tick             uint64
	turnTimeoutTicks uint64
	cooldownTicks    uint64
	rng              *rand.Rand
}

func newZone(meta ZoneMeta, hz int) *Zone {
	step := moveSpeedPerSec / hz
	if step < 1 {
		step = 1
	}
	z := &Zone{
		meta: meta, hz: hz, step: step,
		cmds:    make(chan func(), mailboxSize),
		quit:    make(chan struct{}),
		members: make(map[string]*member),
		combats: make(map[string]*combat),
		dirty:   make(map[string]bool),

		turnTimeoutTicks: uint64(turnTimeoutSec * hz),
		cooldownTicks:    uint64(engageCooldown * hz),
		rng:              rand.New(rand.NewSource(int64(meta.ID)*7919 + time.Now().UnixNano())),
	}
	go z.loop()
	return z
}

func (z *Zone) loop() {
	ticker := time.NewTicker(time.Second / time.Duration(z.hz))
	defer ticker.Stop()
	for {
		select {
		case <-z.quit:
			return
		case fn := <-z.cmds:
			fn()
		case <-ticker.C:
			z.doTick()
		}
	}
}

func (z *Zone) doTick() {
	z.tick++

	var moved []MovedEntity
	for id, m := range z.members {
		if m.inCombat() || (m.intent.DX == 0 && m.intent.DY == 0) {
			continue
		}
		nx := clamp(m.char.X + m.intent.DX*z.step)
		ny := clamp(m.char.Y + m.intent.DY*z.step)
		if nx != m.char.X || ny != m.char.Y {
			m.char.X, m.char.Y = nx, ny
			moved = append(moved, MovedEntity{CharacterID: id, X: nx, Y: ny})
		}
	}

	z.resolveTimedOutTurns()

	for id := range z.dirty {
		if m, ok := z.members[id]; ok {
			moved = append(moved, MovedEntity{CharacterID: id, X: m.char.X, Y: m.char.Y})
		}
		delete(z.dirty, id)
	}

	z.detectEncounters()

	var joined []EntityState
	for _, id := range z.joined {
		if m, ok := z.members[id]; ok {
			joined = append(joined, entityStateOf(m.char))
		}
	}
	left := z.left
	z.joined, z.left = nil, nil

	if len(joined) == 0 && len(moved) == 0 && len(left) == 0 {
		return
	}
	delta := DeltaData{Tick: z.tick, Joined: joined, Moved: moved, Left: left}
	for _, m := range z.members {
		m.client.SendEnvelope(protocol.TypeZoneDelta, 0, delta)
	}
}

// ── Commandes ───────────────────────────────────────────────────────────────

func (z *Zone) snapshotFor(charID string) SnapshotData {
	entities := make([]EntityState, 0, len(z.members))
	for _, m := range z.members {
		entities = append(entities, entityStateOf(m.char))
	}
	return SnapshotData{
		ZoneID: z.meta.ID, ZoneName: z.meta.Name, Tier: z.meta.Tier, PvP: z.meta.PvP,
		Self: charID, Entities: entities, Portals: z.meta.Portals, Tick: z.tick,
	}
}

func (z *Zone) enter(char domain.Character, client Client) SnapshotData {
	reply := make(chan SnapshotData, 1)
	z.cmds <- func() {
		z.members[char.ID] = &member{char: char, client: client}
		z.joined = append(z.joined, char.ID)
		reply <- z.snapshotFor(char.ID)
	}
	return <-reply
}

func (z *Zone) leave(charID string) {
	z.cmds <- func() {
		m := z.members[charID]
		if m == nil {
			return
		}
		if m.inCombat() {
			if c := z.combats[m.combatID]; c != nil {
				z.endByForfeit(c, charID)
			}
		}
		delete(z.members, charID)
		z.left = append(z.left, charID)
	}
}

func (z *Zone) move(charID string, in Intent) {
	in = Intent{DX: sign(in.DX), DY: sign(in.DY)}
	z.cmds <- func() {
		if m := z.members[charID]; m != nil && !m.inCombat() {
			m.intent = in
		}
	}
}

func (z *Zone) combatAction(charID, action string) {
	z.cmds <- func() {
		m := z.members[charID]
		if m == nil || !m.inCombat() {
			return
		}
		c := z.combats[m.combatID]
		if c == nil {
			return
		}
		if c.turn != charID {
			m.client.SendEnvelope(protocol.TypeError, 0, protocol.ErrorData{
				Code: "not_your_turn", Message: "ce n'est pas ton tour",
			})
			return
		}
		if action != "attack" && action != "flee" {
			action = "attack"
		}
		z.resolveAction(c, charID, action)
	}
}

func (z *Zone) count() int {
	reply := make(chan int, 1)
	z.cmds <- func() { reply <- len(z.members) }
	return <-reply
}

// currentChar retourne l'état autoritatif d'un personnage présent dans la zone
// (position, PV, XP à jour), possédé par la goroutine de la zone.
func (z *Zone) currentChar(charID string) (domain.Character, bool) {
	type result struct {
		c  domain.Character
		ok bool
	}
	reply := make(chan result, 1)
	z.cmds <- func() {
		if m := z.members[charID]; m != nil {
			reply <- result{m.char, true}
		} else {
			reply <- result{domain.Character{}, false}
		}
	}
	r := <-reply
	return r.c, r.ok
}

// ── Gestionnaire de zones ───────────────────────────────────────────────────

// Manager détient toutes les zones actives, leurs métadonnées et les liens de
// transition.
type Manager struct {
	hz    int
	metas map[int]ZoneMeta
	links map[int]Link
	mu    sync.Mutex
	zones map[int]*Zone
}

// NewManager crée le gestionnaire à partir des métadonnées de zones et des liens
// de transition (chargés au démarrage).
func NewManager(hz int, metas map[int]ZoneMeta, links map[int]Link) *Manager {
	if hz <= 0 {
		hz = 1
	}
	if metas == nil {
		metas = map[int]ZoneMeta{}
	}
	if links == nil {
		links = map[int]Link{}
	}
	return &Manager{hz: hz, metas: metas, links: links, zones: make(map[int]*Zone)}
}

func (m *Manager) zoneFor(id int) *Zone {
	m.mu.Lock()
	defer m.mu.Unlock()
	z, ok := m.zones[id]
	if !ok {
		meta, has := m.metas[id]
		if !has {
			meta = ZoneMeta{ID: id, Name: "Zone inconnue", Tier: "green"}
		}
		z = newZone(meta, m.hz)
		m.zones[id] = z
	}
	return z
}

func (m *Manager) lookup(id int) *Zone {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.zones[id]
}

// Enter place un personnage dans sa zone et retourne le snapshot.
func (m *Manager) Enter(char domain.Character, client Client) SnapshotData {
	return m.zoneFor(char.ZoneID).enter(char, client)
}

// Leave retire un personnage de sa zone (idempotent).
func (m *Manager) Leave(char domain.Character) {
	if z := m.lookup(char.ZoneID); z != nil {
		z.leave(char.ID)
	}
}

// Move enregistre l'intention de déplacement d'un personnage.
func (m *Manager) Move(char domain.Character, in Intent) {
	if z := m.lookup(char.ZoneID); z != nil {
		z.move(char.ID, in)
	}
}

// CombatAction transmet l'action de combat d'un personnage à sa zone.
func (m *Manager) CombatAction(char domain.Character, action string) {
	if z := m.lookup(char.ZoneID); z != nil {
		z.combatAction(char.ID, action)
	}
}

// TransitionError décrit pourquoi une transition a été refusée.
type TransitionError struct {
	Code    string
	Message string
}

func (e *TransitionError) Error() string { return e.Message }

// Transition emprunte un lien (portail/barrière) depuis la zone courante du
// personnage. Retourne le personnage relocalisé et le snapshot de la nouvelle
// zone. Appelé depuis la goroutine de la session (client-initié).
func (m *Manager) Transition(char domain.Character, linkID int, client Client) (domain.Character, SnapshotData, error) {
	m.mu.Lock()
	link, ok := m.links[linkID]
	m.mu.Unlock()
	if !ok {
		return char, SnapshotData{}, &TransitionError{"unknown_link", "portail inconnu"}
	}
	if link.FromZoneID != char.ZoneID {
		return char, SnapshotData{}, &TransitionError{"wrong_zone", "ce portail n'est pas dans ta zone"}
	}

	// La position (et l'XP) autoritatives vivent dans l'acteur de zone : on les
	// relit avant de valider la proximité au portail.
	if z := m.lookup(char.ZoneID); z != nil {
		if cur, present := z.currentChar(char.ID); present {
			char = cur
		}
	}

	if dx, dy := char.X-link.FromX, char.Y-link.FromY; dx*dx+dy*dy > transitionRadius*transitionRadius {
		return char, SnapshotData{}, &TransitionError{"too_far", "approche-toi du portail"}
	}
	if char.Level < link.MinLevel {
		return char, SnapshotData{}, &TransitionError{"level_too_low", "niveau insuffisant pour cette barrière"}
	}

	m.Leave(char)
	char.ZoneID = link.ToZoneID
	char.X, char.Y = link.ToX, link.ToY
	snap := m.Enter(char, client)
	return char, snap, nil
}

// CurrentChar retourne l'état autoritatif d'un personnage dans sa zone (position,
// PV, XP), ou le personnage inchangé s'il n'y est plus présent.
func (m *Manager) CurrentChar(char domain.Character) (domain.Character, bool) {
	if z := m.lookup(char.ZoneID); z != nil {
		return z.currentChar(char.ID)
	}
	return char, false
}

// Count retourne le nombre de personnages présents dans une zone.
func (m *Manager) Count(zoneID int) int {
	if z := m.lookup(zoneID); z != nil {
		return z.count()
	}
	return 0
}

// Close arrête les boucles de toutes les zones.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, z := range m.zones {
		close(z.quit)
	}
	m.zones = make(map[int]*Zone)
}

func clamp(v int) int {
	if v > worldBound {
		return worldBound
	}
	if v < -worldBound {
		return -worldBound
	}
	return v
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	default:
		return 0
	}
}
