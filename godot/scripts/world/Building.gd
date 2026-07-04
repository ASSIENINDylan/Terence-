class_name Building
extends Node3D
## Bâtiment généré procéduralement avec des CSG (murs, toit, porte, fenêtres).
## Deux presets sont fournis : la Forge et l'Apothicairerie.
## Aucune image externe : tout est de la géométrie + matériaux.

enum Kind { FORGE, APOTHICAIRE }

@export var kind: Kind = Kind.FORGE
@export var wall_color: Color = Color("#d8c4a0")
@export var roof_color: Color = Color("#8a3b2e")
@export var width: float = 6.0
@export var depth: float = 5.0
@export var wall_height: float = 3.2
@export var sign_text: String = "Forge"

func _ready() -> void:
	if kind == Kind.APOTHICAIRE:
		wall_color = Color("#c9d6c2")
		roof_color = Color("#4a6b8a")
		sign_text = "Apothicairerie"
	_build()

func _build() -> void:
	# --- Murs (bloc plein évidé par la porte) ---
	var walls := CSGCombiner3D.new()
	walls.use_collision = true
	add_child(walls)

	var box := CSGBox3D.new()
	box.size = Vector3(width, wall_height, depth)
	box.position = Vector3(0, wall_height * 0.5, 0)
	box.material = _mat(wall_color, 0.85)
	walls.add_child(box)

	# Porte (soustraction)
	var door := CSGBox3D.new()
	door.operation = CSGShape3D.OPERATION_SUBTRACTION
	door.size = Vector3(1.3, 2.2, 0.6)
	door.position = Vector3(0, 1.1, depth * 0.5)
	walls.add_child(door)

	# Fenêtres (soustractions)
	for sx in [-1.9, 1.9]:
		var win := CSGBox3D.new()
		win.operation = CSGShape3D.OPERATION_SUBTRACTION
		win.size = Vector3(0.9, 0.9, 0.6)
		win.position = Vector3(sx, 1.9, depth * 0.5)
		walls.add_child(win)

	# --- Toit (prisme via CSGPolygon extrudé) ---
	var roof := CSGPolygon3D.new()
	roof.polygon = PackedVector2Array([
		Vector2(-width * 0.5 - 0.3, 0),
		Vector2(0, 2.0),
		Vector2(width * 0.5 + 0.3, 0),
	])
	roof.depth = depth + 0.6
	roof.mode = CSGPolygon3D.MODE_DEPTH
	roof.position = Vector3(0, wall_height, -depth * 0.5 - 0.3)
	roof.material = _mat(roof_color, 0.7)
	add_child(roof)

	# --- Cadre de porte ---
	var frame := CSGBox3D.new()
	frame.size = Vector3(1.6, 2.5, 0.2)
	frame.position = Vector3(0, 1.25, depth * 0.5 + 0.05)
	frame.material = _mat(Color("#5a3b25"), 0.8)
	add_child(frame)

	# --- Enseigne ---
	var sign := Label3D.new()
	sign.text = sign_text
	sign.font_size = 64
	sign.pixel_size = 0.006
	sign.modulate = Color("#fff4d6")
	sign.outline_size = 12
	sign.outline_modulate = Color(0, 0, 0, 0.7)
	sign.position = Vector3(0, 2.6, depth * 0.5 + 0.2)
	sign.billboard = BaseMaterial3D.BILLBOARD_DISABLED
	add_child(sign)

	# --- Petite lueur à la porte (torche) ---
	var lamp := OmniLight3D.new()
	lamp.light_color = Color("#ffb457")
	lamp.light_energy = 2.2
	lamp.omni_range = 5.0
	lamp.position = Vector3(1.0, 2.2, depth * 0.5 + 0.3)
	add_child(lamp)

func _mat(color: Color, rough: float) -> StandardMaterial3D:
	var m := StandardMaterial3D.new()
	m.albedo_color = color
	m.roughness = rough
	m.metallic = 0.0
	return m
