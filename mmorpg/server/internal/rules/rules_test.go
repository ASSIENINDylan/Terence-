package rules

import (
	"testing"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
)

func feu() domain.Character {
	// Feu : spécialité attaque (str), 15/10/10.
	return domain.Character{Element: "feu", SpecStat: "str", Level: 1, HP: 100, MaxHP: 100, Str: 15, Def: 10, Agi: 10, MaxEnergy: BaseEnergy, Energy: BaseEnergy}
}

func TestFormulas(t *testing.T) {
	c := feu()
	if got := Power(c); got != 12.5 { // 0.5*15+0.3*10+0.2*10
		t.Fatalf("Power: attendu 12.5, obtenu %v", got)
	}
	if got := Mitig(c); got != 11 { // 0.5*10+0.3*10+0.2*15
		t.Fatalf("Mitig: attendu 11, obtenu %v", got)
	}
	if got := FleeVal(c); got != 11.5 { // 0.5*10+0.3*15+0.2*10
		t.Fatalf("FleeVal: attendu 11.5, obtenu %v", got)
	}
}

func TestTechDamageStrongerAndMultiAttribute(t *testing.T) {
	atk := feu()
	def := domain.Character{Str: 10, Def: 10, Agi: 10}

	normal := Damage(atk, def)
	tech, _ := FindTech("feu", "feu1")
	techDmg := TechDamage(atk, tech, def)

	if techDmg <= normal {
		t.Fatalf("une technique doit taper plus fort qu'une attaque normale (tech=%d, normal=%d)", techDmg, normal)
	}

	// Les autres attributs comptent : augmenter def/agi (non-spécialité) doit
	// augmenter les dégâts de technique (donc l'affinité n'est pas le seul facteur).
	atk2 := atk
	atk2.Def += 20
	atk2.Agi += 20
	if TechDamage(atk2, tech, def) <= techDmg {
		t.Fatalf("les attributs hors affinité doivent aussi compter dans les dégâts de technique")
	}

	// Mais l'affinité doit peser davantage : +X sur la spécialité rapporte plus
	// que +X réparti sur un autre attribut.
	bySpec := atk
	bySpec.Str += 10
	byOther := atk
	byOther.Def += 10
	if TechDamage(bySpec, tech, def) <= TechDamage(byOther, tech, def) {
		t.Fatalf("l'affinité doit peser plus que les autres attributs")
	}
}

func TestLevelingGrantsPointsAndTechPoints(t *testing.T) {
	c := feu()
	// XP requise : niveau 1→2 = 50, 2→3 = 100, ... jusqu'à 5→6 = 250.
	// Total pour atteindre le niveau 6 : 50+100+150+200+250 = 750.
	ApplyXP(&c, 750)
	if c.Level != 6 {
		t.Fatalf("niveau attendu 6, obtenu %d", c.Level)
	}
	if c.AttrPoints != 5 {
		t.Fatalf("points d'attribut attendus 5, obtenu %d", c.AttrPoints)
	}
	if c.TechPoints != 1 { // 1 tous les 5 niveaux → niveau 5 franchi
		t.Fatalf("point de technique attendu 1, obtenu %d", c.TechPoints)
	}
	if c.MaxHP != MaxHPForLevel(6) {
		t.Fatalf("PV max non mis à jour: %d", c.MaxHP)
	}
}

func TestAcademySpending(t *testing.T) {
	c := feu()
	c.AttrPoints = 2
	c.PerfectPoints = 1

	// Point normal sur la spécialité (str) : +2, −0,5 aux autres.
	if err := SpendAttr(&c, "str", false); err != nil {
		t.Fatalf("spend spé: %v", err)
	}
	if c.Str != 17 || c.Def != 9.5 || c.Agi != 9.5 {
		t.Fatalf("après point spé: attendu 17/9.5/9.5, obtenu %v/%v/%v", c.Str, c.Def, c.Agi)
	}
	// Point normal sur un autre attribut (def) : +1, sans malus.
	if err := SpendAttr(&c, "def", false); err != nil {
		t.Fatalf("spend autre: %v", err)
	}
	if c.Def != 10.5 || c.Str != 17 || c.Agi != 9.5 {
		t.Fatalf("après point autre: attendu 17/10.5/9.5, obtenu %v/%v/%v", c.Str, c.Def, c.Agi)
	}
	// Point parfait sur la spécialité (str) : +2, sans malus.
	if err := SpendAttr(&c, "str", true); err != nil {
		t.Fatalf("spend parfait: %v", err)
	}
	if c.Str != 19 || c.Def != 10.5 || c.Agi != 9.5 {
		t.Fatalf("après parfait spé: attendu 19/10.5/9.5, obtenu %v/%v/%v", c.Str, c.Def, c.Agi)
	}
	// Plus de points : erreur.
	if err := SpendAttr(&c, "str", false); err != ErrNotEnoughPoints {
		t.Fatalf("attendu ErrNotEnoughPoints, obtenu %v", err)
	}
}

func TestLearnTechAndPotions(t *testing.T) {
	c := feu()
	if _, err := LearnTech(&c, "feu1"); err != ErrNotEnoughPoints {
		t.Fatalf("sans point de technique: attendu ErrNotEnoughPoints, obtenu %v", err)
	}
	c.TechPoints = 1
	if _, err := LearnTech(&c, "feu1"); err != nil {
		t.Fatalf("apprentissage: %v", err)
	}
	if !c.KnowsTech("feu1") || c.TechPoints != 0 {
		t.Fatalf("technique non apprise correctement")
	}
	if _, err := LearnTech(&c, "eau1"); err != ErrUnknownTech {
		t.Fatalf("technique d'un autre élément: attendu ErrUnknownTech, obtenu %v", err)
	}

	// Potions.
	c.Gold = 30
	if err := BuyPotion(&c); err != nil || c.Potions != 1 || c.Gold != 30-PotionCost {
		t.Fatalf("achat potion incorrect: potions=%d gold=%d err=%v", c.Potions, c.Gold, err)
	}
	c.HP = 20
	healed, err := UsePotion(&c)
	if err != nil || healed != PotionHeal || c.HP != 70 || c.Potions != 0 {
		t.Fatalf("usage potion incorrect: healed=%d hp=%d err=%v", healed, c.HP, err)
	}
}

func TestPerfectKillThreshold(t *testing.T) {
	if PerfectKillThreshold("orange") != PerfectKillsOrange || PerfectKillThreshold("red") != PerfectKillsRed || PerfectKillThreshold("green") != 0 {
		t.Fatal("seuils de kills parfaits incorrects")
	}
}
