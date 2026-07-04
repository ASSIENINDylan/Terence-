extends CanvasLayer
## Panneau de dialogue. Affiche les répliques une à une ; le joueur avance
## avec E / Espace. Bloque les entrées de jeu tant qu'un dialogue est ouvert.

@onready var _panel: Control = $Panel
@onready var _name_label: Label = $Panel/Margin/VBox/NameLabel
@onready var _text_label: Label = $Panel/Margin/VBox/TextLabel
@onready var _hint: Label = $Panel/Margin/VBox/Hint

var _open := false

func _ready() -> void:
	_panel.visible = false
	SignalBus.dialogue_started.connect(_on_started)
	SignalBus.dialogue_line_shown.connect(_on_line)
	SignalBus.dialogue_finished.connect(_on_finished)

func _on_started(_npc_name: String) -> void:
	_open = true
	_panel.visible = true
	GameManager.input_locked = true

func _on_line(speaker: String, text: String) -> void:
	_name_label.text = speaker
	_text_label.text = text
	_hint.text = "▼  E / Espace"

func _on_finished() -> void:
	_open = false
	_panel.visible = false
	GameManager.input_locked = false

func _unhandled_input(event: InputEvent) -> void:
	if not _open:
		return
	if event.is_action_pressed("interact") or event.is_action_pressed("ui_confirm_action"):
		get_viewport().set_input_as_handled()
		SignalBus.dialogue_advanced.emit()
