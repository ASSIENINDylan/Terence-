package zone

import (
	"sync"
	"testing"
	"time"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
)

// nopSender ignore les messages.
type nopSender struct{}

func (nopSender) SendEnvelope(string, uint64, any) {}

// capSender capture tous les messages reçus, pour vérifier ce qui est diffusé.
type capEntry struct {
	typ  string
	data any
}
type capSender struct {
	mu sync.Mutex
	e  []capEntry
}

func newCapSender() *capSender { return &capSender{} }

func (c *capSender) SendEnvelope(t string, _ uint64, d any) {
	c.mu.Lock()
	c.e = append(c.e, capEntry{t, d})
	c.mu.Unlock()
}

func (c *capSender) each(fn func(capEntry)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.e {
		fn(e)
	}
}

func (c *capSender) has(typ string) bool {
	found := false
	c.each(func(e capEntry) {
		if e.typ == typ {
			found = true
		}
	})
	return found
}

func (c *capSender) lastMovedX(id string) (int, bool) {
	x, ok := 0, false
	c.each(func(e capEntry) {
		if d, is := e.data.(DeltaData); is {
			for _, mv := range d.Moved {
				if mv.CharacterID == id {
					x, ok = mv.X, true
				}
			}
		}
	})
	return x, ok
}

func (c *capSender) hasJoined(id string) bool {
	found := false
	c.each(func(e capEntry) {
		if d, is := e.data.(DeltaData); is {
			for _, j := range d.Joined {
				if j.CharacterID == id {
					found = true
				}
			}
		}
	})
	return found
}

func (c *capSender) hasLeft(id string) bool {
	found := false
	c.each(func(e capEntry) {
		if d, is := e.data.(DeltaData); is {
			for _, l := range d.Left {
				if l == id {
					found = true
				}
			}
		}
	})
	return found
}

func (c *capSender) combatEnd() (CombatEndData, bool) {
	var end CombatEndData
	ok := false
	c.each(func(e capEntry) {
		if d, is := e.data.(CombatEndData); is {
			end, ok = d, true
		}
	})
	return end, ok
}

