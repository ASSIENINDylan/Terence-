// Package zone gère l'état vivant des zones (§4, §9).
//
// Chaque zone est un ACTEUR : une unique goroutine possède tout son état
// (présences, positions, combats) et le fait évoluer à chaque tick. Toutes les
// interactions (entrer, sortir, bouger, agir en combat) sont des commandes
// déposées dans une mailbox et exécutées par cette goroutine — donc sans verrou.
//
//   - T3 : entrée en zone et snapshot.
//   - T4 : boucle de tick + mouvement autoritatif + deltas.
//   - T5 : combat au tour par tour déclenché par la rencontre de deux
//     personnages de factions opposées dans une zone PvP (voir combat.go).
//
// La perte de cet état (redémarrage) n'entraîne aucune perte durable : les
// positions de départ sont reprises depuis PostgreSQL (write-back en T8).
package zone

import (
	"math/rand"
	"sync"
	"time"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
)

const (
	moveSpeedPerSec = 48  // vitesse d'un personnage (unités/s)
	worldBound      = 480 // limites de la zone : positions bornées à ±worldBound
	mailboxSize     = 256

	encounterRadius = 12 // distance d'engagement d'un combat (unités)
	turnTimeoutSec  = 15 // délai max d'un tour avant attaque automatique
	engageCooldown  = 3  // secondes d'immunité après un combat (anti-re-engagement)
)

// Sender est ce dont une zone a besoin pour pousser des messages à un joueur.
type Sender interface {
	SendEnvelope(msgType string, seq uint64, data any)
}

// Intent est une intention de déplacement (direction, chaque axe dans {-1,0,1}).
type Intent struct {
	DX int `json:"dx"`
	DY int `json:"dy"`
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

// SnapshotData est l'état complet d'une zone, envoyé à l'entrée (T3).
type SnapshotData struct {
	ZoneID   int           `json:"zone_id"`
	Self     string        `json:"self"`
	Entities []EntityState `json:"entities"`
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
	sender Sender
	intent Intent

	combatID      string // "" = libre de se déplacer ; sinon en combat
	cooldownUntil uint64 // tick avant lequel aucun nouveau combat ne peut s'engager
}

func (m *member) inCombat() bool { return m.combatID != "" }

// Zone est un acteur : sa goroutine loop() possède tout l'état ci-dessous.
type Zone struct {
	id   int
	hz   int
	step int
	pvp  bool

	cmds chan func()
	quit chan struct{}

	members map[string]*member
	combats map[string]*combat
	combatN int

	joined []string        // ids entrés depuis le dernier tick
	left   []string        // ids sortis depuis le dernier tick
	dirty  map[string]bool // positions changées hors mouvement (respawns)

	tick             uint64
	turnTimeoutTicks uint64
	cooldownTicks    uint64
	rng              *rand.Rand
}

func newZone(id, hz int, pvp bool) *Zone {
	step := moveSpeedPerSec / hz
	if step < 1 {
		step = 1
	}
	z := &Zone{
		id: id, hz: hz, step: step, pvp: pvp,
		cmds:    make(chan func(), mailboxSize),
		quit:    make(chan struct{}),
		members: make(map[string]*member),
		combats: make(map[string]*combat),
		dirty:   make(map[string]bool),

		turnTimeoutTicks: uint64(turnTimeoutSec * hz),
		cooldownTicks:    uint64(engageCooldown * hz),
		rng:              rand.New(rand.NewSource(int64(id)*7919 + time.Now().UnixNano())),
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

// doTick fait avancer le temps : mouvement, tours de combat en dépassement de
// délai, détection des rencontres, puis diffusion du zone.delta.
func (z *Zone) doTick() {
	z.tick++

	// 1. Mouvement autoritatif (les personnages en combat ne bougent pas).
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

	// 2. Tours de combat expirés → attaque automatique du joueur passif.
	z.resolveTimedOutTurns()

	// 3. Positions changées par un respawn → à diffuser aussi.
	for id := range z.dirty {
		if m, ok := z.members[id]; ok {
			moved = append(moved, MovedEntity{CharacterID: id, X: m.char.X, Y: m.char.Y})
		}
		delete(z.dirty, id)
	}

	// 4. Rencontres → engagement de combats.
	z.detectEncounters()

	// 5. Diffusion du delta.
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
		m.sender.SendEnvelope(protocol.TypeZoneDelta, 0, delta)
	}
}

// ── Commandes (exécutées dans la goroutine de la zone) ──────────────────────

func (z *Zone) enter(char domain.Character, sender Sender) SnapshotData {
	reply := make(chan SnapshotData, 1)
	z.cmds <- func() {
		z.members[char.ID] = &member{char: char, sender: sender}
		z.joined = append(z.joined, char.ID)

		entities := make([]EntityState, 0, len(z.members))
		for _, m := range z.members {
			entities = append(entities, entityStateOf(m.char))
		}
		reply <- SnapshotData{ZoneID: z.id, Self: char.ID, Entities: entities, Tick: z.tick}
	}
	return <-reply
}

func (z *Zone) leave(charID string) {
	z.cmds <- func() {
		m := z.members[charID]
		if m == nil {
			return
		}
		// Un départ en plein combat met fin au combat (forfait).
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

// combatAction traite l'action du joueur actif (voir combat.go).
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
			m.sender.SendEnvelope(protocol.TypeError, 0, protocol.ErrorData{
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

// ── Gestionnaire de zones ───────────────────────────────────────────────────

// Manager détient toutes les zones actives et route les commandes.
type Manager struct {
	hz    int
	pvp   map[int]bool // zones où le combat est autorisé
	mu    sync.Mutex
	zones map[int]*Zone
}

// NewManager crée un gestionnaire de zones. hz est la fréquence de tick ; pvp
// indique, par identifiant de zone, si le combat y est autorisé.
func NewManager(hz int, pvp map[int]bool) *Manager {
	if hz <= 0 {
		hz = 1
	}
	if pvp == nil {
		pvp = map[int]bool{}
	}
	return &Manager{hz: hz, pvp: pvp, zones: make(map[int]*Zone)}
}

func (m *Manager) zoneFor(id int) *Zone {
	m.mu.Lock()
	defer m.mu.Unlock()
	z, ok := m.zones[id]
	if !ok {
		z = newZone(id, m.hz, m.pvp[id])
		m.zones[id] = z
	}
	return z
}

func (m *Manager) lookup(id int) *Zone {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.zones[id]
}

// Enter place un personnage dans sa zone et retourne le snapshot à lui envoyer.
func (m *Manager) Enter(char domain.Character, sender Sender) SnapshotData {
	return m.zoneFor(char.ZoneID).enter(char, sender)
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

// CombatAction transmet l'action de combat d'un personnage à sa zone (T5).
func (m *Manager) CombatAction(char domain.Character, action string) {
	if z := m.lookup(char.ZoneID); z != nil {
		z.combatAction(char.ID, action)
	}
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
