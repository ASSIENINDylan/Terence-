package zone

import (
	"fmt"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
	"github.com/assienindylan/terence-/mmorpg/server/internal/rules"
)

// combat est un affrontement 1v1 au tour par tour entre deux membres présents
// dans la zone. L'état est possédé par la goroutine de la zone (pas de verrou).
type combat struct {
	id       string
	a, b     string // identifiants des personnages
	turn     string // à qui de jouer
	deadline uint64 // tick au-delà duquel le tour expire (attaque automatique)
}

func (c *combat) other(id string) string {
	if id == c.a {
		return c.b
	}
	return c.a
}

// ── Charges utiles réseau (T5) ──────────────────────────────────────────────

// Combatant décrit un participant au début d'un combat.
type Combatant struct {
	CharacterID string  `json:"character_id"`
	Name        string  `json:"name"`
	FactionID   int     `json:"faction_id"`
	Level       int     `json:"level"`
	HP          int     `json:"hp"`
	MaxHP       int     `json:"max_hp"`
	Str         float64 `json:"str"`
	Def         float64 `json:"def"`
	Agi         float64 `json:"agi"`
}

func combatantOf(c domain.Character) Combatant {
	return Combatant{
		CharacterID: c.ID, Name: c.Name, FactionID: c.FactionID, Level: c.Level,
		HP: c.HP, MaxHP: c.MaxHP, Str: c.Str, Def: c.Def, Agi: c.Agi,
	}
}

// CombatStartData : un combat s'engage.
type CombatStartData struct {
	CombatID  string      `json:"combat_id"`
	Opponents []Combatant `json:"opponents"`
	Turn      string      `json:"turn"`    // à qui de jouer
	TurnMs    int         `json:"turn_ms"` // délai avant attaque automatique
}

// CombatEventData : résultat d'une action (attaque, technique ou fuite).
type CombatEventData struct {
	CombatID    string `json:"combat_id"`
	Actor       string `json:"actor"`
	Action      string `json:"action"` // "attack" | "technique" | "flee"
	Tech        string `json:"tech,omitempty"`
	Target      string `json:"target,omitempty"`
	Damage      int    `json:"damage,omitempty"`
	TargetHP    int    `json:"target_hp,omitempty"`
	ActorEnergy int    `json:"actor_energy,omitempty"`
	Fled        bool   `json:"fled,omitempty"`
	Turn        string `json:"turn,omitempty"` // à qui de jouer ensuite ("" si terminé)
}

