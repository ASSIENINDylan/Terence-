extends CanvasLayer
## Interface du combat tour par tour. Construite par code.
## - Barres de vie des alliés (bas-gauche) et ennemis (haut-droite).
## - Boutons d'action : Puissance / Vitesse / Technique + bascule « Capacité ».
## - Journal de combat déroulant + bandeau de messages.

var _enemy_box: VBoxContainer
var _ally_box: VBoxContainer
var _log: RichTextLabel
var _banner: Label
var _action_panel: PanelContainer
var _skill_check: CheckButton
var _turn_label: Label

var _bars := {}   # Battler -> ProgressBar

func _ready() -> void:
	layer = 10
	_build()
	SignalBus.battle_started.connect(_on_battle_started)
	SignalBus.battler_hp_changed.connect(_on_hp_changed)
	SignalBus.action_resolved.connect(_log_line)
	SignalBus.battle_message.connect(_show_banner)
	BattleManager.request_player_action.connect(_on_request_action)

# ---------------------------------------------------------------- construction
func _build() -> void:
	var root := Control.new()
	root.set_anchors_preset(Control.PRESET_FULL_RECT)
	root.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(root)

	_enemy_box = _make_side_box(root, true)
	_ally_box = _make_side_box(root, false)

	# Bandeau central
	_banner = Label.new()
	_banner.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_banner.set_anchors_preset(Control.PRESET_CENTER_TOP)
	_banner.position = Vector2(-200, 24)
	_banner.custom_minimum_size = Vector2(400, 40)
	_banner.add_theme_font_size_override("font_size", 22)
	_banner.modulate = Color("#fff1c9")
	root.add_child(_banner)

	# Journal
	_log = RichTextLabel.new()
	_log.set_anchors_preset(Control.PRESET_CENTER)
	_log.position = Vector2(-260, -180)
	_log.custom_minimum_size = Vector2(520, 120)
	_log.scroll_following = true
	_log.add_theme_color_override("default_color", Color("#e8e8e8"))
	root.add_child(_log)

	# Label « à qui le tour »
	_turn_label = Label.new()
	_turn_label.set_anchors_preset(Control.PRESET_CENTER_BOTTOM)
	_turn_label.position = Vector2(-200, -150)
	_turn_label.custom_minimum_size = Vector2(400, 24)
	_turn_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	root.add_child(_turn_label)

	# Panneau d'action
	_action_panel = PanelContainer.new()
	_action_panel.set_anchors_preset(Control.PRESET_BOTTOM_WIDE)
	_action_panel.position = Vector2(0, -110)
	_action_panel.custom_minimum_size = Vector2(0, 100)
	root.add_child(_action_panel)

	var hb := HBoxContainer.new()
	hb.alignment = BoxContainer.ALIGNMENT_CENTER
	hb.add_theme_constant_override("separation", 16)
	_action_panel.add_child(hb)

	for kind in [BattleTypes.Kind.PUISSANCE, BattleTypes.Kind.VITESSE, BattleTypes.Kind.TECHNIQUE]:
		var b := Button.new()
		b.text = BattleTypes.kind_name(kind)
		b.custom_minimum_size = Vector2(150, 60)
		b.add_theme_color_override("font_color", BattleTypes.kind_color(kind))
		b.add_theme_font_size_override("font_size", 22)
		b.pressed.connect(_on_action_button.bind(kind))
		hb.add_child(b)

	_skill_check = CheckButton.new()
	_skill_check.text = "Capacité"
	hb.add_child(_skill_check)

	_set_actions_enabled(false)

func _make_side_box(root: Control, enemy: bool) -> VBoxContainer:
	var box := VBoxContainer.new()
	box.add_theme_constant_override("separation", 8)
	if enemy:
		box.set_anchors_preset(Control.PRESET_TOP_RIGHT)
		box.position = Vector2(-320, 24)
	else:
		box.set_anchors_preset(Control.PRESET_BOTTOM_LEFT)
		box.position = Vector2(24, -220)
	box.custom_minimum_size = Vector2(300, 0)
	root.add_child(box)
	return box

# --------------------------------------------------------------------- signaux
func _on_battle_started(allies: Array, enemies: Array) -> void:
	_bars.clear()
	for b in allies:
		_add_bar(_ally_box, b)
	for b in enemies:
		_add_bar(_enemy_box, b)

func _add_bar(box: VBoxContainer, battler) -> void:
	var row := VBoxContainer.new()
	var lbl := Label.new()
	lbl.text = "%s  [%s]" % [battler.display_name, BattleTypes.kind_name(battler.kind)]
	lbl.add_theme_color_override("font_color", battler.color)
	row.add_child(lbl)
	var bar := ProgressBar.new()
	bar.max_value = battler.max_hp
	bar.value = battler.hp
	bar.custom_minimum_size = Vector2(300, 18)
	bar.show_percentage = false
	row.add_child(bar)
	box.add_child(row)
	_bars[battler] = bar

func _on_hp_changed(battler) -> void:
	if _bars.has(battler):
		var bar: ProgressBar = _bars[battler]
		var tween := create_tween()
		tween.tween_property(bar, "value", battler.hp, 0.3)

func _log_line(text: String) -> void:
	_log.append_text(text + "\n")

func _show_banner(text: String) -> void:
	_banner.text = text
	_banner.modulate.a = 1.0
	var tw := create_tween()
	tw.tween_interval(1.4)
	tw.tween_property(_banner, "modulate:a", 0.0, 0.6)

func _on_request_action(battler) -> void:
	_turn_label.text = "Tour de %s — choisissez votre type d'attaque" % battler.display_name
	_set_actions_enabled(true)

func _on_action_button(kind: int) -> void:
	_set_actions_enabled(false)
	_turn_label.text = ""
	BattleManager.submit_player_action(kind, _skill_check.button_pressed)
	_skill_check.button_pressed = false

func _set_actions_enabled(on: bool) -> void:
	for c in _action_panel.get_child(0).get_children():
		if c is Button:
			c.disabled = not on
