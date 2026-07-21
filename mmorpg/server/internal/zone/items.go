package zone

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
	"github.com/assienindylan/terence-/mmorpg/server/internal/rules"
)

// ── État réseau de l'inventaire ─────────────────────────────────────────────

// InventoryEntry est une ligne d'inventaire enrichie du template (nom, type,
// stats) pour l'affichage client.
type InventoryEntry struct {
	ID         string           `json:"id"`
	TemplateID int              `json:"template_id"`
	Code       string           `json:"code"`
	Name       string           `json:"name"`
	Type       domain.ItemType  `json:"type"`
	Quantity   int              `json:"quantity"`
	Equipped   bool             `json:"equipped"`
	Stats      domain.ItemStats `json:"stats"`
}

// InventoryData est le message char.inventory : l'inventaire complet du joueur.
type InventoryData struct {
	Items []InventoryEntry `json:"items"`
}

// inventoryDataOf enrichit l'inventaire d'un personnage via le catalogue.
func (z *Zone) inventoryDataOf(c domain.Character) InventoryData {
	items := make([]InventoryEntry, 0, len(c.Inventory))
	for _, it := range c.Inventory {
		e := InventoryEntry{
			ID: it.ID, TemplateID: it.TemplateID, Quantity: it.Quantity, Equipped: it.Equipped,
		}
		if t, ok := z.cat.ByID(it.TemplateID); ok {
			e.Code, e.Name, e.Type, e.Stats = t.Code, t.Name, t.Type, t.Stats
		}
		items = append(items, e)
	}
	return InventoryData{Items: items}
}

// sendInventory pousse l'inventaire courant au client.
func (z *Zone) sendInventory(m *member) {
	if m != nil {
		m.client.SendEnvelope(protocol.TypeInventory, 0, z.inventoryDataOf(m.char))
	}
}

// recomputeEquip recalcule les bonus d'équipement d'un membre à partir de son
// inventaire (objets équipés) et du catalogue.
func (z *Zone) recomputeEquip(m *member) {
	str, def, agi := rules.EquipBonus(z.cat, m.char.Inventory)
	m.char.StrBonus, m.char.DefBonus, m.char.AgiBonus = str, def, agi
}

// ── Butin : objets lâchés au sol ────────────────────────────────────────────

// dropLoot fait tomber un objet du palier à la position (x, y). Aucun effet si
// le palier ne lâche rien ou si le catalogue est absent. L'apparition est
// diffusée au prochain tick.
func (z *Zone) dropLoot(x, y int) {
	codes := rules.LootCodesFor(z.meta.Tier)
	if len(codes) == 0 || z.cat == nil {
		return
	}
	code := codes[z.rng.Intn(len(codes))]
	t, ok := z.cat.ByCode(code)
	if !ok {
		return
	}
	g := &groundItem{id: newUUID(), templateID: t.ID, x: clamp(x), y: clamp(y)}
	z.ground[g.id] = g
	z.itemsGround = append(z.itemsGround, z.groundStateOf(g))
}

// ── Commandes d'inventaire (Manager → acteur de zone) ───────────────────────

// PickupItem ramasse un objet au sol proche du joueur.
func (m *Manager) PickupItem(char domain.Character, itemID string) {
	if z := m.lookup(char.ZoneID); z != nil {
		z.cmds <- func() { z.pickupItem(char.ID, itemID) }
	}
}

// EquipItem équipe (ou déséquipe) une arme/armure de l'inventaire.
func (m *Manager) EquipItem(char domain.Character, itemID string) {
	if z := m.lookup(char.ZoneID); z != nil {
		z.cmds <- func() { z.equipItem(char.ID, itemID) }
	}
}

// UseItem consomme un objet (potion/élixir) de l'inventaire.
func (m *Manager) UseItem(char domain.Character, itemID string) {
	if z := m.lookup(char.ZoneID); z != nil {
		z.cmds <- func() { z.useItem(char.ID, itemID) }
	}
}

// BuyItem achète une arme/armure au forgeron (service du village).
func (m *Manager) BuyItem(char domain.Character, code string) {
	if z := m.lookup(char.ZoneID); z != nil {
		z.villageAction(char.ID, func(mem *member) { z.buyItem(mem, code) })
	}
}

func (z *Zone) buyItem(mem *member, code string) {
	price, ok := rules.ItemPrice(code)
	t, ok2 := z.cat.ByCode(code)
	if !ok || !ok2 || !rules.IsEquipable(t.Type) {
		mem.client.SendEnvelope(protocol.TypeError, 0, protocol.ErrorData{Code: "unknown_item", Message: "objet indisponible à la forge"})
		return
	}
	if mem.char.Gold < price {
		mem.client.SendEnvelope(protocol.TypeError, 0, protocol.ErrorData{Code: "not_enough_gold", Message: "pas assez d'or"})
		return
	}
	mem.char.Gold -= price
	mem.char.Inventory = append(mem.char.Inventory, domain.InventoryItem{ID: newUUID(), TemplateID: t.ID, Quantity: 1})
	z.sendInventory(mem)
	z.sendCharUpdate(mem)
}

