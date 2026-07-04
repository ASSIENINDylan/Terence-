extends CanvasLayer
## Affichage tête haute du village : nom du lieu, or, aide aux commandes.

@onready var _gold: Label = $Root/TopRight/Gold
@onready var _help: Label = $Root/BottomRight/Help

func _ready() -> void:
	_gold.text = "Or : %d" % GameManager.player_gold
	_help.text = "Z Q S D : se déplacer   ·   Clic droit : caméra   ·   E : interagir"
	SignalBus.battle_ended.connect(func(_w): _gold.text = "Or : %d" % GameManager.player_gold)
