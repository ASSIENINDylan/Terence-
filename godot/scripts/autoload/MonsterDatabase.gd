extends Node
## Base de données des monstres/alliés et des rencontres.
## Les couleurs servent à peindre les modèles 3D (générés par code, sans asset).

# En var (et non const) car Color("#..") n'est pas une expression constante.
var MONSTERS := {
	"compagnon_lueur": {
		"id": "compagnon_lueur", "name": "Lueur", "kind": BattleTypes.Kind.TECHNIQUE,
		"color": Color("#69c86b"), "hp": 60, "attack": 14, "defense": 8, "speed": 12,
		"skill": "Éclat Sylvestre", "skill_power": 1.8,
	},
	"molosse": {
		"id": "molosse", "name": "Molosse des Remparts", "kind": BattleTypes.Kind.PUISSANCE,
		"color": Color("#e05a4d"), "hp": 55, "attack": 15, "defense": 7, "speed": 8,
		"skill": "Charge Brutale", "skill_power": 1.7,
	},
	"rôdeur": {
		"id": "rôdeur", "name": "Rôdeur Véloce", "kind": BattleTypes.Kind.VITESSE,
		"color": Color("#4db6e0"), "hp": 45, "attack": 13, "defense": 5, "speed": 16,
		"skill": "Danse des Lames", "skill_power": 1.6,
	},
	"gardien_pierre": {
		"id": "gardien_pierre", "name": "Gardien de Pierre", "kind": BattleTypes.Kind.TECHNIQUE,
		"color": Color("#9a86c4"), "hp": 80, "attack": 12, "defense": 12, "speed": 6,
		"skill": "Rempart Runique", "skill_power": 1.5,
	},
}

## Rencontres : quels ennemis affronter, avec quels alliés (au-delà de l'équipe).
const ENCOUNTERS := {
	"molosse_remparts": {
		"title": "Un molosse bloque le chemin !",
		"enemies": ["molosse"],
	},
	"duel_garde": {
		"title": "Garde Halbert vous défie en duel !",
		"enemies": ["rôdeur", "gardien_pierre"],
	},
}

func get_monster(id: String) -> Dictionary:
	return MONSTERS.get(id, MONSTERS["molosse"]).duplicate(true)

func get_encounter(id: String) -> Dictionary:
	return ENCOUNTERS.get(id, ENCOUNTERS["molosse_remparts"]).duplicate(true)