func (z *Zone) pickupItem(charID, itemID string) {
	mem := z.members[charID]
	if mem == nil {
		return
	}
	g := z.ground[itemID]
	if g == nil {
		mem.client.SendEnvelope(protocol.TypeError, 0, protocol.ErrorData{
			Code: "item_gone", Message: "cet objet n'est plus au sol",
		})
		return
	}
	if dx, dy := mem.char.X-g.x, mem.char.Y-g.y; dx*dx+dy*dy > pickupRadius*pickupRadius {
		mem.client.SendEnvelope(protocol.TypeError, 0, protocol.ErrorData{
			Code: "too_far", Message: "approche-toi de l'objet",
		})
		return
	}
	t, ok := z.cat.ByID(g.templateID)
	if !ok {
		return
	}
	// Objet empilable : on cumule sur un exemplaire existant si possible.
	if t.Stackable {
		for i := range mem.char.Inventory {
			ex := &mem.char.Inventory[i]
			if ex.TemplateID == g.templateID && ex.Quantity < t.MaxStack {
				ex.Quantity++
				z.takeGround(g)
				z.sendInventory(mem)
				return
			}
		}
	}
	// Sinon, nouvel exemplaire : il conserve l'identifiant de l'objet au sol.
	mem.char.Inventory = append(mem.char.Inventory, domain.InventoryItem{
		ID: g.id, TemplateID: g.templateID, Quantity: 1, Equipped: false,
	})
	z.takeGround(g)
	z.sendInventory(mem)
}

// takeGround retire un objet du sol et programme sa disparition côté client.
func (z *Zone) takeGround(g *groundItem) {
	delete(z.ground, g.id)
	z.itemsTaken = append(z.itemsTaken, g.id)
}

func (z *Zone) equipItem(charID, itemID string) {
	mem := z.members[charID]
	if mem == nil {
		return
	}
	it, ok := mem.char.FindItem(itemID)
	if !ok {
		return
	}
	t, ok := z.cat.ByID(it.TemplateID)
	if !ok || !rules.IsEquipable(t.Type) {
		mem.client.SendEnvelope(protocol.TypeError, 0, protocol.ErrorData{
			Code: "not_equipable", Message: "cet objet ne s'équipe pas",
		})
		return
	}
	if it.Equipped {
		it.Equipped = false
	} else {
		// Un seul objet équipé par emplacement (arme/armure) : on retire l'autre.
		slot := rules.EquipSlot(t.Type)
		for i := range mem.char.Inventory {
			other := &mem.char.Inventory[i]
			if !other.Equipped {
				continue
			}
			if ot, ok := z.cat.ByID(other.TemplateID); ok && rules.EquipSlot(ot.Type) == slot {
				other.Equipped = false
			}
		}
		it.Equipped = true
	}
	z.recomputeEquip(mem)
	z.sendInventory(mem)
	z.sendCharUpdate(mem) // les statistiques effectives ont changé
}

func (z *Zone) useItem(charID, itemID string) {
	mem := z.members[charID]
	if mem == nil {
		return
	}
	it, ok := mem.char.FindItem(itemID)
	if !ok {
		return
	}
	t, ok := z.cat.ByID(it.TemplateID)
	if !ok {
		return
	}
	if _, err := rules.ConsumeHeal(&mem.char, t); err != nil {
		mem.client.SendEnvelope(protocol.TypeError, 0, protocol.ErrorData{
			Code: "use_failed", Message: err.Error(),
		})
		return
	}
	it.Quantity--
	if it.Quantity <= 0 {
		z.removeItem(mem, itemID)
	}
	z.sendInventory(mem)
	z.sendCharUpdate(mem)
}

// removeItem retire un exemplaire de l'inventaire d'un membre.
func (z *Zone) removeItem(mem *member, itemID string) {
	inv := mem.char.Inventory[:0]
	for _, it := range mem.char.Inventory {
		if it.ID != itemID {
			inv = append(inv, it)
		}
	}
	mem.char.Inventory = inv
}

// newUUID génère un UUID v4 (identifiant d'exemplaire, compatible colonne uuid).
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variante RFC 4122
	var s [36]byte
	hex.Encode(s[0:8], b[0:4])
	s[8] = '-'
	hex.Encode(s[9:13], b[4:6])
	s[13] = '-'
	hex.Encode(s[14:18], b[6:8])
	s[18] = '-'
	hex.Encode(s[19:23], b[8:10])
	s[23] = '-'
	hex.Encode(s[24:36], b[10:16])
	return string(s[:])
}