func (c *capSender) currentTurn() string {
	turn := ""
	c.each(func(e capEntry) {
		switch d := e.data.(type) {
		case CombatStartData:
			if d.Turn != "" {
				turn = d.Turn
			}
		case CombatEventData:
			if d.Turn != "" {
				turn = d.Turn
			}
		}
	})
	return turn
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

func char(id string, zoneID int) domain.Character { return charF(id, zoneID, 1) }

func charF(id string, zoneID, faction int) domain.Character {
	return domain.Character{
		ID: id, Name: id, FactionID: faction, Level: 1,
		HP: 100, MaxHP: 100, Str: 10, Def: 10, Agi: 10, ZoneID: zoneID,
	}
}

func containsID(entities []EntityState, id string) bool {
	for _, e := range entities {
		if e.CharacterID == id {
			return true
		}
	}
	return false
}

// ── Entrée / snapshot / isolation (T3) ──────────────────────────────────────

func TestEnterReturnsSelfInSnapshot(t *testing.T) {
	m := NewManager(20, nil)
	defer m.Close()
	snap := m.Enter(char("a", 1), nopSender{})
	if snap.ZoneID != 1 || snap.Self != "a" || len(snap.Entities) != 1 || !containsID(snap.Entities, "a") {
		t.Fatalf("le joueur qui entre doit se voir : %+v", snap)
	}
}

func TestSecondEntrantSeesBoth(t *testing.T) {
	m := NewManager(20, nil)
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
	m := NewManager(20, nil)
	defer m.Close()
	m.Enter(char("a", 1), nopSender{})
	snap := m.Enter(char("b", 2), nopSender{})
	if len(snap.Entities) != 1 || !containsID(snap.Entities, "b") {
		t.Fatalf("b (zone 2) ne doit pas voir a (zone 1) : %+v", snap.Entities)
	}
}

func TestLeaveRemovesPresence(t *testing.T) {
	m := NewManager(20, nil)
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
	m.Leave(a) // idempotent
	m.Leave(char("inconnu", 99))
}

// ── Ticks & deltas (T4) ─────────────────────────────────────────────────────

func TestMoveProducesDelta(t *testing.T) {
	m := NewManager(50, nil)
	defer m.Close()
	s := newCapSender()
	m.Enter(char("a", 1), s)
	m.Move(char("a", 1), Intent{DX: 1, DY: 0})
	if !waitFor(time.Second, func() bool { x, ok := s.lastMovedX("a"); return ok && x > 0 }) {
		t.Fatal("le personnage aurait dû se déplacer en x>0")
	}
	m.Move(char("a", 1), Intent{DX: 0, DY: 0})
}

func TestJoinNotifiesExistingPlayers(t *testing.T) {
	m := NewManager(50, nil)
	defer m.Close()
	sa := newCapSender()
	m.Enter(char("a", 1), sa)
	m.Enter(char("b", 1), newCapSender())
	if !waitFor(time.Second, func() bool { return sa.hasJoined("b") }) {
		t.Fatal("a aurait dû recevoir l'arrivée de b (delta joined)")
	}
}

func TestLeaveNotifiesRemainingPlayers(t *testing.T) {
	m := NewManager(50, nil)
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

// ── Combat au tour par tour (T5) ────────────────────────────────────────────

// pvpZone active le PvP sur la zone 1.
func pvpZone() map[int]bool { return map[int]bool{1: true} }

func TestEncounterStartsCombat(t *testing.T) {
	m := NewManager(50, pvpZone())
	defer m.Close()
	sa, sb := newCapSender(), newCapSender()
	m.Enter(charF("a", 1, 1), sa) // Lumière
	m.Enter(charF("b", 1, 2), sb) // Ombre, même position → rencontre
	if !waitFor(time.Second, func() bool { return sa.has(protocol.TypeCombatStart) && sb.has(protocol.TypeCombatStart) }) {
		t.Fatal("un combat aurait dû s'engager entre factions opposées")
	}
}

func TestNoCombatBetweenAllies(t *testing.T) {
	m := NewManager(50, pvpZone())
	defer m.Close()
	sa := newCapSender()
	m.Enter(charF("a", 1, 1), sa)
	m.Enter(charF("b", 1, 1), newCapSender()) // même faction
	time.Sleep(200 * time.Millisecond)
	if sa.has(protocol.TypeCombatStart) {
		t.Fatal("aucun combat ne doit s'engager entre alliés")
	}
}

func TestNoCombatInSafeZone(t *testing.T) {
	m := NewManager(50, nil) // aucune zone PvP
	defer m.Close()
	sa := newCapSender()
	m.Enter(charF("a", 1, 1), sa)
	m.Enter(charF("b", 1, 2), newCapSender()) // factions opposées mais zone sûre
	time.Sleep(200 * time.Millisecond)
	if sa.has(protocol.TypeCombatStart) {
		t.Fatal("aucun combat ne doit s'engager dans une zone sûre")
	}
}

// driveToEnd fait attaquer le joueur actif jusqu'à la fin du combat.
func driveToEnd(t *testing.T, m *Manager, s *capSender) CombatEndData {
	t.Helper()
	for i := 0; i < 300; i++ {
		if end, ok := s.combatEnd(); ok {
			return end
		}
		if turn := s.currentTurn(); turn != "" {
			m.CombatAction(char(turn, 1), "attack")
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatal("le combat ne s'est pas terminé")
	return CombatEndData{}
}

func TestCombatAttackToDeath(t *testing.T) {
	m := NewManager(50, pvpZone())
	defer m.Close()
	sa, sb := newCapSender(), newCapSender()
	m.Enter(charF("a", 1, 1), sa)
	m.Enter(charF("b", 1, 2), sb)
	if !waitFor(time.Second, func() bool { return sa.has(protocol.TypeCombatStart) }) {
		t.Fatal("combat non engagé")
	}

	end := driveToEnd(t, m, sa)
	if end.Reason != "death" {
		t.Fatalf("le combat aurait dû se terminer par une mort, reason=%q", end.Reason)
	}
	if end.Winner == "" || end.Loser == "" || end.Winner == end.Loser {
		t.Fatalf("vainqueur/perdant incohérents: %+v", end)
	}
	if end.Respawn == nil || end.Respawn.HP != 100 {
		t.Fatalf("le perdant aurait dû réapparaître en pleine santé: %+v", end.Respawn)
	}
	// Les deux personnages sont toujours dans la zone (le perdant a réapparu).
	if m.Count(1) != 2 {
		t.Fatalf("les deux combattants devraient rester dans la zone, obtenu %d", m.Count(1))
	}
}

func TestFleeIsProcessed(t *testing.T) {
	m := NewManager(50, pvpZone())
	defer m.Close()
	sa, sb := newCapSender(), newCapSender()
	m.Enter(charF("a", 1, 1), sa)
	m.Enter(charF("b", 1, 2), sb)
	if !waitFor(time.Second, func() bool { return sa.has(protocol.TypeCombatStart) }) {
		t.Fatal("combat non engagé")
	}
	turn := sa.currentTurn()
	m.CombatAction(char(turn, 1), "flee")

	// La fuite se traduit soit par une fin de combat (réussite), soit par un
	// événement de combat (échec, le tour passe) — dans les deux cas, traitée.
	ok := waitFor(time.Second, func() bool {
		if end, ended := sa.combatEnd(); ended && end.Reason == "flee" {
			return true
		}
		return sa.has(protocol.TypeCombatEvent)
	})
	if !ok {
		t.Fatal("la tentative de fuite aurait dû être traitée")
	}
}
