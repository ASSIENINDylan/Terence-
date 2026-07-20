// Package rules centralise les règles CHIFFRÉES du jeu : formules de combat,
// montée de niveau, académie (points d'attribut) et techniques. Ce sont des
// fonctions pures, sans état ni I/O — donc faciles à tester et à RÉ-ÉQUILIBRER
// (tous les nombres réglables sont regroupés ci-dessous).
package rules

import (
	"errors"
	"math"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
)

// ── Constantes d'équilibrage (à ajuster librement) ──────────────────────────
const (
	XPPerLevel           = 50 // XP pour passer un niveau = XPPerLevel * niveau
	BaseHP               = 100
	HPPerLevel           = 8
	BaseEnergy           = 50
	EnergyRegenPerTurn   = 3 // énergie regagnée à chaque tour de combat
	TechPointEveryLevels = 5 // 1 point de technique tous les N niveaux

	// Récompenses (un mob rapportera moins ; les mobs viendront plus tard).
	KillXPReward   = 50
	KillGoldReward = 25

	// Académie.
	SpecGain      = 2.0 // spécialité : +2, mais −0,5 aux autres
	SpecPenalty   = 0.5
	OtherGain     = 1.0 // autre attribut : +1, sans malus
	PerfSpecGain  = 2.0 // point parfait sur la spécialité : +2, sans malus
	PerfOtherGain = 1.5 // point parfait sur un autre : +1,5, sans malus
	MinStat       = 1.0

	// Séries de kills → point parfait.
	PerfectKillsOrange = 10
	PerfectKillsRed    = 5

	// Boutique.
	PotionCost = 25
	PotionHeal = 50

	// Formules de combat. Dégâts = échelle × puissance − facteur × réduction.
	DamageScale  = 1.4
	DmgMitFactor = 0.5
	DmgFloor     = 1
	FleeBase     = 0.20
	FleeScale    = 0.03

	// Techniques : l'affinité (stat du village) pèse lourd, mais les deux autres
	// attributs comptent aussi. Nettement plus fortes qu'une attaque normale.
	TechSpecW     = 0.65
	TechOtherW    = 0.25
	TechMitFactor = 0.30

	// IA des mobs : fuir si les PV tombent sous ce seuil et que l'adversaire est
	// encore vaillant.
	MobFleeHPPct = 30
	MobOppHPPct  = 40
)

var (
	ErrNotEnoughPoints = errors.New("rules: pas assez de points")
	ErrUnknownStat     = errors.New("rules: attribut inconnu")
	ErrNotEnoughGold   = errors.New("rules: pas assez d'or")
	ErrNoPotion        = errors.New("rules: aucune potion ou déjà à pleine santé")
	ErrUnknownTech     = errors.New("rules: technique inconnue")
	ErrAlreadyLearned  = errors.New("rules: technique déjà apprise")
)

// ── Formules dérivées (attaque / défense / agilité) ─────────────────────────

// Power : puissance d'attaque.
func Power(c domain.Character) float64 { return 0.5*c.Str + 0.3*c.Def + 0.2*c.Agi }

// Mitig : réduction (défense).
func Mitig(c domain.Character) float64 { return 0.5*c.Def + 0.3*c.Agi + 0.2*c.Str }

// FleeVal : valeur de fuite.
func FleeVal(c domain.Character) float64 { return 0.5*c.Agi + 0.3*c.Str + 0.2*c.Def }

// Damage : dégâts d'une attaque normale.
func Damage(attacker, defender domain.Character) int {
	return floorAtLeast(DamageScale*Power(attacker)-DmgMitFactor*Mitig(defender), DmgFloor)
}

// FleeChance : probabilité de réussir une fuite.
func FleeChance(fleer, opponent domain.Character) float64 {
	ch := FleeBase + FleeScale*(FleeVal(fleer)-FleeVal(opponent))
	return math.Max(0.05, math.Min(0.9, ch))
}

// ── Techniques ──────────────────────────────────────────────────────────────

