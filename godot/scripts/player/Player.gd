class_name Player
extends CharacterBody3D
## Contrôleur du personnage joueur en 3D.
## - Déplacement Z/Q/S/D relatif à la caméra, avec rotation douce du modèle.
## - Détection de l'interactif le plus proche (PNJ, monstre) et touche E.

@export var move_speed: float = 6.0
@export var acceleration: float = 12.0
@export var rotation_speed: float = 12.0
@export var interact_range: float = 2.6

var _gravity: float = ProjectSettings.get_setting("physics/3d/default_gravity", 24.0)
var _current_interactable: Node = null

@onready var _model: Node3D = $Model
@onready var _prompt: Label3D = $InteractPrompt

func _ready() -> void:
	add_to_group("player")
	# Restaure la position d'avant-combat le cas échéant.
	global_transform = GameManager.consume_player_transform()
	_prompt.visible = false

func _physics_process(delta: float) -> void:
	_handle_movement(delta)
	_update_interactable()

func _handle_movement(delta: float) -> void:
	# Gravité.
	if not is_on_floor():
		velocity.y -= _gravity * delta

	# Direction d'entrée, projetée dans le repère de la caméra.
	# Bloquée pendant les dialogues.
	var input := Vector2.ZERO
	if not GameManager.input_locked:
		input = Input.get_vector("move_left", "move_right", "move_forward", "move_back")
	var cam := get_viewport().get_camera_3d()
	var dir := Vector3.ZERO
	if cam and input != Vector2.ZERO:
		var basis := cam.global_transform.basis
		var forward := -Vector3(basis.z.x, 0, basis.z.z).normalized()
		var right := Vector3(basis.x.x, 0, basis.x.z).normalized()
		dir = (right * input.x + forward * -input.y).normalized()

	var target := dir * move_speed
	velocity.x = move_toward(velocity.x, target.x, acceleration * delta * move_speed)
	velocity.z = move_toward(velocity.z, target.z, acceleration * delta * move_speed)
	move_and_slide()

	# Oriente le modèle vers la direction de marche.
	if dir.length() > 0.1:
		var want := atan2(dir.x, dir.z)
		_model.rotation.y = lerp_angle(_model.rotation.y, want, rotation_speed * delta)

func _update_interactable() -> void:
	var nearest: Node = null
	var best := interact_range
	for node in get_tree().get_nodes_in_group("interactable"):
		if node is Node3D:
			var d := global_position.distance_to((node as Node3D).global_position)
			if d < best:
				best = d
				nearest = node

	if nearest != _current_interactable:
		_current_interactable = nearest
		if nearest and nearest.has_method("get_prompt"):
			_prompt.text = "[E] " + str(nearest.call("get_prompt"))
			_prompt.visible = true
			SignalBus.interactable_entered.emit(nearest)
		else:
			_prompt.visible = false
			SignalBus.interactable_exited.emit(nearest)

func _unhandled_input(event: InputEvent) -> void:
	# Pendant un dialogue, c'est l'UI qui gère la touche interagir.
	if GameManager.input_locked:
		return
	if event.is_action_pressed("interact") and _current_interactable:
		if _current_interactable.has_method("interact"):
			# On consomme l'événement pour que l'UI ne saute pas la 1re réplique.
			get_viewport().set_input_as_handled()
			_current_interactable.call("interact")
