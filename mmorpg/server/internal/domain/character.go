// Package domain porte les entités métier du jeu et les règles pures qui les
// gouvernent, indépendamment du stockage (store) et du transport (gateway).
// En T3 : le personnage. Les objets, territoires… suivront.
package domain

// Character est un personnage joueur : l'entité centrale du jeu (§6), exposée au
// client par son uuid. C'est la projection « vivante » d'une ligne de la table
// characters — l'état chargé en mémoire pendant qu'un joueur est connecté.
type Character struct {
	ID        string // uuid, identifiant stable exposé au client
	AccountID int64
	Name      string
	FactionID int

	Level int
	XP    int64
	HP    int
	MaxHP int
	Str   int // attaque
	Def   int // défense
	Agi   int // agilité (initiative + fuite)

	// Position courante dans le monde.
	ZoneID int
	X      int
	Y      int

	// HomeZoneID est la zone du village de départ : lieu de réapparition à la
	// mort (§ paliers de zones).
	HomeZoneID int
}

// Alive indique si le personnage a encore des points de vie. Le combat (T5)
// s'appuiera dessus.
func (c Character) Alive() bool { return c.HP > 0 }
