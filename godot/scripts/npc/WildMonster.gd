class_name WildMonster
extends StaticBody3D
## Monstre sauvage posté dans le village. Interagir déclenche un combat.
## Le modèle (corps + yeux) est généré et coloré par code.

@export var encounter_id: String = "molosse_remparts"
@export var monster_color: Color = Color("#e05a4d")
@export var display_name: String = "Molosse des Remparts"

func _ready() -> void:
	add_to_group("interactable")
	_build()

func get_prompt() -> String:
	return "Affronter " + display_name

func interact() -> void:
	SignalBus.battle_message.emit(display_name + " se dresse devant vous !")
	SignalBus.battle_requested.emit(encounter_id)

func _build() -> void:
	# Corps
	var body := MeshInstance3D.new()
	var body_mesh := SphereMesh.new()
	body_mesh.radius = 0.7
	body_mesh.height = 1.2
	body.mesh = body_mesh
	body.position = Vector3(0, 0.7, 0)
	var mat := StandardMaterial3D.new()
	mat.albedo_color = monster_color
	mat.roughness = 0.5
	mat.metallic = 0.1
	body.material_override = mat
	add_child(body)

	# Collision
	var col := CollisionShape3D.new()
	var shape := SphereShape3D.new()
	shape.radius = 0.8
	col.shape = shape
	col.position = Vector3(0, 0.7, 0)
	add_child(col)

	# Yeux lumineux
	for sx in [-0.25, 0.25]:
		var eye := MeshInstance3D.new()
		var eye_mesh := SphereMesh.new()
		eye_mesh.radius = 0.12
		eye_mesh.height = 0.24
		eye.mesh = eye_mesh
		eye.position = Vector3(sx, 1.05, 0.55)
		var emat := StandardMaterial3D.new()
		emat.albedo_color = Color("#fff2b0")
		emat.emission_enabled = true
		emat.emission = Color("#ffd257")
		emat.emission_energy_multiplier = 2.5
		eye.material_override = emat
		add_child(eye)
