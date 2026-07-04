class_name NPC
extends StaticBody3D
## Habitant du village. Interactif (touche E) : lance un dialogue, qui peut
## éventuellement déclencher un combat. Le corps est peint en code (aucun asset).

@export var npc_id: String = "forgeron"
@export var body_color: Color = Color("#c98a52")
@export var can_rotate_to_player: bool = true

var _busy := false

@onready var _body_mesh: MeshInstance3D = $Body
@onready var _head_mesh: MeshInstance3D = $Head

func _ready() -> void:
	add_to_group("interactable")
	_paint()

func _process(delta: float) -> void:
	# Le PNJ se tourne doucement vers le joueur quand il est proche.
	if not can_rotate_to_player:
		return
	var players := get_tree().get_nodes_in_group("player")
	if players.is_empty():
		return
	var p: Node3D = players[0]
	if global_position.distance_to(p.global_position) < 4.0:
		var to_player := p.global_position - global_position
		var want := atan2(to_player.x, to_player.z)
		rotation.y = lerp_angle(rotation.y, want, 6.0 * delta)

func get_prompt() -> String:
	return "Parler à " + DialogueDatabase.get_display_name(npc_id)

func interact() -> void:
	if _busy:
		return
	_busy = true
	await _run_dialogue()
	_busy = false

func _run_dialogue() -> void:
	SignalBus.dialogue_started.emit(DialogueDatabase.get_display_name(npc_id))
	for line in DialogueDatabase.get_lines(npc_id):
		SignalBus.dialogue_line_shown.emit(line["speaker"], line["text"])
		# Attend que l'UI signale « ligne suivante » (touche E/Espace).
		await SignalBus.dialogue_advanced
	SignalBus.dialogue_finished.emit()
	# Un dialogue peut mener à un combat.
	var trigger := DialogueDatabase.get_battle_trigger(npc_id)
	if trigger != "":
		SignalBus.battle_requested.emit(trigger)

func _paint() -> void:
	var mat := StandardMaterial3D.new()
	mat.albedo_color = body_color
	mat.roughness = 0.7
	_body_mesh.material_override = mat
	var skin := StandardMaterial3D.new()
	skin.albedo_color = Color("#e8c39e")
	skin.roughness = 0.6
	_head_mesh.material_override = skin
