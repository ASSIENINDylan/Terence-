extends Control
## Écran-titre. « Jouer » (ou Entrée) charge le village.

@onready var _play: Button = $Center/VBox/PlayButton

func _ready() -> void:
	_play.pressed.connect(_start)
	_play.grab_focus()

func _unhandled_input(event: InputEvent) -> void:
	if event.is_action_pressed("ui_accept"):
		_start()

func _start() -> void:
	get_tree().change_scene_to_file(GameManager.VILLAGE_SCENE)
