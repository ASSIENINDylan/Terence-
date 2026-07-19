package zone

import (
	"sync"
	"testing"
	"time"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
)

// nopSender ignore les messages (pour les tests qui ne vérifient pas l'envoi).
type nopSender struct{}

func (nopSender) SendEnvelope(string, uint64, any) {}

// capSender capture les zone.delta reçus, pour vérifier ce qui est diffusé.
type capSender struct {
	mu     sync.Mutex
	deltas []DeltaData
}

func newCapSender() *capSender { return &capSender{} }

func (c *capSender) SendEnvelope(msgType string, _ uint64, data any) {
	if msgType != protocol.TypeZoneDelta {
		return
	}
	if d, ok := data.(DeltaData); ok {
		c.mu.Lock()
		c.deltas = append(c.deltas, d)
		c.mu.Unlock()
	}
}

func (c *capSender) lastMovedX(id string) (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	x, ok := 0, false
	for _, d := range c.deltas {
		for _, mv := range d.Moved {
			if mv.CharacterID == id {
				x, ok = mv.X, true
			}
		}
	}
	return x, ok
}

func (c *capSender) hasJoined(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, d := range c.deltas {
		for _, e := range d.Joined {
			if e.CharacterID == id {
				return true
			}
		}
	}
	return false
}

func (c *capSender) hasLeft(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, d := range c.deltas {
		for _, left := range d.Left {
			if left == id {
				return true
			}
		}
	}
	return false
}

func waitFor(d time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}

func char(id string, zoneID int) domain.Character {
	return domain.Character{ID: id, Name: id, FactionID: 1, Level: 1, HP: 100, MaxHP: 100, ZoneID: zoneID}
}

func containsID(entities []EntityState, id string) bool {
	for _, e := range entities {
		if e.CharacterID == id {
			return true
		}
	}
	return false
}

// ── Entrée / snapshot / isolation (hérité de T3) ────────────────────────────

func TestEnterReturnsSelfInSnapshot(t *testing.T) {
	m := NewManager(20)
	defer m.Close()
	snap := m.Enter(char("a", 1), nopSender{})

	if snap.ZoneID != 1 || snap.Self != "a" {
		t.Fatalf("snapshot inattendu: %+v", snap)
	}
	if len(snap.Entities) != 1 || !containsID(snap.Entities, "a") {
		t.Fatalf("le joueur qui entre doit se voir : %+v", snap.Entities)
	}
}

func TestSecondEntrantSeesBoth(t *testing.T) {
	m := NewManager(20)
	defer m.Close()
	m.Enter(char("a", 1), nopSender{})
	snap := m.Enter(char("b", 1), nopSender{})

	if len(snap.Entities) != 2 || !containsID(snap.Entities, "a") || !containsID(snap.Entities, "b") {
		t.Fatalf("le 2e entrant doit voir a et b : %+v", snap.Entities)
	}
	if m.Count(1) != 2 {
		t.Fatalf("zone 1 devrait contenir 2 présences, obtenu %d", m.Count(1))
	}
}

func TestZonesAreIsolated(t *testing.T) {
	m := NewManager(20)
	defer m.Close()
	m.Enter(char("a", 1), nopSender{})
	snap := m.Enter(char("b", 2), nopSender{})

	if len(snap.Entities) != 1 || !containsID(snap.Entities, "b") {
		t.Fatalf("b (zone 2) ne doit pas voir a (zone 1) : %+v", snap.Entities)
	}
	if m.Count(1) != 1 || m.Count(2) != 1 {
		t.Fatalf("comptes de zone incohérents: z1=%d z2=%d", m.Count(1), m.Count(2))
	}
}

func TestLeaveRemovesPresence(t *testing.T) {
	m := NewManager(20)
	defer m.Close()
	a := char("a", 1)
	m.Enter(a, nopSender{})
	m.Enter(char("b", 1), nopSender{})

	m.Leave(a)
	if m.Count(1) != 1 {
		t.Fatalf("après le départ de a, zone 1 devrait contenir 1 présence, obtenu %d", m.Count(1))
	}

	snap := m.Enter(char("c", 1), nopSender{})
	if containsID(snap.Entities, "a") {
		t.Fatalf("a ne devrait plus apparaître après son départ : %+v", snap.Entities)
	}

	// Leave est idempotent.
	m.Leave(a)
	m.Leave(char("inconnu", 99))
}

// ── Ticks & deltas (T4) ─────────────────────────────────────────────────────

func TestMoveProducesDelta(t *testing.T) {
	m := NewManager(50) // ticks ~20 ms
	defer m.Close()
	s := newCapSender()
	m.Enter(char("a", 1), s) // spawn (0,0)

	m.Move(char("a", 1), Intent{DX: 1, DY: 0})

	ok := waitFor(time.Second, func() bool {
		x, moved := s.lastMovedX("a")
		return moved && x > 0
	})
	if !ok {
		t.Fatalf("le personnage aurait dû se déplacer en x>0")
	}
	m.Move(char("a", 1), Intent{DX: 0, DY: 0}) // stop
}

func TestMoveIsClampedToDirection(t *testing.T) {
	m := NewManager(50)
	defer m.Close()
	s := newCapSender()
	m.Enter(char("a", 1), s)

	// Un client malveillant tente une amplitude énorme : bornée à ±1 par axe.
	m.Move(char("a", 1), Intent{DX: 9999, DY: 0})
	time.Sleep(60 * time.Millisecond) // ~3 ticks
	m.Move(char("a", 1), Intent{DX: 0, DY: 0})

	x, moved := s.lastMovedX("a")
	if !moved {
		t.Fatal("aucun déplacement capturé")
	}
	// À 50 Hz, le pas est de 1 unité/tick ; en ~3 ticks on reste très en deçà
	// de la borne du monde. L'amplitude 9999 n'a donc PAS été appliquée.
	if x > 10*worldBound {
		t.Fatalf("l'amplitude du client n'aurait pas dû être appliquée, x=%d", x)
	}
}

func TestJoinNotifiesExistingPlayers(t *testing.T) {
	m := NewManager(50)
	defer m.Close()
	sa := newCapSender()
	m.Enter(char("a", 1), sa)
	m.Enter(char("b", 1), newCapSender())

	// a (déjà présent) doit apprendre l'arrivée de b via un zone.delta.
	if !waitFor(time.Second, func() bool { return sa.hasJoined("b") }) {
		t.Fatal("a aurait dû recevoir l'arrivée de b (delta joined)")
	}
}

func TestLeaveNotifiesRemainingPlayers(t *testing.T) {
	m := NewManager(50)
	defer m.Close()
	sa := newCapSender()
	m.Enter(char("a", 1), sa)
	b := char("b", 1)
	m.Enter(b, newCapSender())

	m.Leave(b)
	if !waitFor(time.Second, func() bool { return sa.hasLeft("b") }) {
		t.Fatal("a aurait dû recevoir le départ de b (delta left)")
	}
}
