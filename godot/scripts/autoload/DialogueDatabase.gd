extends Node
## Base de données des dialogues, indexée par id de PNJ.
## Chaque dialogue = liste de répliques { "speaker": String, "text": String }.
## Certains PNJ peuvent déclencher un combat à la fin du dialogue.

const DIALOGUES := {
	"forgeron": {
		"display_name": "Bertram le Forgeron",
		"lines": [
			{ "speaker": "Bertram", "text": "Bienvenue à Val-des-Brumes, voyageur." },
			{ "speaker": "Bertram", "text": "Ma forge tourne jour et nuit... mais les monstres rôdent près des remparts." },
			{ "speaker": "Bertram", "text": "Tu veux te faire la main ? Va donc défier le molosse à l'entrée du village." },
		],
		"triggers_battle": "",
	},
	"apothicaire": {
		"display_name": "Dame Ysole",
		"lines": [
			{ "speaker": "Ysole", "text": "Approche, n'aie pas peur de mes fioles." },
			{ "speaker": "Ysole", "text": "Une potion avant le combat n'a jamais tué personne. Au contraire." },
			{ "speaker": "Ysole", "text": "On dit qu'un lien avec un monstre rend plus fort... apprivoise-les !" },
		],
		"triggers_battle": "",
	},
	"garde": {
		"display_name": "Garde Halbert",
		"lines": [
			{ "speaker": "Halbert", "text": "Halte ! Tu prétends protéger le village ?" },
			{ "speaker": "Halbert", "text": "Prouve-le. Affronte mon monstre en duel, ici et maintenant !" },
		],
		"triggers_battle": "duel_garde",
	},
}

func has_dialogue(npc_id: String) -> bool:
	return DIALOGUES.has(npc_id)

func get_lines(npc_id: String) -> Array:
	if DIALOGUES.has(npc_id):
		return DIALOGUES[npc_id]["lines"]
	return [{ "speaker": "???", "text": "..." }]

func get_display_name(npc_id: String) -> String:
	if DIALOGUES.has(npc_id):
		return DIALOGUES[npc_id]["display_name"]
	return npc_id

func get_battle_trigger(npc_id: String) -> String:
	if DIALOGUES.has(npc_id):
		return DIALOGUES[npc_id].get("triggers_battle", "")
	return ""
