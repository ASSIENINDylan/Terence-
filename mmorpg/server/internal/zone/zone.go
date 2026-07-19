// Package zone gère l'état vivant des zones (§4, §9).
//
// Depuis T4, chaque zone est un ACTEUR : une unique goroutine possède tout son
// état (présences, positions) et le fait évoluer à chaque tick. Toutes les
// interactions (entrer, sortir, intention de mouvement) sont des commandes
// déposées dans une mailbox et exécutées par cette goroutine — donc sans verrou
// ni course de données. Le serveur est seul maître des positions : le client
// n'envoie qu'une intention de direction, jamais une position.
//
// La perte de cet état (redémarrage) n'entraîne aucune perte durable : les
// positions de départ sont reprises depuis PostgreSQL (la persistance périodique
// des positions viendra en T8).
package zone

import (
	"sync"
	"time"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
)

// Paramètres de mouvement (v0.1). La vitesse est exprimée en unités par seconde
// et convertie en pas par tick, pour rester constante quelle que soit la
// fréquence de tick.
const (
	moveSpeedPerSec = 48  // vitesse d'un personnage
	worldBound      = 480 // limites de la zone : positions bornées à ±worldBound
	mailboxSize     = 256 // commandes en attente avant saturation (peu probable)
)

// Sender est ce dont une zone a besoin pour pousser des messages à un joueur.
// *gateway.Conn le satisfait ; cette interface évite à zone de dépendre de la
// gateway (pas de cycle).
type Sender interface {
	SendEnvelope(msgType string, seq uint64, data any)
}

// Intent est une intention de déplacement : une direction, chaque composante
// dans {-1, 0, 1}. Elle persiste jusqu'à la prochaine intention (un vecteur nul
// arrête le personnage).
type Intent struct {
	DX int `json:"dx"`
	DY int `json:"dy"`
}

// EntityState est la représentation réseau d'une entité. On n'expose au client
// que ce qu'il a le droit de voir (§10).
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

// SnapshotData est l'état complet et cohérent d'une zone, envoyé à un joueur qui
// entre (T3).
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

// DeltaData diffuse les changements d'une zone sur un tick (T4). Un delta n'est
// émis que s'il contient au moins un changement.
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
}

// Zone est un acteur : sa goroutine loop() possède members et l'espace de
// changements en cours (joined/left).
type Zone struct {
	id   int
	hz   int
	step int

	cmds chan func()
	quit chan struct{}

	members map[string]*member
	joined  []string // ids entrés depuis le dernier tick
	left    []string // ids sortis depuis le dernier tick
	tick    uint64
}

func newZone(id, hz int) *Zone {
	step := moveSpeedPerSec / hz
	if step < 1 {
		step = 1
	}
	z := &Zone{
		id: id, hz: hz, step: step,
		cmds:    make(chan func(), mailboxSize),
		quit:    make(chan struct{}),
		members: make(map[string]*member),
	}
	go z.loop()
	return z
}

// loop est l'unique propriétaire de l'état de la zone : il exécute les commandes
// et fait avancer le temps à chaque tick.
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

// doTick applique les intentions de mouvement, calcule les changements et diffuse
// le zone.delta correspondant aux membres présents.
func (z *Zone) doTick() {
	z.tick++

	var moved []MovedEntity
	for id, m := range z.members {
		if m.intent.DX == 0 && m.intent.DY == 0 {
			continue
		}
		nx := clamp(m.char.X + m.intent.DX*z.step)
		ny := clamp(m.char.Y + m.intent.DY*z.step)
		if nx != m.char.X || ny != m.char.Y {
			m.char.X, m.char.Y = nx, ny
			moved = append(moved, MovedEntity{CharacterID: id, X: nx, Y: ny})
		}
	}

	var joined []EntityState
	for _, id := range z.joined {
		if m, ok := z.members[id]; ok {
			joined = append(joined, entityStateOf(m.char))
		}
	}
	left := z.left
	z.joined, z.left = nil, nil

	if len(joined) == 0 && len(moved) == 0 && len(left) == 0 {
		return // rien à diffuser ce tick
	}

	delta := DeltaData{Tick: z.tick, Joined: joined, Moved: moved, Left: left}
	for _, m := range z.members {
		m.sender.SendEnvelope(protocol.TypeZoneDelta, 0, delta)
	}
}

// enter ajoute un membre et retourne l'état courant de la zone (snapshot). Les
// autres membres apprendront son arrivée via le prochain zone.delta.
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

// leave retire un membre ; son départ sera diffusé au prochain tick.
func (z *Zone) leave(charID string) {
	z.cmds <- func() {
		if _, ok := z.members[charID]; ok {
			delete(z.members, charID)
			z.left = append(z.left, charID)
		}
	}
}

// move enregistre l'intention de déplacement d'un membre (appliquée aux ticks
// suivants). La direction est bornée à {-1, 0, 1} par axe : le client ne peut
// pas « accélérer » en trichant sur l'amplitude.
func (z *Zone) move(charID string, in Intent) {
	in = Intent{DX: sign(in.DX), DY: sign(in.DY)}
	z.cmds <- func() {
		if m, ok := z.members[charID]; ok {
			m.intent = in
		}
	}
}

// count retourne le nombre de membres présents (diagnostic, tests).
func (z *Zone) count() int {
	reply := make(chan int, 1)
	z.cmds <- func() { reply <- len(z.members) }
	return <-reply
}

// Manager détient toutes les zones actives et route les commandes vers la bonne.
type Manager struct {
	hz    int
	mu    sync.Mutex
	zones map[int]*Zone
}

// NewManager crée un gestionnaire de zones. hz est la fréquence de tick des
// zones (config TICK_HZ). Les zones sont instanciées à la demande.
func NewManager(hz int) *Manager {
	if hz <= 0 {
		hz = 1
	}
	return &Manager{hz: hz, zones: make(map[int]*Zone)}
}

func (m *Manager) zoneFor(id int) *Zone {
	m.mu.Lock()
	defer m.mu.Unlock()
	z, ok := m.zones[id]
	if !ok {
		z = newZone(id, m.hz)
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

// Leave retire un personnage de sa zone (déconnexion). Idempotent.
func (m *Manager) Leave(char domain.Character) {
	if z := m.lookup(char.ZoneID); z != nil {
		z.leave(char.ID)
	}
}

// Move enregistre l'intention de déplacement d'un personnage dans sa zone.
func (m *Manager) Move(char domain.Character, in Intent) {
	if z := m.lookup(char.ZoneID); z != nil {
		z.move(char.ID, in)
	}
}

// Count retourne le nombre de personnages présents dans une zone.
func (m *Manager) Count(zoneID int) int {
	if z := m.lookup(zoneID); z != nil {
		return z.count()
	}
	return 0
}

// Close arrête les boucles de toutes les zones (arrêt propre du serveur).
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
