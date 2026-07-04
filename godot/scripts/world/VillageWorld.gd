extends Node3D
## Construit le village « Val-des-Brumes » : ambiance/éclairage, sol et
## collisions, décor (arbres, rochers, chemin), puis instancie le joueur,
## les deux bâtiments, les habitants, le monstre et l'interface.

const PlayerScene := preload("res://scenes/Player.tscn")
const NPCScene := preload("res://scenes/NPC.tscn")
const BuildingScene := preload("res://scenes/Building.tscn")
const WildMonsterScene := preload("res://scenes/WildMonster.tscn")
const HUDScene := preload("res://scenes/ui/HUD.tscn")
const DialogueUIScene := preload("res://scenes/ui/DialogueUI.tscn")

func _ready() -> void:
	_build_atmosphere()
	_build_ground()
	_scatter_decor()
	_place_buildings()
	_place_inhabitants()
	_place_player()
	# Interfaces
	add_child(HUDScene.instantiate())
	add_child(DialogueUIScene.instantiate())

# --------------------------------------------------------- ambiance & lumière
func _build_atmosphere() -> void:
	var sky_mat := ProceduralSkyMaterial.new()
	sky_mat.sky_top_color = Color("#3f6bb5")
	sky_mat.sky_horizon_color = Color("#cfe3ff")
	sky_mat.ground_bottom_color = Color("#5a6b52")
	sky_mat.ground_horizon_color = Color("#cfe3ff")
	sky_mat.sun_angle_max = 30.0
	var sky := Sky.new()
	sky.sky_material = sky_mat

	var env := Environment.new()
	env.background_mode = Environment.BG_SKY
	env.sky = sky
	env.ambient_light_source = Environment.AMBIENT_SOURCE_SKY
	env.ambient_light_energy = 0.8
	env.tonemap_mode = Environment.TONE_MAPPER_ACES
	env.tonemap_white = 1.2
	env.glow_enabled = true
	env.glow_intensity = 0.5
	env.glow_bloom = 0.15
	env.ssao_enabled = true
	env.ssao_radius = 1.2
	env.ssil_enabled = true
	env.fog_enabled = true
	env.fog_light_color = Color("#c9d8e8")
	env.fog_density = 0.006
	env.fog_sky_affect = 0.2

	var we := WorldEnvironment.new()
	we.environment = env
	add_child(we)

	var sun := DirectionalLight3D.new()
	sun.rotation_degrees = Vector3(-48, 55, 0)
	sun.light_color = Color("#fff0d4")
	sun.light_energy = 1.15
	sun.shadow_enabled = true
	sun.directional_shadow_max_distance = 80.0
	add_child(sun)

# ------------------------------------------------------------------- le sol
func _build_ground() -> void:
	var floor_body := StaticBody3D.new()
	add_child(floor_body)

	var shape := CollisionShape3D.new()
	var box := BoxShape3D.new()
	box.size = Vector3(60, 1, 60)
	shape.shape = box
	shape.position = Vector3(0, -0.5, 0)
	floor_body.add_child(shape)

	var ground := MeshInstance3D.new()
	var plane := PlaneMesh.new()
	plane.size = Vector2(60, 60)
	ground.mesh = plane
	var gmat := StandardMaterial3D.new()
	gmat.albedo_color = Color("#5b7a44")
	gmat.roughness = 1.0
	ground.material_override = gmat
	floor_body.add_child(ground)

	# Chemin central (dalle claire)
	var path := MeshInstance3D.new()
	var pmesh := PlaneMesh.new()
	pmesh.size = Vector2(4, 40)
	path.mesh = pmesh
	var pmat := StandardMaterial3D.new()
	pmat.albedo_color = Color("#b6a184")
	pmat.roughness = 0.95
	path.material_override = pmat
	path.position = Vector3(0, 0.02, 0)
	add_child(path)

	# Muret d'enceinte simple (4 CSGBox)
	for data in [
		[Vector3(0, 1, -22), Vector3(48, 2, 1)],
		[Vector3(0, 1, 22), Vector3(48, 2, 1)],
		[Vector3(-24, 1, 0), Vector3(1, 2, 46)],
		[Vector3(24, 1, 0), Vector3(1, 2, 46)],
	]:
		var wall := CSGBox3D.new()
		wall.use_collision = true
		wall.size = data[1]
		wall.position = data[0]
		var wmat := StandardMaterial3D.new()
		wmat.albedo_color = Color("#8a8375")
		wmat.roughness = 0.9
		wall.material = wmat
		add_child(wall)

