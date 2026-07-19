// Package zone gère l'état vivant des zones : quels personnages y sont présents,
// et la fabrication des vues envoyées au client (§4, §9).
//
// En T3, une zone est un registre en mémoire des présences, protégé par mutex :
// entrer, sortir, et produire un zone.snapshot (photo de l'état à l'entrée).
// La boucle de tick et les deltas de mouvement viendront s'y greffer en T4.
//
// La perte de cet état (redémarrage) n'entraîne aucune perte durable : les
// positions sont reprises depuis PostgreSQL, seule une reconnexion est requise.
package zone

import (
	"sync"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
)

// Sender est ce dont une zone a besoin pour pousser des messages à un joueur.
// *gateway.Conn le satisfait. Cette interface évite à zone de dépendre de la
// gateway (pas de cycle) et prépare l'envoi des deltas (T4).
type Sender interface {
	SendEnvelope(msgType string, seq uint64, data any)
}

// EntityState est la représentation réseau d'une entité dans un snapshot.
// On n'expose au client que ce qu'il a le droit de voir (§10) : jamais les
// statistiques internes complètes ni les identifiants de compte.
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
		CharacterID: c.ID,
		Name:        c.Name,
		FactionID:   c.FactionID,
		Level:       c.Level,
		HP:          c.HP,
		MaxHP:       c.MaxHP,
		X:           c.X,
		Y:           c.Y,
	}
}

// SnapshotData est la charge utile d'un message zone.snapshot : l'état complet
// et cohérent de la zone au moment de l'entrée du joueur.
type SnapshotData struct {
	ZoneID   int           `json:"zone_id"`
	Self     string        `json:"self"` // character_id du destinataire
	Entities []EntityState `json:"entities"`
}

// presence associe un personnage présent dans une zone au canal permettant de
// lui pousser des messages (utilisé pour les deltas en T4).
type presence struct {
	char   domain.Character
	sender Sender
}

// Zone est le registre en mémoire des présences d'une zone.
type Zone struct {
	id      int
	mu      sync.RWMutex
	members map[string]*presence // clé : character_id
}

func newZone(id int) *Zone {
	return &Zone{id: id, members: make(map[string]*presence)}
}

// snapshot construit une photo cohérente de la zone pour le destinataire self.
func (z *Zone) snapshot(self string) SnapshotData {
	z.mu.RLock()
	defer z.mu.RUnlock()

	entities := make([]EntityState, 0, len(z.members))
	for _, p := range z.members {
		entities = append(entities, entityStateOf(p.char))
	}
	return SnapshotData{ZoneID: z.id, Self: self, Entities: entities}
}

// Manager détient toutes les zones actives et route les joueurs vers la leur.
type Manager struct {
	mu    sync.Mutex
	zones map[int]*Zone
}

// NewManager crée un gestionnaire de zones vide. Les zones sont instanciées à la
// demande, quand un premier joueur y entre.
func NewManager() *Manager {
	return &Manager{zones: make(map[int]*Zone)}
}

// zoneFor retourne la zone d'identifiant id, en la créant si besoin.
func (m *Manager) zoneFor(id int) *Zone {
	m.mu.Lock()
	defer m.mu.Unlock()
	z, ok := m.zones[id]
	if !ok {
		z = newZone(id)
		m.zones[id] = z
	}
	return z
}

// Enter place un personnage dans sa zone (char.ZoneID) et retourne le snapshot
// à lui envoyer : l'état de la zone incluant le personnage qui vient d'entrer.
func (m *Manager) Enter(char domain.Character, sender Sender) SnapshotData {
	z := m.zoneFor(char.ZoneID)

	z.mu.Lock()
	z.members[char.ID] = &presence{char: char, sender: sender}
	z.mu.Unlock()

	return z.snapshot(char.ID)
}

// Leave retire un personnage de sa zone (déconnexion). Idempotent.
func (m *Manager) Leave(char domain.Character) {
	m.mu.Lock()
	z, ok := m.zones[char.ZoneID]
	m.mu.Unlock()
	if !ok {
		return
	}
	z.mu.Lock()
	delete(z.members, char.ID)
	z.mu.Unlock()
}

// Count retourne le nombre de personnages présents dans une zone (diagnostic,
// tests).
func (m *Manager) Count(zoneID int) int {
	m.mu.Lock()
	z, ok := m.zones[zoneID]
	m.mu.Unlock()
	if !ok {
		return 0
	}
	z.mu.RLock()
	defer z.mu.RUnlock()
	return len(z.members)
}
