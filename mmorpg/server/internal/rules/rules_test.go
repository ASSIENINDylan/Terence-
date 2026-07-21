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

// ── Objets & inventaire (T6) ─────────────────────────────────────────────────

func itemCatalogue() *domain.Catalogue {
	return domain.NewCatalogue([]domain.ItemTemplate{
		{ID: 1, Code: "epee", Name: "Épée", Type: domain.ItemWeapon, Stats: domain.ItemStats{Atk: 6}},
		{ID: 2, Code: "cotte", Name: "Cotte", Type: domain.ItemArmor, Stats: domain.ItemStats{Def: 4}},
		{ID: 3, Code: "potion", Name: "Potion", Type: domain.ItemConsumable, Stackable: true, MaxStack: 10, Stats: domain.ItemStats{Heal: 40}},
	})
}

func TestEquipBonusAppliesToEffectiveStatsAndDamage(t *testing.T) {
	cat := itemCatalogue()
	c := feu() // 15/10/10, str spec
	c.Inventory = []domain.InventoryItem{
		{ID: "a", TemplateID: 1, Quantity: 1, Equipped: true}, // arme +6 atk
		{ID: "b", TemplateID: 2, Quantity: 1, Equipped: true}, // armure +4 def
		{ID: "c", TemplateID: 3, Quantity: 2},                 // potion, non équipée
	}
	str, def, agi := EquipBonus(cat, c.Inventory)
	if str != 6 || def != 4 || agi != 0 {
		t.Fatalf("bonus d'équipement attendus (6,4,0), obtenus (%v,%v,%v)", str, def, agi)
	}
	c.StrBonus, c.DefBonus, c.AgiBonus = str, def, agi
	if c.EffStr() != 21 || c.EffDef() != 14 || c.EffAgi() != 10 {
		t.Fatalf("stats effectives incorrectes : %v/%v/%v", c.EffStr(), c.EffDef(), c.EffAgi())
	}
	// Les dégâts effectifs dépassent les dégâts « nus » (bonus d'attaque pris en compte).
	def2 := domain.Character{Str: 10, Def: 10, Agi: 10}
	bare := feu()
	if Damage(c, def2) <= Damage(bare, def2) {
		t.Fatalf("l'équipement devrait augmenter les dégâts : équipé=%d nu=%d", Damage(c, def2), Damage(bare, def2))
	}
}

func TestConsumeHeal(t *testing.T) {
	cat := itemCatalogue()
	potion, _ := cat.ByID(3)
	c := feu()
	c.HP, c.MaxHP = 30, 100
	healed, err := ConsumeHeal(&c, potion)
	if err != nil || healed != 40 || c.HP != 70 {
		t.Fatalf("soin attendu +40 → 70 PV, obtenu healed=%d hp=%d err=%v", healed, c.HP, err)
	}
	// À pleine santé : refus.
	c.HP = c.MaxHP
	if _, err := ConsumeHeal(&c, potion); err != ErrNoHeal {
		t.Fatalf("attendu ErrNoHeal à pleine santé, obtenu %v", err)
	}
	// Un objet non consommable ne soigne pas.
	sword, _ := cat.ByID(1)
	c.HP = 10
	if _, err := ConsumeHeal(&c, sword); err != ErrNotConsumable {
		t.Fatalf("attendu ErrNotConsumable pour une arme, obtenu %v", err)
	}
}

func TestLootCodesFor(t *testing.T) {
	if len(LootCodesFor("green")) == 0 || len(LootCodesFor("red")) == 0 {
		t.Fatal("les paliers vert/rouge devraient lâcher du butin")
	}
	if LootCodesFor("village") != nil {
		t.Fatal("le village ne devrait rien lâcher")
	}
}
