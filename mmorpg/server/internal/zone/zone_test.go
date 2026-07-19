package zone

import (
	"testing"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
)

// nopSender ignore les messages : la logique de zone testée ici ne dépend pas de
// l'envoi (celui-ci sera exercé par les deltas en T4).
type nopSender struct{}

func (nopSender) SendEnvelope(string, uint64, any) {}

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

func TestEnterReturnsSelfInSnapshot(t *testing.T) {
	m := NewManager()
	snap := m.Enter(char("a", 1), nopSender{})

	if snap.ZoneID != 1 || snap.Self != "a" {
		t.Fatalf("snapshot inattendu: %+v", snap)
	}
	if len(snap.Entities) != 1 || !containsID(snap.Entities, "a") {
		t.Fatalf("le joueur qui entre doit se voir : %+v", snap.Entities)
	}
}

func TestSecondEntrantSeesBoth(t *testing.T) {
	m := NewManager()
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
	m := NewManager()
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
	m := NewManager()
	a := char("a", 1)
	m.Enter(a, nopSender{})
	m.Enter(char("b", 1), nopSender{})

	m.Leave(a)
	if m.Count(1) != 1 {
		t.Fatalf("après le départ de a, zone 1 devrait contenir 1 présence, obtenu %d", m.Count(1))
	}

	// Un nouvel entrant ne doit plus voir a.
	snap := m.Enter(char("c", 1), nopSender{})
	if containsID(snap.Entities, "a") {
		t.Fatalf("a ne devrait plus apparaître après son départ : %+v", snap.Entities)
	}

	// Leave est idempotent.
	m.Leave(a)
	m.Leave(char("inconnu", 99))
}
