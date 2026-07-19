package zone

import (
	"fmt"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
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
	CharacterID string `json:"character_id"`
	Name        string `json:"name"`
	FactionID   int    `json:"faction_id"`
	Level       int    `json:"level"`
	HP          int    `json:"hp"`
	MaxHP       int    `json:"max_hp"`
	Str         int    `json:"str"`
	Def         int    `json:"def"`
	Agi         int    `json:"agi"`
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

// CombatEventData : résultat d'une action (attaque ou tentative de fuite).
type CombatEventData struct {
	CombatID string `json:"combat_id"`
	Actor    string `json:"actor"`
	Action   string `json:"action"` // "attack" | "flee"
	Target   string `json:"target,omitempty"`
	Damage   int    `json:"damage,omitempty"`
	TargetHP int    `json:"target_hp,omitempty"`
	Fled     bool   `json:"fled,omitempty"`
	Turn     string `json:"turn,omitempty"` // à qui de jouer ensuite ("" si terminé)
}

// RespawnInfo décrit la réapparition d'un personnage mort.
type RespawnInfo struct {
	CharacterID string `json:"character_id"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	HP          int    `json:"hp"`
}

// CombatEndData : fin du combat.
type CombatEndData struct {
	CombatID string       `json:"combat_id"`
	Winner   string       `json:"winner,omitempty"`
	Loser    string       `json:"loser,omitempty"`
	Reason   string       `json:"reason"` // "death" | "flee" | "opponent_left"
	Respawn  *RespawnInfo `json:"respawn,omitempty"`
}

// ── Détection des rencontres ────────────────────────────────────────────────

// detectEncounters engage un combat entre deux membres libres de factions
// opposées suffisamment proches, dans une zone où le PvP est autorisé.
func (z *Zone) detectEncounters() {
	if !z.pvp {
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
	id := fmt.Sprintf("cb-%d-%d", z.id, z.combatN)
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
			z.resolveAction(c, c.turn, "attack") // le joueur passif attaque par défaut
		}
	}
}

// resolveAction applique l'action du joueur actif et fait avancer le combat.
func (z *Zone) resolveAction(c *combat, actorID, action string) {
	actor := z.members[actorID]
	opp := z.members[c.other(actorID)]
	if actor == nil || opp == nil {
		z.clearCombat(c) // un participant a disparu : on nettoie
		return
	}

	if action == "flee" {
		if z.fleeSucceeds(actor.char, opp.char) {
			z.sendBoth(c, protocol.TypeCombatEvent, CombatEventData{
				CombatID: c.id, Actor: actorID, Action: "flee", Fled: true,
			})
			z.endCombat(c, "", "", "flee", nil)
			return
		}
		// Échec : le tour passe à l'adversaire.
		c.turn = opp.char.ID
		c.deadline = z.tick + z.turnTimeoutTicks
		z.sendBoth(c, protocol.TypeCombatEvent, CombatEventData{
			CombatID: c.id, Actor: actorID, Action: "flee", Fled: false, Turn: c.turn,
		})
		return
	}

	// Attaque.
	dmg := damage(actor.char, opp.char)
	opp.char.HP -= dmg
	if opp.char.HP < 0 {
		opp.char.HP = 0
	}
	ev := CombatEventData{
		CombatID: c.id, Actor: actorID, Action: "attack",
		Target: opp.char.ID, Damage: dmg, TargetHP: opp.char.HP,
	}

	if opp.char.HP == 0 { // mort
		z.sendBoth(c, protocol.TypeCombatEvent, ev)
		z.handleDeath(c, actorID, opp.char.ID)
		return
	}
	c.turn = opp.char.ID
	c.deadline = z.tick + z.turnTimeoutTicks
	ev.Turn = c.turn
	z.sendBoth(c, protocol.TypeCombatEvent, ev)
}

// handleDeath fait réapparaître le perdant et clôt le combat.
func (z *Zone) handleDeath(c *combat, winnerID, loserID string) {
	loser := z.members[loserID]
	if loser != nil {
		loser.char.HP = loser.char.MaxHP // réapparition en pleine santé
		loser.char.X, loser.char.Y = 0, 0
		z.dirty[loserID] = true // sa nouvelle position sera diffusée au tick
	}
	respawn := &RespawnInfo{CharacterID: loserID, X: 0, Y: 0}
	if loser != nil {
		respawn.HP = loser.char.MaxHP
	}
	z.endCombat(c, winnerID, loserID, "death", respawn)
}

// endCombat nettoie le combat, applique l'immunité post-combat et notifie.
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
		w.sender.SendEnvelope(protocol.TypeCombatEnd, 0, CombatEndData{
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
		m.sender.SendEnvelope(msgType, 0, data)
	}
	if m := z.members[c.b]; m != nil {
		m.sender.SendEnvelope(msgType, 0, data)
	}
}

// ── Règles de combat ────────────────────────────────────────────────────────

// damage : dégâts d'une attaque, gouvernés par la force de l'attaquant et la
// défense de la cible. Au moins 1 pour qu'un combat se termine toujours.
func damage(atk, def domain.Character) int {
	d := atk.Str*3 - def.Def
	if d < 1 {
		d = 1
	}
	return d
}

// fleeSucceeds : la fuite réussit avec une probabilité gouvernée par l'agilité
// relative des deux personnages.
func (z *Zone) fleeSucceeds(fleer, opp domain.Character) bool {
	chance := 0.35 + 0.02*float64(fleer.Agi-opp.Agi)
	if chance < 0.1 {
		chance = 0.1
	}
	if chance > 0.85 {
		chance = 0.85
	}
	return z.rng.Float64() < chance
}

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