// Technique est un coup spécial appris au temple. Sa puissance s'applique surtout
// à l'affinité du village, mais les autres attributs comptent aussi.
type Technique struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Cost int     `json:"cost"` // énergie
	Mult float64 `json:"mult"`
}

var techniques = map[string][]Technique{
	"feu": {
		{ID: "feu1", Name: "Flamme ardente", Cost: 15, Mult: 1.7},
		{ID: "feu2", Name: "Déflagration", Cost: 26, Mult: 2.3},
		{ID: "feu3", Name: "Météore infernal", Cost: 42, Mult: 3.1},
	},
	"eau": {
		{ID: "eau1", Name: "Lame liquide", Cost: 15, Mult: 1.7},
		{ID: "eau2", Name: "Raz-de-marée", Cost: 26, Mult: 2.3},
		{ID: "eau3", Name: "Abysse déchaîné", Cost: 42, Mult: 3.1},
	},
	"terre": {
		{ID: "ter1", Name: "Éclat rocheux", Cost: 15, Mult: 1.7},
		{ID: "ter2", Name: "Séisme", Cost: 26, Mult: 2.3},
		{ID: "ter3", Name: "Colosse de pierre", Cost: 42, Mult: 3.1},
	},
}

// ── Mobs PvE ────────────────────────────────────────────────────────────────

// MobSpec décrit un type de mob selon le palier de la zone. Ils rapportent moins
// d'XP et d'or qu'un joueur.
type MobSpec struct {
	Name  string
	Str   float64
	Def   float64
	Agi   float64
	HP    int
	XP    int64
	Gold  int64
	Count int // nombre de mobs présents dans la zone
}

var mobSpecs = map[string]MobSpec{
	"green":  {Name: "Lutin des prés", Str: 6, Def: 6, Agi: 6, HP: 34, XP: 8, Gold: 5, Count: 3},
	"orange": {Name: "Maraudeur", Str: 12, Def: 10, Agi: 9, HP: 62, XP: 16, Gold: 12, Count: 3},
	"red":    {Name: "Damné", Str: 17, Def: 14, Agi: 13, HP: 95, XP: 26, Gold: 22, Count: 4},
}

// MobSpecFor retourne la spec de mob d'un palier (ok=false si le palier n'a pas
// de mobs — village).
func MobSpecFor(tier string) (MobSpec, bool) {
	s, ok := mobSpecs[tier]
	return s, ok
}

// TechniquesFor retourne les techniques disponibles pour un élément.
func TechniquesFor(element string) []Technique { return techniques[element] }

// FindTech retrouve une technique d'un élément par son identifiant.
func FindTech(element, id string) (Technique, bool) {
	for _, t := range techniques[element] {
		if t.ID == id {
			return t, true
		}
	}
	return Technique{}, false
}

// TechDamage : dégâts d'une technique — dominés par l'affinité, mais les autres
// attributs y contribuent aussi.
func TechDamage(attacker domain.Character, t Technique, defender domain.Character) int {
	spec := statValue(attacker, attacker.SpecStat)
	others := (attacker.Str + attacker.Def + attacker.Agi) - spec
	base := t.Mult * (TechSpecW*spec + TechOtherW*others)
	return floorAtLeast(base-TechMitFactor*Mitig(defender), DmgFloor)
}

// ── Progression / niveaux ───────────────────────────────────────────────────

// XPNeeded : XP requise pour passer du niveau donné au suivant.
func XPNeeded(level int) int64 { return int64(XPPerLevel) * int64(level) }

// MaxHPForLevel : PV maximum à un niveau donné.
func MaxHPForLevel(level int) int { return BaseHP + (level-1)*HPPerLevel }

// ApplyXP ajoute de l'XP et applique les montées de niveau : chaque niveau donne
// un point d'attribut, un point de technique tous les TechPointEveryLevels, et
// augmente les PV max (soin complet). Retourne le nombre de niveaux gagnés.
func ApplyXP(c *domain.Character, gained int64) (levelsGained int) {
	c.XP += gained
	for c.XP >= XPNeeded(c.Level) {
		c.XP -= XPNeeded(c.Level)
		c.Level++
		c.AttrPoints++
		if c.Level%TechPointEveryLevels == 0 {
			c.TechPoints++
		}
		c.MaxHP = MaxHPForLevel(c.Level)
		c.HP = c.MaxHP
		levelsGained++
	}
	return levelsGained
}

