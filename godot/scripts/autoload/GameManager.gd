extends Node
## État global de la partie + transitions de scènes.
## Gère le va-et-vient entre le village (exploration) et l'arène (combat),
## et conserve la position du joueur pour l'y remettre au retour.

const VILLAGE_SCENE := "res://scenes/Village.tscn"
const BATTLE_SCENE := "res://scenes/combat/BattleScene.tscn"

## Progression du joueur, conservée entre les scènes.
var player_gold: int = 50
var player_level: int = 1
var battles_won: int = 0

## Équipe du joueur : liste d'ids de monstres (voir MonsterDatabase).
var party: Array[String] = ["compagnon_lueur"]

## Rencontre en cours (id transmis à BattleScene).
var pending_encounter: String = ""

## Verrou d'entrées : vrai pendant un dialogue (bloque le déplacement).
var input_locked: bool = false

## Position/orientation du joueur à restaurer au retour de combat.
var _saved_player_transform: Transform3D = Transform3D.IDENTITY
var _has_saved_transform: bool = false

func _ready() -> void:
	_ensure_inputs()
	SignalBus.battle_requested.connect(_on_battle_requested)
	SignalBus.battle_ended.connect(_on_battle_ended)

## Garantit l'existence des actions (par touche physique : ZQSD sur AZERTY,
## WASD sur QWERTY) même si le project.godot n'a pas été importé.
func _ensure_inputs() -> void:
	var actions := {
		"move_forward": [KEY_W, KEY_UP],
		"move_back": [KEY_S, KEY_DOWN],
		"move_left": [KEY_A, KEY_LEFT],
		"move_right": [KEY_D, KEY_RIGHT],
		"interact": [KEY_E, KEY_ENTER],
		"ui_confirm_action": [KEY_SPACE],
	}
	for action in actions:
		if InputMap.has_action(action):
			continue   # déjà défini par project.godot : on ne double pas
		InputMap.add_action(action)
		for key in actions[action]:
			var ev := InputEventKey.new()
			ev.physical_keycode = key
			InputMap.action_add_event(action, ev)

func save_player_transform(xform: Transform3D) -> void:
	_saved_player_transform = xform
	_has_saved_transform = true

func consume_player_transform() -> Transform3D:
	# Renvoie la transform sauvegardée une seule fois (sinon identité par défaut).
	if _has_saved_transform:
		_has_saved_transform = false
		return _saved_player_transform
	return Transform3D(Basis.IDENTITY, Vector3(0, 1, 6))

func _on_battle_requested(encounter_id: String) -> void:
	pending_encounter = encounter_id
	_change_scene(BATTLE_SCENE)

func _on_battle_ended(player_won: bool) -> void:
	if player_won:
		battles_won += 1
		player_gold += 30
	_change_scene(VILLAGE_SCENE)

func _change_scene(path: String) -> void:
	# Petit fondu au noir pour adoucir la transition.
	await _fade(true)
	get_tree().change_scene_to_file(path)
	await get_tree().process_frame
	await _fade(false)

func _fade(to_black: bool) -> void:
	var layer := CanvasLayer.new()
	layer.layer = 128
	var rect := ColorRect.new()
	rect.color = Color(0, 0, 0, 0.0 if to_black else 1.0)
	rect.set_anchors_preset(Control.PRESET_FULL_RECT)
	rect.mouse_filter = Control.MOUSE_FILTER_IGNORE
	layer.add_child(rect)
	get_tree().root.add_child(layer)
	var tween := create_tween()
	tween.tween_property(rect, "color:a", 1.0 if to_black else 0.0, 0.35)
	await tween.finished
	if not to_black:
		layer.queue_free()
