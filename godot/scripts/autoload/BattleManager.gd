extends Node
## Moteur du combat tour par tour, style « tête-à-tête » de Monster Hunter Stories.
##
## Boucle d'un round :
##   1. Chaque camp choisit une action (type d'attaque + capacité ou non).
##      -> le joueur via l'UI, l'ennemi via l'IA.
##   2. On résout dans l'ordre de vitesse. Quand deux combattants s'attaquent
##      mutuellement, c'est un DUEL : le triangle de types décide qui l'emporte.
##      Le gagnant inflige des dégâts majorés et encaisse peu ; l'égalité de type
##      fait mal aux deux.
##
## BattleManager est autonome : il pilote la logique, émet des signaux pour l'UI,
## et attend le choix du joueur via `submit_player_action`.

signal request_player_action(battler)   ## Demande à l'UI le choix du joueur
signal battle_over(player_won: bool)

var allies: Array[Battler] = []
var enemies: Array[Battler] = []
var _running := false
var _pending_player_kind := BattleTypes.Kind.PUISSANCE
var _pending_player_skill := false

signal _player_action_submitted()

func start(ally_defs: Array, enemy_defs: Array) -> void:
	allies.clear()
	enemies.clear()
	for d in ally_defs:
		var b := Battler.new()
		b.setup(d, false)
		allies.append(b)
	for d in enemy_defs:
		var b := Battler.new()
		b.setup(d, true)
		enemies.append(b)
	SignalBus.battle_started.emit(allies, enemies)
	_running = true
	_run_loop()

## Appelé par l'UI quand le joueur a choisi son action pour le tour.
func submit_player_action(kind: int, use_skill: bool) -> void:
	_pending_player_kind = kind
	_pending_player_skill = use_skill
	_player_action_submitted.emit()

func _run_loop() -> void:
	while _running:
		# --- Phase de choix ---
		for ally in _living(allies):
			SignalBus.turn_started.emit(ally)
			request_player_action.emit(ally)
			await _player_action_submitted
			ally.chosen_kind = _pending_player_kind
			ally.use_skill = _pending_player_skill
		for enemy in _living(enemies):
			_ai_choose(enemy)

		# --- Phase de résolution (par vitesse décroissante) ---
		var order := _living(allies) + _living(enemies)
		order.sort_custom(func(a, b): return a.speed > b.speed)
		for actor in order:
			if not actor.is_alive():
				continue
			await _resolve_action(actor)
			if _check_end():
				return

		SignalBus.battle_message.emit("Nouveau tour !")
		await get_tree().create_timer(0.4).timeout

func _resolve_action(actor: Battler) -> void:
	var target := _pick_target(actor)
	if target == null:
		return

	# Duel ? (la cible attaque aussi l'attaquant ce tour-ci et est en vie)
	var is_duel := _targets_back(target, actor)
	var advantage := 0
	if is_duel:
		advantage = BattleTypes.compare(actor.chosen_kind, target.chosen_kind)

	var dmg := _compute_damage(actor, target, advantage)
	var dealt := target.take_damage(dmg)
	SignalBus.battler_hp_changed.emit(target)

	var kind_txt := BattleTypes.kind_name(actor.chosen_kind)
	var verb := actor.skill_name if actor.use_skill else "attaque " + kind_txt
	var duel_txt := ""
	if is_duel and advantage > 0:
		duel_txt = "  ⚔ DUEL GAGNÉ !"
	elif is_duel and advantage < 0:
		duel_txt = "  (duel perdu)"
	elif is_duel:
		duel_txt = "  (choc frontal)"
	SignalBus.action_resolved.emit("%s utilise %s sur %s : %d dégâts%s" % [
		actor.display_name, verb, target.display_name, dealt, duel_txt])

	if not target.is_alive():
		SignalBus.battler_died.emit(target)
		SignalBus.action_resolved.emit("%s est vaincu !" % target.display_name)

	await get_tree().create_timer(0.55).timeout

func _compute_damage(actor: Battler, target: Battler, advantage: int) -> int:
	var base: float = float(actor.attack)
	if actor.use_skill:
		base *= actor.skill_power
	# Le triangle de types module fortement les dégâts lors d'un duel.
	if advantage > 0:
		base *= 1.6      # on gagne le tête-à-tête
	elif advantage < 0:
		base *= 0.5      # on le perd
	var mitigation: float = float(target.defense) * 0.5
	var dmg: float = maxf(1.0, base - mitigation)
	dmg *= randf_range(0.9, 1.1)   # petite variance
	return int(round(dmg))

func _ai_choose(enemy: Battler) -> void:
	# IA simple : 70 % du temps elle joue son type favori, sinon aléatoire ;
	# capacité une fois de temps en temps.
	if randf() < 0.7:
		enemy.chosen_kind = enemy.kind
	else:
		enemy.chosen_kind = randi() % 3
	enemy.use_skill = randf() < 0.25

func _pick_target(actor: Battler) -> Battler:
	var pool := _living(enemies) if not actor.is_enemy else _living(allies)
	if pool.is_empty():
		return null
	return pool[randi() % pool.size()]

## Vrai si `target` vise `actor` ce tour (donc duel réciproque possible).
func _targets_back(target: Battler, actor: Battler) -> bool:
	if not target.is_alive():
		return false
	# Camps opposés forcément ; on considère qu'un combattant vivant du camp
	# adverse « riposte » et engage donc le duel de types.
	return target.is_enemy != actor.is_enemy

func _living(arr: Array) -> Array:
	return arr.filter(func(b): return b.is_alive())

func _check_end() -> bool:
	if _living(enemies).is_empty():
		_end(true)
		return true
	if _living(allies).is_empty():
		_end(false)
		return true
	return false

func _end(player_won: bool) -> void:
	_running = false
	battle_over.emit(player_won)
