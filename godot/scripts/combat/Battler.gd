class_name Battler
extends RefCounted
## État runtime d'un combattant (allié ou ennemi) pendant une bataille.
## Construit à partir d'une définition de monstre (voir MonsterDatabase).

var id: String
var display_name: String
var kind: int                       # BattleTypes.Kind favori du monstre
var color: Color = Color.WHITE

var max_hp: int
var hp: int
var attack: int
var defense: int
var speed: int
var skill_name: String
var skill_power: float = 1.6        # multiplicateur de dégâts de la capacité
var is_enemy: bool = false

## Choix d'action du tour courant, rempli par le joueur ou l'IA.
var chosen_kind: int = BattleTypes.Kind.PUISSANCE
var use_skill: bool = false

func setup(def: Dictionary, enemy: bool) -> void:
	id = def.get("id", "?")
	display_name = def.get("name", "Monstre")
	kind = def.get("kind", BattleTypes.Kind.PUISSANCE)
	color = def.get("color", Color.WHITE)
	max_hp = def.get("hp", 40)
	hp = max_hp
	attack = def.get("attack", 12)
	defense = def.get("defense", 6)
	speed = def.get("speed", 10)
	skill_name = def.get("skill", "Frappe")
	skill_power = def.get("skill_power", 1.6)
	is_enemy = enemy

func is_alive() -> bool:
	return hp > 0

func take_damage(amount: int) -> int:
	var dealt: int = clampi(amount, 0, hp)
	hp -= dealt
	return dealt

func heal(amount: int) -> void:
	hp = clampi(hp + amount, 0, max_hp)
