extends SpringArm3D
## Caméra 3e personne qui suit le joueur en douceur et pivote à la souris
## (clic droit maintenu). Le SpringArm gère la collision caméra/décor.

@export var target_path: NodePath
@export var follow_lerp: float = 8.0
@export var mouse_sensitivity: float = 0.005
@export var min_pitch: float = -0.9
@export var max_pitch: float = 0.35

var _target: Node3D
var _yaw: float = 0.0
var _pitch: float = -0.35

func _ready() -> void:
	_target = get_node_or_null(target_path)
	rotation = Vector3(_pitch, _yaw, 0)

func _unhandled_input(event: InputEvent) -> void:
	if event is InputEventMouseMotion and Input.is_mouse_button_pressed(MOUSE_BUTTON_RIGHT):
		_yaw -= event.relative.x * mouse_sensitivity
		_pitch = clampf(_pitch - event.relative.y * mouse_sensitivity, min_pitch, max_pitch)

func _process(delta: float) -> void:
	if _target:
		global_position = global_position.lerp(
			_target.global_position + Vector3.UP * 1.4, follow_lerp * delta)
	rotation.y = _yaw
	rotation.x = _pitch
