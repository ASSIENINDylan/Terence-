package zone

import (
	"sync"
	"testing"
	"time"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
)

// nopSender ignore les messages et les relocalisations.
type nopSender struct{}

func (nopSender) SendEnvelope(string, uint64, any) {}
func (nopSender) Relocate(domain.Character)        {}

// capSender capture les messages reçus et les relocalisations demandées.
type capEntry struct {
	typ  string
	data any
}
type capSender struct {
	mu        sync.Mutex
	e         []capEntry
	relocated *domain.Character
}

func newCapSender() *capSender { return &capSender{} }

func (c *capSender) SendEnvelope(t string, _ uint64, d any) {
	c.mu.Lock()
	c.e = append(c.e, capEntry{t, d})
	c.mu.Unlock()
}

func (c *capSender) Relocate(ch domain.Character) {
	c.mu.Lock()
	c.relocated = &ch
	c.mu.Unlock()
}

func (c *capSender) lastRelocate() (domain.Character, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.relocated == nil {
		return domain.Character{}, false
	}
	return *c.relocated, true
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

// fighter est un combattant avec un village de départ et de l'XP (pour tester la
// mort : réapparition + pénalité d'XP).
func fighter(id string, faction, homeZone int, xp int64) domain.Character {
	c := charF(id, 1, faction)
	c.HomeZoneID = homeZone
	c.XP = xp
	return c
}

func containsID(entities []EntityState, id string) bool {
	for _, e := range entities {
		if e.CharacterID == id {
			return true
		}
	}
	return false
}

// pvpMetas décrit la zone 1 comme une zone orange (PvP, −½ XP à la mort).
func pvpMetas() map[int]ZoneMeta {
	return map[int]ZoneMeta{1: {ID: 1, Name: "Marches", Tier: "orange", PvP: true, DeathXPLoss: 0.5}}
}

// ── Entrée / snapshot / isolation (T3) ──────────────────────────────────────

func TestEnterReturnsSelfInSnapshot(t *testing.T) {
	m := NewManager(20, nil, nil)
	defer m.Close()
	snap := m.Enter(char("a", 1), nopSender{})
	if snap.ZoneID != 1 || snap.Self != "a" || len(snap.Entities) != 1 || !containsID(snap.Entities, "a") {
		t.Fatalf("le joueur qui entre doit se voir : %+v", snap)
	}
}

func TestSecondEntrantSeesBoth(t *testing.T) {
	m := NewManager(20, nil, nil)
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
	m := NewManager(20, nil, nil)
	defer m.Close()
	m.Enter(char("a", 1), nopSender{})
	snap := m.Enter(char("b", 2), nopSender{})
	if len(snap.Entities) != 1 || !containsID(snap.Entities, "b") {
		t.Fatalf("b (zone 2) ne doit pas voir a (zone 1) : %+v", snap.Entities)
	}
}

func TestLeaveRemovesPresence(t *testing.T) {
	m := NewManager(20, nil, nil)
	defer m.Close()
	a := char("a", 1)
	m.Enter(a, nopSender{})
	m.Enter(char("b", 1), nopSender{})
	m.Leave(a)
	if m.Count(1) != 1 {
		t.Fatalf("après le départ de a, zone 1 devrait contenir 1 présence, obtenu %d", m.Count(1))
	}
	m.Leave(a) // idempotent
	m.Leave(char("inconnu", 99))
}

// ── Ticks & deltas (T4) ─────────────────────────────────────────────────────

func TestMoveProducesDelta(t *testing.T) {
	m := NewManager(50, nil, nil)
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
	m := NewManager(50, nil, nil)
	defer m.Close()
	sa := newCapSender()
	m.Enter(char("a", 1), sa)
	m.Enter(char("b", 1), newCapSender())
	if !waitFor(time.Second, func() bool { return sa.hasJoined("b") }) {
		t.Fatal("a aurait dû recevoir l'arrivée de b (delta joined)")
	}
}

func TestLeaveNotifiesRemainingPlayers(t *testing.T) {
	m := NewManager(50, nil, nil)
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

// ── Transitions de zone (portails / barrières) ──────────────────────────────

func TestTransitionMovesPlayerToLinkedZone(t *testing.T) {
	metas := map[int]ZoneMeta{
		1: {ID: 1, Name: "Village", Tier: "village"},
		2: {ID: 2, Name: "Vert", Tier: "green"},
	}
	links := map[int]Link{
		7: {ID: 7, FromZoneID: 1, ToZoneID: 2, Kind: "portal", FromX: 0, FromY: 0, ToX: 5, ToY: 5},
	}
	m := NewManager(50, metas, links)
	defer m.Close()

	c := char("a", 1)
	m.Enter(c, nopSender{})
	newC, snap, err := m.Transition(c, 7, nopSender{})
	if err != nil {
		t.Fatalf("transition refusée: %v", err)
	}
	if newC.ZoneID != 2 || newC.X != 5 || newC.Y != 5 {
		t.Fatalf("le personnage aurait dû arriver en zone 2 (5,5) : %+v", newC)
	}
	if snap.ZoneID != 2 || snap.ZoneName != "Vert" {
		t.Fatalf("snapshot de destination incorrect : %+v", snap)
	}
	if m.Count(1) != 0 || m.Count(2) != 1 {
		t.Fatalf("présences après transition incohérentes: z1=%d z2=%d", m.Count(1), m.Count(2))
	}
}

func TestTransitionRejectedWhenTooFar(t *testing.T) {
	metas := map[int]ZoneMeta{1: {ID: 1}, 2: {ID: 2}}
	links := map[int]Link{7: {ID: 7, FromZoneID: 1, ToZoneID: 2, Kind: "portal", FromX: 0, FromY: 0, ToX: 0, ToY: 0}}
	m := NewManager(50, metas, links)
	defer m.Close()

	c := char("a", 1)
	c.X, c.Y = 200, 0 // loin du portail
	m.Enter(c, nopSender{})
	if _, _, err := m.Transition(c, 7, nopSender{}); err == nil {
		t.Fatal("la transition aurait dû être refusée (trop loin du portail)")
	}
	if _, _, err := m.Transition(char("a", 1), 999, nopSender{}); err == nil {
		t.Fatal("un lien inconnu aurait dû être refusé")
	}
}

// ── Combat au tour par tour (T5) ────────────────────────────────────────────

func TestEncounterStartsCombat(t *testing.T) {
	m := NewManager(50, pvpMetas(), nil)
	defer m.Close()
	sa, sb := newCapSender(), newCapSender()
	m.Enter(fighter("a", 1, 9, 0), sa)
	m.Enter(fighter("b", 2, 9, 0), sb)
	if !waitFor(time.Second, func() bool { return sa.has(protocol.TypeCombatStart) && sb.has(protocol.TypeCombatStart) }) {
		t.Fatal("un combat aurait dû s'engager entre factions opposées")
	}
}

func TestNoCombatBetweenAllies(t *testing.T) {
	m := NewManager(50, pvpMetas(), nil)
	defer m.Close()
	sa := newCapSender()
	m.Enter(fighter("a", 1, 9, 0), sa)
	m.Enter(fighter("b", 1, 9, 0), newCapSender())
	time.Sleep(200 * time.Millisecond)
	if sa.has(protocol.TypeCombatStart) {
		t.Fatal("aucun combat ne doit s'engager entre alliés")
	}
}

func TestNoCombatInSafeZone(t *testing.T) {
	m := NewManager(50, nil, nil) // zone 1 par défaut : verte, pas de PvP
	defer m.Close()
	sa := newCapSender()
	m.Enter(fighter("a", 1, 9, 0), sa)
	m.Enter(fighter("b", 2, 9, 0), newCapSender())
	time.Sleep(200 * time.Millisecond)
	if sa.has(protocol.TypeCombatStart) {
		t.Fatal("aucun combat ne doit s'engager hors zone PvP")
	}
}

func driveToEnd(t *testing.T, m *Manager, s *capSender) CombatEndData {
	t.Helper()
	for i := 0; i < 300; i++ {
		if end, ok := s.combatEnd(); ok {
			return end
		}
		if turn := s.currentTurn(); turn != "" {
			m.CombatAction(char(turn, 1), "attack", "")
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatal("le combat ne s'est pas terminé")
	return CombatEndData{}
}

func TestCombatDeathRespawnsAtVillageWithXPPenalty(t *testing.T) {
	m := NewManager(50, pvpMetas(), nil) // orange : −½ XP
	defer m.Close()
	sa, sb := newCapSender(), newCapSender()
	m.Enter(fighter("a", 1, 9, 100), sa) // village = zone 9, 100 XP
	m.Enter(fighter("b", 2, 9, 100), sb)
	if !waitFor(time.Second, func() bool { return sa.has(protocol.TypeCombatStart) }) {
		t.Fatal("combat non engagé")
	}

	end := driveToEnd(t, m, sa)
	if end.Reason != "death" || end.Winner == "" || end.Loser == "" {
		t.Fatalf("le combat aurait dû se terminer par une mort : %+v", end)
	}
	if end.Respawn == nil || end.Respawn.ZoneID != 9 || end.Respawn.HP != 100 {
		t.Fatalf("le perdant aurait dû réapparaître au village (zone 9) en pleine santé : %+v", end.Respawn)
	}
	// Orange : le perdant perd la moitié de ses 100 XP → il lui en reste 50.
	if end.XPLost != 50 || end.Respawn.XP != 50 {
		t.Fatalf("pénalité d'XP incorrecte (attendu −50 → 50) : lost=%d xp=%d", end.XPLost, end.Respawn.XP)
	}
	// Le vainqueur reste, le perdant a quitté la zone (relocalisé au village).
	if m.Count(1) != 1 {
		t.Fatalf("seul le vainqueur devrait rester dans la zone, obtenu %d", m.Count(1))
	}
	loserSender := sa
	if end.Loser != "a" {
		loserSender = sb
	}
	if rc, ok := loserSender.lastRelocate(); !ok || rc.ZoneID != 9 {
		t.Fatalf("le perdant aurait dû être relocalisé au village (zone 9), obtenu %+v (ok=%v)", rc, ok)
	}
}

func TestRedZoneWipesAllXP(t *testing.T) {
	metas := map[int]ZoneMeta{1: {ID: 1, Tier: "red", PvP: true, DeathXPLoss: 1.0}}
	m := NewManager(50, metas, nil)
	defer m.Close()
	sa, sb := newCapSender(), newCapSender()
	m.Enter(fighter("a", 1, 9, 200), sa)
	m.Enter(fighter("b", 2, 9, 200), sb)
	if !waitFor(time.Second, func() bool { return sa.has(protocol.TypeCombatStart) }) {
		t.Fatal("combat non engagé")
	}
	end := driveToEnd(t, m, sa)
	if end.Respawn == nil || end.Respawn.XP != 0 || end.XPLost != 200 {
		t.Fatalf("en zone rouge, toute l'XP devrait être perdue : lost=%d xp=%d", end.XPLost, end.Respawn.XP)
	}
}

func TestFleeIsProcessed(t *testing.T) {
	m := NewManager(50, pvpMetas(), nil)
	defer m.Close()
	sa, sb := newCapSender(), newCapSender()
	m.Enter(fighter("a", 1, 9, 0), sa)
	m.Enter(fighter("b", 2, 9, 0), sb)
	if !waitFor(time.Second, func() bool { return sa.has(protocol.TypeCombatStart) }) {
		t.Fatal("combat non engagé")
	}
	turn := sa.currentTurn()
	m.CombatAction(char(turn, 1), "flee", "")
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

// ── Mobs PvE ────────────────────────────────────────────────────────────────

func mobMetas(tier string, xpLoss float64) map[int]ZoneMeta {
	return map[int]ZoneMeta{1: {ID: 1, Name: "Zone", Tier: tier, PvP: tier == "orange" || tier == "red", MobTier: tier, DeathXPLoss: xpLoss}}
}

// forcePlayerOntoMob téléporte un joueur sur un mob pour déclencher la rencontre
// (exécuté dans la goroutine de la zone : sûr).
func (m *Manager) forcePlayerOntoMob(playerID string, zoneID int) {
	z := m.lookup(zoneID)
	if z == nil {
		return
	}
	done := make(chan struct{})
	z.cmds <- func() {
		p := z.members[playerID]
		for _, mob := range z.members {
			if mob.isMob {
				p.char.X, p.char.Y = mob.char.X, mob.char.Y
				break
			}
		}
		close(done)
	}
	<-done
}

func (c *capSender) lastCharState() (CharState, bool) {
	var cs CharState
	ok := false
	c.each(func(e capEntry) {
		if d, is := e.data.(CharState); is {
			cs, ok = d, true
		}
	})
	return cs, ok
}

func hero(id string) domain.Character {
	return domain.Character{ID: id, Name: id, Level: 1, HP: 100, MaxHP: 100,
		Str: 30, Def: 20, Agi: 20, ZoneID: 1, HomeZoneID: 9, Element: "feu", SpecStat: "str", MaxEnergy: 50, Energy: 50}
}

func TestMobsSpawnInMobZone(t *testing.T) {
	m := NewManager(50, mobMetas("green", 0), nil)
	defer m.Close()
	// Les zones sont créées à la demande : l'entrée du joueur instancie la zone
	// verte, qui fait apparaître ses 3 mobs. On compte donc 3 mobs + 1 joueur.
	m.Enter(hero("p"), newCapSender())
	if got := m.Count(1); got != 4 {
		t.Fatalf("la zone verte devrait contenir 3 mobs + le joueur, obtenu %d", got)
	}
}

func TestPlayerKillsMobAndIsRewarded(t *testing.T) {
	m := NewManager(50, mobMetas("green", 0), nil)
	defer m.Close()
	s := newCapSender()
	m.Enter(hero("p"), s)
	m.forcePlayerOntoMob("p", 1)

	if !waitFor(2*time.Second, func() bool { return s.has(protocol.TypeCombatStart) }) {
		t.Fatal("un combat PvE aurait dû s'engager avec un mob")
	}
	// Le joueur attaque à son tour ; le mob agit via l'IA automatiquement.
	var end CombatEndData
	ok := false
	for i := 0; i < 300 && !ok; i++ {
		if e, has := s.combatEnd(); has {
			end, ok = e, true
			break
		}
		if s.currentTurn() == "p" {
			m.CombatAction(char("p", 1), "attack", "")
		}
		time.Sleep(15 * time.Millisecond)
	}
	if !ok || end.Winner != "p" {
		t.Fatalf("le héros aurait dû vaincre le mob : %+v (ok=%v)", end, ok)
	}
	// Récompense : de l'or (moins qu'un joueur) via char.update.
	cs, has := s.lastCharState()
	if !has || cs.Gold <= 0 {
		t.Fatalf("le joueur aurait dû gagner de l'or en tuant le mob : %+v", cs)
	}
}

// ── Inventaire (T6) ──────────────────────────────────────────────────────────

func lootCatalogue() *domain.Catalogue {
	return domain.NewCatalogue([]domain.ItemTemplate{
		{ID: 1, Code: "dague_usee", Name: "Dague usée", Type: domain.ItemWeapon, Stats: domain.ItemStats{Atk: 3}},
		{ID: 2, Code: "tunique_cuir", Name: "Tunique de cuir", Type: domain.ItemArmor, Stats: domain.ItemStats{Def: 3}},
		{ID: 3, Code: "potion_soin", Name: "Potion de soin", Type: domain.ItemConsumable, Stackable: true, MaxStack: 10, Stats: domain.ItemStats{Heal: 40}},
	})
}

// testDropCode injecte un objet au sol d'un code donné (exécuté dans l'acteur).
func (m *Manager) testDropCode(zoneID int, code string, x, y int) string {
	z := m.lookup(zoneID)
	reply := make(chan string, 1)
	z.cmds <- func() {
		t, ok := z.cat.ByCode(code)
		if !ok {
			reply <- ""
			return
		}
		g := &groundItem{id: newUUID(), templateID: t.ID, x: x, y: y}
		z.ground[g.id] = g
		reply <- g.id
	}
	return <-reply
}

// setHP force les PV d'un membre (exécuté dans l'acteur).
func (m *Manager) setHP(zoneID int, charID string, hp int) {
	z := m.lookup(zoneID)
	done := make(chan struct{})
	z.cmds <- func() {
		if mem := z.members[charID]; mem != nil {
			mem.char.HP = hp
		}
		close(done)
	}
	<-done
}

func (c *capSender) lastInventory() (InventoryData, bool) {
	var inv InventoryData
	ok := false
	c.each(func(e capEntry) {
		if d, is := e.data.(InventoryData); is {
			inv, ok = d, true
		}
	})
	return inv, ok
}

func TestPickupEquipUpdatesEffectiveStats(t *testing.T) {
	m := NewManager(50, mobMetas("green", 0), nil)
	m.SetCatalogue(lootCatalogue())
	defer m.Close()
	s := newCapSender()
	h := hero("p") // à (X,Y) aléatoire ? non : hero place ZoneID=1, X/Y=0.
	m.Enter(h, s)

	// Un butin (dague +3 atk) tombe à portée du joueur (0,0).
	itemID := m.testDropCode(1, "dague_usee", 0, 0)
	if itemID == "" {
		t.Fatal("échec de l'injection du butin")
	}

	// Ramasser : l'inventaire reçu contient la dague.
	m.PickupItem(char("p", 1), itemID)
	if !waitFor(time.Second, func() bool {
		inv, ok := s.lastInventory()
		return ok && len(inv.Items) == 1 && inv.Items[0].ID == itemID
	}) {
		t.Fatal("la dague ramassée devrait apparaître dans l'inventaire")
	}

	// Équiper : les statistiques effectives augmentent (bonus d'attaque).
	m.EquipItem(char("p", 1), itemID)
	if !waitFor(time.Second, func() bool {
		cs, ok := s.lastCharState()
		return ok && cs.StrBonus == 3
	}) {
		cs, _ := s.lastCharState()
		t.Fatalf("l'arme équipée devrait donner +3 d'attaque : %+v", cs)
	}
}

func TestPickupAndUseConsumableHeals(t *testing.T) {
	m := NewManager(50, mobMetas("green", 0), nil)
	m.SetCatalogue(lootCatalogue())
	defer m.Close()
	s := newCapSender()
	m.Enter(hero("p"), s)

	itemID := m.testDropCode(1, "potion_soin", 0, 0)
	m.PickupItem(char("p", 1), itemID)
	if !waitFor(time.Second, func() bool { _, ok := s.lastInventory(); return ok }) {
		t.Fatal("la potion devrait être ramassée")
	}
	// Blesser le joueur puis boire la potion (+40 PV).
	m.setHP(1, "p", 30)
	m.UseItem(char("p", 1), itemID)
	if !waitFor(time.Second, func() bool {
		cs, ok := s.lastCharState()
		return ok && cs.HP == 70
	}) {
		cs, _ := s.lastCharState()
		t.Fatalf("la potion aurait dû soigner jusqu'à 70 PV : %+v", cs)
	}
}
