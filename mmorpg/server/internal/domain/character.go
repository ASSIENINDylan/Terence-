// Package domain porte les entités métier du jeu et les règles pures qui les
// gouvernent, indépendamment du stockage (store) et du transport (gateway).
package domain

// Character est un personnage joueur : l'entité centrale du jeu (§6). C'est la
// projection « vivante » d'une ligne de la table characters.
type Character struct {
	ID        string // uuid, identifiant stable exposé au client
	AccountID int64
	Name      string
	FactionID int

	// Élément du village de départ (feu/eau/terre) et statistique de spécialité
	// associée (str/def/agi). Chargés via la faction.
	Element  string
	SpecStat string

	Level int
	XP    int64

	HP    int
	MaxHP int

	// Statistiques de base, en flottant : l'académie peut les faire varier de
	// 0,5 point (§ progression).
	Str float64 // attaque
	Def float64 // défense
	Agi float64 // agilité (initiative + fuite)

	// Ressources et progression.
	Energy    int
	MaxEnergy int
	Gold      int64
	Potions   int

	AttrPoints    int      // points d'attribut (académie), 1 par niveau
	PerfectPoints int      // points parfaits (séries de kills)
	TechPoints    int      // points de technique, 1 tous les 5 niveaux
	LearnedTechs  []string // identifiants des techniques apprises (temple)

	// Position courante dans le monde.
	ZoneID int
	X      int
	Y      int

	// HomeZoneID : zone du village de départ (réapparition à la mort).
	HomeZoneID int
}

// Alive indique si le personnage a encore des points de vie.
func (c Character) Alive() bool { return c.HP > 0 }

// KnowsTech indique si le personnage a appris la technique id.
func (c Character) KnowsTech(id string) bool {
	for _, t := range c.LearnedTechs {
		if t == id {
			return true
		}
	}
	return false
}