# -------------------------------------------------------------------- décor
func _scatter_decor() -> void:
	var rng := RandomNumberGenerator.new()
	rng.seed = 1337
	for i in range(18):
		var pos := Vector3(rng.randf_range(-20, 20), 0, rng.randf_range(-20, 20))
		# On évite le chemin et le centre.
		if absf(pos.x) < 4.0 or (absf(pos.x) < 9 and absf(pos.z) < 9):
			continue
		if i % 4 == 0:
			_add_rock(pos, rng)
		else:
			_add_tree(pos, rng)

func _add_tree(pos: Vector3, rng: RandomNumberGenerator) -> void:
	var tree := Node3D.new()
	tree.position = pos
	var h := rng.randf_range(2.4, 3.6)

	var trunk := MeshInstance3D.new()
	var tm := CylinderMesh.new()
	tm.top_radius = 0.22
	tm.bottom_radius = 0.32
	tm.height = h
	trunk.mesh = tm
	trunk.position = Vector3(0, h * 0.5, 0)
	trunk.material_override = _flat(Color("#6b4a2b"), 0.9)
	tree.add_child(trunk)

	for j in range(2):
		var leaves := MeshInstance3D.new()
		var sm := SphereMesh.new()
		var r := rng.randf_range(1.1, 1.6) - j * 0.35
		sm.radius = r
		sm.height = r * 2.0
		leaves.mesh = sm
		leaves.position = Vector3(0, h + j * 0.7, 0)
		var g := rng.randf_range(0.35, 0.55)
		leaves.material_override = _flat(Color(0.2, g, 0.22), 0.85)
		tree.add_child(leaves)
	add_child(tree)

func _add_rock(pos: Vector3, rng: RandomNumberGenerator) -> void:
	var rock := MeshInstance3D.new()
	var sm := SphereMesh.new()
	var r := rng.randf_range(0.5, 1.0)
	sm.radius = r
	sm.height = r * 1.4
	rock.mesh = sm
	rock.position = pos + Vector3(0, r * 0.4, 0)
	rock.scale = Vector3(1, 0.7, 1)
	rock.material_override = _flat(Color("#7d7d82"), 0.95)
	add_child(rock)

# --------------------------------------------------------------- bâtiments
func _place_buildings() -> void:
	var forge := BuildingScene.instantiate()
	forge.kind = Building.Kind.FORGE
	forge.position = Vector3(-9, 0, -6)
	forge.rotate_y(deg_to_rad(20))
	add_child(forge)

	var apo := BuildingScene.instantiate()
	apo.kind = Building.Kind.APOTHICAIRE
	apo.position = Vector3(9, 0, -6)
	apo.rotate_y(deg_to_rad(-20))
	add_child(apo)

# --------------------------------------------------------------- habitants
func _place_inhabitants() -> void:
	_add_npc("forgeron", Color("#b5651d"), Vector3(-6, 0, -3))
	_add_npc("apothicaire", Color("#6a8f6a"), Vector3(6, 0, -3))
	_add_npc("garde", Color("#4a5a8a"), Vector3(2.5, 0, 4))

	var molosse := WildMonsterScene.instantiate()
	molosse.encounter_id = "molosse_remparts"
	molosse.monster_color = Color("#e05a4d")
	molosse.display_name = "Molosse des Remparts"
	molosse.position = Vector3(-3, 0, 12)
	add_child(molosse)

func _add_npc(id: String, color: Color, pos: Vector3) -> void:
	var npc := NPCScene.instantiate()
	npc.npc_id = id
	npc.body_color = color
	npc.position = pos
	add_child(npc)

func _place_player() -> void:
	var player := PlayerScene.instantiate()
	add_child(player)
	# Player.gd restaure lui-même sa position via GameManager au _ready.

func _flat(color: Color, rough: float) -> StandardMaterial3D:
	var m := StandardMaterial3D.new()
	m.albedo_color = color
	m.roughness = rough
	return m
