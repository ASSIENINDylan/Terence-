class_name BattleTypes
extends RefCounted
## Constantes et règles du triangle d'attaque, cœur du système « tête-à-tête »
## inspiré de Monster Hunter Stories :
##   PUISSANCE  bat  TECHNIQUE
##   TECHNIQUE  bat  VITESSE
##   VITESSE    bat  PUISSANCE

enum Kind { PUISSANCE, VITESSE, TECHNIQUE }

const KIND_NAMES := {
	Kind.PUISSANCE: "Puissance",
	Kind.VITESSE: "Vitesse",
	Kind.TECHNIQUE: "Technique",
}

const KIND_COLORS := {
	Kind.PUISSANCE: Color(0.878, 0.353, 0.302),  # rouge
	Kind.VITESSE:   Color(0.302, 0.714, 0.878),  # bleu
	Kind.TECHNIQUE: Color(0.412, 0.784, 0.420),  # vert
}

## Renvoie 1 si `a` bat `b`, -1 si `a` perd, 0 en cas d'égalité (même type).
static func compare(a: int, b: int) -> int:
	if a == b:
		return 0
	match a:
		Kind.PUISSANCE:
			return 1 if b == Kind.TECHNIQUE else -1
		Kind.VITESSE:
			return 1 if b == Kind.PUISSANCE else -1
		Kind.TECHNIQUE:
			return 1 if b == Kind.VITESSE else -1
	return 0

static func kind_name(k: int) -> String:
	return KIND_NAMES.get(k, "?")

static func kind_color(k: int) -> Color:
	return KIND_COLORS.get(k, Color.WHITE)