// RespawnInfo décrit la réapparition d'un personnage mort (au village).
type RespawnInfo struct {
	CharacterID string `json:"character_id"`
	ZoneID      int    `json:"zone_id"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	HP          int    `json:"hp"`
	XP          int64  `json:"xp"`
}

// CombatEndData : fin du combat.
type CombatEndData struct {
	CombatID     string       `json:"combat_id"`
	Winner       string       `json:"winner,omitempty"`
	Loser        string       `json:"loser,omitempty"`
	Reason       string       `json:"reason"` // "death" | "flee" | "opponent_left"
	Respawn      *RespawnInfo `json:"respawn,omitempty"`
	XPLost       int64        `json:"xp_lost,omitempty"`
	WinnerXPGain int64        `json:"winner_xp_gain,omitempty"`
}

// ── Détection des rencontres ────────────────────────────────────────────────

// detectEncounters engage un combat entre deux membres libres de factions
// opposées suffisamment proches, dans une zone où le PvP est autorisé.
func (z *Zone) detectEncounters() {
	if !z.meta.PvP {
		return
	}
	var free []*member
	for _, m := range z.members {
		if !m.inCombat() && z.tick >= m.cooldownUntil {
			free = append(free, m)
		}
	}
	for i := 0; i < len(free); i++ {
		for j := i + 1; j < len(free); j++ {
			a, b := free[i], free[j]
			if a.inCombat() || b.inCombat() { // engagé dans ce même passage
				continue
			}
			if a.char.FactionID == b.char.FactionID {
				continue // alliés
			}
			if dist2(a.char, b.char) <= encounterRadius*encounterRadius {
				z.startCombat(a, b)
			}
		}
	}
}

func (z *Zone) startCombat(a, b *member) {
	z.combatN++
	id := fmt.Sprintf("cb-%d-%d", z.meta.ID, z.combatN)
	a.combatID, b.combatID = id, id
	a.intent, b.intent = Intent{}, Intent{} // le mouvement s'arrête

	first := initiative(a.char, b.char)
	c := &combat{id: id, a: a.char.ID, b: b.char.ID, turn: first, deadline: z.tick + z.turnTimeoutTicks}
	z.combats[id] = c

	start := CombatStartData{
		CombatID:  id,
		Opponents: []Combatant{combatantOf(a.char), combatantOf(b.char)},
		Turn:      first,
		TurnMs:    turnTimeoutSec * 1000,
	}
	z.sendBoth(c, protocol.TypeCombatStart, start)
}

// ── Résolution des tours ────────────────────────────────────────────────────

func (z *Zone) resolveTimedOutTurns() {
	var expired []*combat
	for _, c := range z.combats {
		if z.tick >= c.deadline {
			expired = append(expired, c)
		}
	}
	for _, c := range expired {
		if _, ok := z.combats[c.id]; ok {
			z.resolveAction(c, c.turn, "attack", "") // attaque par défaut
		}
	}
}

// resolveAction applique l'action du joueur actif et fait avancer le combat.
// action ∈ {"attack", "technique", "flee"} ; techID est requis pour "technique".
func (z *Zone) resolveAction(c *combat, actorID, action, techID string) {
	actor := z.members[actorID]
	opp := z.members[c.other(actorID)]
	if actor == nil || opp == nil {
		z.clearCombat(c) // un participant a disparu : on nettoie
		return
	}

	if action == "flee" {
		if z.rng.Float64() < rules.FleeChance(actor.char, opp.char) {
			z.sendBoth(c, protocol.TypeCombatEvent, CombatEventData{
				CombatID: c.id, Actor: actorID, Action: "flee", Fled: true,
			})
			z.endCombat(c, "", "", "flee", nil)
			return
		}
		c.turn = opp.char.ID
		c.deadline = z.tick + z.turnTimeoutTicks
		z.sendBoth(c, protocol.TypeCombatEvent, CombatEventData{
			CombatID: c.id, Actor: actorID, Action: "flee", Fled: false, Turn: c.turn,
		})
		return
	}

	ev := CombatEventData{CombatID: c.id, Actor: actorID, Target: opp.char.ID}

	if action == "technique" {
		t, ok := rules.FindTech(actor.char.Element, techID)
		if !ok || !actor.char.KnowsTech(techID) || actor.char.Energy < t.Cost {
			// Action invalide : on renvoie une erreur et on ne consomme pas le tour.
			actor.client.SendEnvelope(protocol.TypeError, 0, protocol.ErrorData{
				Code: "invalid_technique", Message: "technique indisponible ou énergie insuffisante",
			})
			return
		}
		actor.char.Energy -= t.Cost
		dmg := rules.TechDamage(actor.char, t, opp.char)
		opp.char.HP = maxi(0, opp.char.HP-dmg)
		ev.Action = "technique"
		ev.Tech = t.Name
		ev.Damage = dmg
		ev.ActorEnergy = actor.char.Energy
		z.sendCharUpdate(actor)
	} else { // attaque normale
		dmg := rules.Damage(actor.char, opp.char)
		opp.char.HP = maxi(0, opp.char.HP-dmg)
		ev.Action = "attack"
		ev.Damage = dmg
	}
	ev.TargetHP = opp.char.HP

	if opp.char.HP == 0 { // mort
		z.sendBoth(c, protocol.TypeCombatEvent, ev)
		z.handleDeath(c, actorID, opp.char.ID)
		return
	}
	// Tour suivant : l'adversaire regagne de l'énergie.
	c.turn = opp.char.ID
	c.deadline = z.tick + z.turnTimeoutTicks
	opp.char.Energy = mini(opp.char.MaxEnergy, opp.char.Energy+rules.EnergyRegenPerTurn)
	z.sendCharUpdate(opp)
	ev.Turn = c.turn
	z.sendBoth(c, protocol.TypeCombatEvent, ev)
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func mini(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleDeath applique la pénalité d'XP selon le palier de la zone, récompense
// le vainqueur, clôt le combat et relocalise le perdant vers son village.
func (z *Zone) handleDeath(c *combat, winnerID, loserID string) {
	loser := z.members[loserID]
	winner := z.members[winnerID]

	var xpLost, newXP, winGain int64
	if loser != nil {
		xpLost = int64(float64(loser.char.XP) * z.meta.DeathXPLoss)
		newXP = loser.char.XP - xpLost
		if newXP < 0 {
			newXP = 0
		}
	}
	if winner != nil {
		winGain = rules.KillXPReward
		winner.char.Gold += rules.KillGoldReward
		rules.ApplyXP(&winner.char, winGain) // montée de niveau (points d'attribut/technique)
		// Série de kills sans quitter la zone → point parfait (orange/red).
		if thr := rules.PerfectKillThreshold(z.meta.Tier); thr > 0 {
			winner.streak++
			if winner.streak >= thr {
				winner.streak = 0
				winner.char.PerfectPoints++
			}
		}
		z.sendCharUpdate(winner)
	}

	homeZone := z.meta.ID // repli défensif si le village est inconnu
	if loser != nil && loser.char.HomeZoneID != 0 {
		homeZone = loser.char.HomeZoneID
	}
	respawn := &RespawnInfo{CharacterID: loserID, ZoneID: homeZone, X: 0, Y: 0, HP: 0, XP: newXP}
	if loser != nil {
		respawn.HP = loser.char.MaxHP
	}
	z.sendBoth(c, protocol.TypeCombatEnd, CombatEndData{
		CombatID: c.id, Winner: winnerID, Loser: loserID, Reason: "death",
		Respawn: respawn, XPLost: xpLost, WinnerXPGain: winGain,
	})

	z.setCooldown(winnerID)
	z.clearCombat(c)

	// Relocalise le perdant vers son village (hors de cette zone).
	if loser != nil {
		relocated := loser.char
		relocated.HP = loser.char.MaxHP
		relocated.XP = newXP
		relocated.ZoneID = homeZone
		relocated.X, relocated.Y = 0, 0
		delete(z.members, loserID)
		z.left = append(z.left, loserID)
		loser.client.Relocate(relocated)
	}
}

// endCombat clôt un combat sans mort (fuite), avec immunité post-combat.
func (z *Zone) endCombat(c *combat, winner, loser, reason string, respawn *RespawnInfo) {
	end := CombatEndData{CombatID: c.id, Winner: winner, Loser: loser, Reason: reason, Respawn: respawn}
	z.sendBoth(c, protocol.TypeCombatEnd, end)
	z.setCooldown(c.a)
	z.setCooldown(c.b)
	z.clearCombat(c)
}

// endByForfeit clôt un combat dont un participant s'est déconnecté.
func (z *Zone) endByForfeit(c *combat, quitterID string) {
	winnerID := c.other(quitterID)
	if w := z.members[winnerID]; w != nil {
		w.client.SendEnvelope(protocol.TypeCombatEnd, 0, CombatEndData{
			CombatID: c.id, Winner: winnerID, Loser: quitterID, Reason: "opponent_left",
		})
		w.cooldownUntil = z.tick + z.cooldownTicks
		w.combatID = ""
	}
	delete(z.combats, c.id)
}

func (z *Zone) clearCombat(c *combat) {
	if m := z.members[c.a]; m != nil {
		m.combatID = ""
	}
	if m := z.members[c.b]; m != nil {
		m.combatID = ""
	}
	delete(z.combats, c.id)
}

func (z *Zone) setCooldown(id string) {
	if m := z.members[id]; m != nil {
		m.cooldownUntil = z.tick + z.cooldownTicks
	}
}

// sendBoth envoie un message aux deux participants encore présents.
func (z *Zone) sendBoth(c *combat, msgType string, data any) {
	if m := z.members[c.a]; m != nil {
		m.client.SendEnvelope(msgType, 0, data)
	}
	if m := z.members[c.b]; m != nil {
		m.client.SendEnvelope(msgType, 0, data)
	}
}

// ── Règles de combat ────────────────────────────────────────────────────────

// initiative : qui joue en premier — le plus agile, à égalité le plus petit id
// (déterministe).
func initiative(a, b domain.Character) string {
	if a.Agi != b.Agi {
		if a.Agi > b.Agi {
			return a.ID
		}
		return b.ID
	}
	if a.ID < b.ID {
		return a.ID
	}
	return b.ID
}

func dist2(a, b domain.Character) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx + dy*dy
}
