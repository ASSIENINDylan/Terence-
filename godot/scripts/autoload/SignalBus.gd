extends Node
## Bus de signaux global.
## Permet aux systèmes (joueur, PNJ, UI, combat) de communiquer sans se
## connaître directement : chacun émet/écoute sur ce bus.

# --- Exploration / dialogue ---
signal interactable_entered(target: Node)   ## Le joueur est à portée d'un objet interactif
signal interactable_exited(target: Node)    ## Le joueur quitte la portée
signal dialogue_started(npc_name: String)
signal dialogue_line_shown(speaker: String, text: String)
signal dialogue_advanced()   ## L'UI demande la réplique suivante (touche E/Espace)
signal dialogue_finished()   ## Fin du dialogue : l'UI ferme le panneau

# --- Transitions ---
signal battle_requested(encounter_id: String)  ## Un PNJ/monstre déclenche un combat
signal battle_ended(player_won: bool)

# --- Combat (tour par tour) ---
signal battle_started(allies: Array, enemies: Array)
signal turn_started(battler)               ## Battler dont c'est le tour
signal action_resolved(log_line: String)   ## Ligne à afficher dans le journal de combat
signal battler_hp_changed(battler)
signal battler_died(battler)
signal battle_message(text: String)        ## Message d'ambiance / bandeau
