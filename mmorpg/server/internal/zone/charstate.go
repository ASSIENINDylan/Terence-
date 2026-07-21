package zone

import (
	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
	"github.com/assienindylan/terence-/mmorpg/server/internal/protocol"
	"github.com/assienindylan/terence-/mmorpg/server/internal/rules"
)

// CharState est l'état complet du personnage envoyé au client (message
// char.update) pour alimenter la fiche : niveau, XP, statistiques, ressources,
// points à dépenser et catalogue de techniques de l'élément.
type CharState struct {
	Level    int   `json:"level"`
	XP       int64 `json:"xp"`
	XPNeeded int64 `json:"xp_needed"`

	Str float64 `json:"str"`
	Def float64 `json:"def"`
	Agi float64 `json:"agi"`

	// Bonus d'équipement (armes/armures) : le client affiche « base (+bonus) ».
	StrBonus float64 `json:"str_bonus"`
	DefBonus float64 `json:"def_bonus"`
	AgiBonus float64 `json:"agi_bonus"`

	HP        int `json:"hp"`
	MaxHP     int `json:"max_hp"`
	Energy    int `json:"energy"`
	MaxEnergy int `json:"max_energy"`

	Gold    int64 `json:"gold"`
	Potions int   `json:"potions"`

	AttrPoints    int `json:"attr_points"`
	PerfectPoints int `json:"perfect_points"`
	TechPoints    int `json:"tech_points"`

	Element      string            `json:"element"`
	SpecStat     string            `json:"spec_stat"`
	LearnedTechs []string          `json:"learned_techs"`
	Techniques   []rules.Technique `json:"techniques"` // catalogue de l'élément (temple)
}

// CharStateOf construit l'état à envoyer au client depuis un personnage.
func CharStateOf(c domain.Character) CharState {
	learned := c.LearnedTechs
	if learned == nil {
		learned = []string{}
	}
	return CharState{
		Level: c.Level, XP: c.XP, XPNeeded: rules.XPNeeded(c.Level),
		Str: c.Str, Def: c.Def, Agi: c.Agi,
		StrBonus: c.StrBonus, DefBonus: c.DefBonus, AgiBonus: c.AgiBonus,
		HP: c.HP, MaxHP: c.MaxHP, Energy: c.Energy, MaxEnergy: c.MaxEnergy,
		Gold: c.Gold, Potions: c.Potions,
		AttrPoints: c.AttrPoints, PerfectPoints: c.PerfectPoints, TechPoints: c.TechPoints,
		Element: c.Element, SpecStat: c.SpecStat, LearnedTechs: learned,
		Techniques: rules.TechniquesFor(c.Element),
	}
}

// sendCharUpdate envoie l'état courant du personnage à son client.
func (z *Zone) sendCharUpdate(m *member) {
	if m != nil {
		m.client.SendEnvelope(protocol.TypeCharUpdate, 0, CharStateOf(m.char))
	}
}