// ── Académie (points d'attribut) ────────────────────────────────────────────

// SpendAttr dépense un point d'attribut (normal ou parfait) sur str/def/agi.
//   - normal, spécialité : +2 mais −0,5 aux autres.
//   - normal, autre : +1, sans malus.
//   - parfait, spécialité : +2, sans malus ; parfait, autre : +1,5, sans malus.
func SpendAttr(c *domain.Character, attr string, perfect bool) error {
	if attr != "str" && attr != "def" && attr != "agi" {
		return ErrUnknownStat
	}
	if perfect {
		if c.PerfectPoints < 1 {
			return ErrNotEnoughPoints
		}
		c.PerfectPoints--
		if attr == c.SpecStat {
			addStat(c, attr, PerfSpecGain)
		} else {
			addStat(c, attr, PerfOtherGain)
		}
		return nil
	}
	if c.AttrPoints < 1 {
		return ErrNotEnoughPoints
	}
	c.AttrPoints--
	if attr == c.SpecStat {
		addStat(c, attr, SpecGain)
		for _, o := range []string{"str", "def", "agi"} {
			if o != attr {
				addStat(c, o, -SpecPenalty)
			}
		}
	} else {
		addStat(c, attr, OtherGain)
	}
	return nil
}

// ── Techniques : apprentissage ──────────────────────────────────────────────

// LearnTech apprend une technique au temple (dépense un point de technique).
func LearnTech(c *domain.Character, id string) (Technique, error) {
	t, ok := FindTech(c.Element, id)
	if !ok {
		return Technique{}, ErrUnknownTech
	}
	if c.KnowsTech(id) {
		return Technique{}, ErrAlreadyLearned
	}
	if c.TechPoints < 1 {
		return Technique{}, ErrNotEnoughPoints
	}
	c.TechPoints--
	c.LearnedTechs = append(c.LearnedTechs, id)
	return t, nil
}

// ── Boutique ────────────────────────────────────────────────────────────────

// BuyPotion achète une potion de soin.
func BuyPotion(c *domain.Character) error {
	if c.Gold < PotionCost {
		return ErrNotEnoughGold
	}
	c.Gold -= PotionCost
	c.Potions++
	return nil
}

// UsePotion consomme une potion et soigne. Retourne les PV rendus.
func UsePotion(c *domain.Character) (int, error) {
	if c.Potions < 1 || c.HP >= c.MaxHP {
		return 0, ErrNoPotion
	}
	c.Potions--
	heal := PotionHeal
	if heal > c.MaxHP-c.HP {
		heal = c.MaxHP - c.HP
	}
	c.HP += heal
	return heal, nil
}

// PerfectKillThreshold : nombre de kills consécutifs (sans quitter la zone) pour
// un point parfait, selon le palier. 0 si le palier n'en donne pas.
func PerfectKillThreshold(tier string) int {
	switch tier {
	case "orange":
		return PerfectKillsOrange
	case "red":
		return PerfectKillsRed
	default:
		return 0
	}
}

// ── Utilitaires ─────────────────────────────────────────────────────────────

func statValue(c domain.Character, key string) float64 {
	switch key {
	case "str":
		return c.Str
	case "def":
		return c.Def
	case "agi":
		return c.Agi
	}
	return 0
}

func addStat(c *domain.Character, key string, delta float64) {
	switch key {
	case "str":
		c.Str = math.Max(MinStat, c.Str+delta)
	case "def":
		c.Def = math.Max(MinStat, c.Def+delta)
	case "agi":
		c.Agi = math.Max(MinStat, c.Agi+delta)
	}
}

func floorAtLeast(v float64, floor int) int {
	r := int(math.Round(v))
	if r < floor {
		return floor
	}
	return r
}
