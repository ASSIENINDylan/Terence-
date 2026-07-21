package domain

// ItemType classe un objet du catalogue (§8). Une arme et une armure s'équipent
// et modifient les statistiques ; un consommable se boit/utilise une fois.
type ItemType string

const (
	ItemWeapon     ItemType = "weapon"
	ItemArmor      ItemType = "armor"
	ItemConsumable ItemType = "consumable"
)

// ItemStats est la charge « stats » (jsonb) d'un template : bonus d'équipement
// (atk/def/agi) ou effet d'un consommable (heal).
type ItemStats struct {
	Atk  float64 `json:"atk,omitempty"`  // bonus d'attaque (arme)
	Def  float64 `json:"def,omitempty"`  // bonus de défense (armure)
	Agi  float64 `json:"agi,omitempty"`  // bonus d'agilité
	Heal int     `json:"heal,omitempty"` // soin rendu (consommable)
}

// ItemTemplate est une entrée immuable du catalogue d'objets.
type ItemTemplate struct {
	ID        int
	Code      string
	Name      string
	Type      ItemType
	Stackable bool
	MaxStack  int
	Stats     ItemStats
}

// InventoryItem est un exemplaire possédé (§8) : un identifiant unique par
// exemplaire est le pilier de l'anti-duplication.
type InventoryItem struct {
	ID         string `json:"id"` // uuid de l'exemplaire
	TemplateID int    `json:"template_id"`
	Quantity   int    `json:"quantity"`
	Equipped   bool   `json:"equipped"`
}

// Catalogue indexe le catalogue immuable d'objets par identifiant et par code,
// chargé une fois au démarrage (données de référence stables).
type Catalogue struct {
	byID   map[int]ItemTemplate
	byCode map[string]ItemTemplate
}

// NewCatalogue construit un catalogue à partir des templates chargés.
func NewCatalogue(items []ItemTemplate) *Catalogue {
	c := &Catalogue{
		byID:   make(map[int]ItemTemplate, len(items)),
		byCode: make(map[string]ItemTemplate, len(items)),
	}
	for _, it := range items {
		c.byID[it.ID] = it
		c.byCode[it.Code] = it
	}
	return c
}

// ByID retrouve un template par identifiant.
func (c *Catalogue) ByID(id int) (ItemTemplate, bool) {
	if c == nil {
		return ItemTemplate{}, false
	}
	t, ok := c.byID[id]
	return t, ok
}

// ByCode retrouve un template par code.
func (c *Catalogue) ByCode(code string) (ItemTemplate, bool) {
	if c == nil {
		return ItemTemplate{}, false
	}
	t, ok := c.byCode[code]
	return t, ok
}

// ── Statistiques effectives (base + équipement) ─────────────────────────────
//
// Les statistiques de base (Str/Def/Agi) proviennent des niveaux et de
// l'académie. L'équipement ajoute des bonus, recalculés à chaque changement
// d'équipement. Le combat utilise les valeurs EFFECTIVES.

// EffStr : attaque effective (base + bonus d'équipement).
func (c Character) EffStr() float64 { return c.Str + c.StrBonus }

// EffDef : défense effective.
func (c Character) EffDef() float64 { return c.Def + c.DefBonus }

// EffAgi : agilité effective.
func (c Character) EffAgi() float64 { return c.Agi + c.AgiBonus }

// FindItem retrouve un exemplaire de l'inventaire par son identifiant.
func (c *Character) FindItem(id string) (*InventoryItem, bool) {
	for i := range c.Inventory {
		if c.Inventory[i].ID == id {
			return &c.Inventory[i], true
		}
	}
	return nil, false
}
