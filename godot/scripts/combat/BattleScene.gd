extends Node3D
## Arène de combat. Construit un décor 3D soigné, place les modèles des
## combattants (générés par code), lance le BattleManager et gère la fin de
## bataille (retour au village via GameManager).

const BattleUIScene := preload("res://scenes/ui/BattleUI.tscn")

func _ready() -> void:
	_build_arena()
	var enc := MonsterDatabase.get_encounter(GameManager.pending_encounter)

	# Alliés = équipe du joueur.
	var ally_defs: Array = []
	for mid in GameManager.party:
		ally_defs.append(MonsterDatabase.get_monster(mid))
	# Ennemis de la rencontre.
	var enemy_defs: Array = []
	for mid in enc["enemies"]:
		enemy_defs.append(MonsterDatabase.get_monster(mid))

	_spawn_models(ally_defs, false)
	_spawn_models(enemy_defs, true)

	add_child(BattleUIScene.instantiate())

	BattleManager.battle_over.connect(_on_battle_over)
	SignalBus.battle_message.emit(enc["title"])
	# Laisse une frame à l'UI pour se connecter avant de démarrer.
	await get_tree().process_frame
	BattleManager.start(ally_defs, enemy_defs)

func _on_battle_over(player_won: bool) -> void:
	SignalBus.battle_message.emit("Victoire !" if player_won else "Défaite...")
	await get_tree().create_timer(1.8).timeout
	SignalBus.battle_ended.emit(player_won)

# ------------------------------------------------------------------ décor 3D
func _build_arena() -> void:
	var env := WorldEnvironment.new()
	env.environment = _make_environment(Color("#20304a"), 0.9)
	add_child(env)

	var sun := DirectionalLight3D.new()
	sun.rotation_degrees = Vector3(-55, 40, 0)
	sun.light_energy = 1.2
	sun.light_color = Color("#ffe6c0")
	sun.shadow_enabled = true
	add_child(sun)

	# Sol circulaire (arène)
	var ground := MeshInstance3D.new()
	var cyl := CylinderMesh.new()
	cyl.top_radius = 9.0
	cyl.bottom_radius = 9.0
	cyl.height = 0.4
	ground.mesh = cyl
	var gmat := StandardMaterial3D.new()
	gmat.albedo_color = Color("#3a4a3a")
	gmat.roughness = 0.9
	ground.material_override = gmat
	ground.position = Vector3(0, -0.2, 0)
	add_child(ground)

	# Caméra fixe légèrement en plongée
	var cam := Camera3D.new()
	cam.position = Vector3(0, 4.5, 9)
	cam.rotation_degrees = Vector3(-22, 0, 0)
	cam.fov = 55
	cam.current = true
	add_child(cam)

func _spawn_models(defs: Array, enemy: bool) -> void:
	var n := defs.size()
	for i in range(n):
		var def: Dictionary = defs[i]
		var m := _make_battler_model(def["color"])
		var x := (i - (n - 1) * 0.5) * 2.6
		var z := -3.5 if enemy else 3.5
		m.position = Vector3(x, 0, z)
		if enemy:
			m.rotate_y(PI)
		add_child(m)
		# Animation « idle » (le nœud est maintenant dans l'arbre).
		var body: Node3D = m.get_node("Body")
		var tw := m.create_tween().set_loops()
		tw.tween_property(body, "position:y", 0.95, 0.9).set_trans(Tween.TRANS_SINE)
		tw.tween_property(body, "position:y", 0.8, 0.9).set_trans(Tween.TRANS_SINE)

		var tag := Label3D.new()
		tag.text = def["name"]
		tag.position = Vector3(x, 2.2, z)
		tag.billboard = BaseMaterial3D.BILLBOARD_ENABLED
		tag.font_size = 40
		tag.pixel_size = 0.008
		add_child(tag)

func _make_battler_model(color: Color) -> Node3D:
	var root := Node3D.new()
	var body := MeshInstance3D.new()
	body.name = "Body"
	var mesh := CapsuleMesh.new()
	mesh.radius = 0.5
	mesh.height = 1.6
	body.mesh = mesh
	body.position = Vector3(0, 0.8, 0)
	var mat := StandardMaterial3D.new()
	mat.albedo_color = color
	mat.roughness = 0.45
	mat.metallic = 0.15
	body.material_override = mat
	root.add_child(body)
	return root

func _make_environment(bg: Color, glow: float) -> Environment:
	var e := Environment.new()
	e.background_mode = Environment.BG_COLOR
	e.background_color = bg
	e.ambient_light_source = Environment.AMBIENT_SOURCE_COLOR
	e.ambient_light_color = Color("#6a7a95")
	e.ambient_light_energy = 0.6
	e.tonemap_mode = Environment.TONE_MAPPER_ACES
	e.glow_enabled = true
	e.glow_intensity = glow
	e.glow_bloom = 0.2
	e.ssao_enabled = true
	e.fog_enabled = true
	e.fog_light_color = bg
	e.fog_density = 0.02
	return e
