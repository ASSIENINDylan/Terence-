# Val-des-Brumes — RPG 3D (Godot 4)

Un petit RPG **3D** jouable sous **Godot 4.3+** : on explore un village en vue
3e personne, on parle aux habitants, et on affronte les monstres dans un
**combat au tour par tour inspiré de Monster Hunter Stories** — avec son fameux
**triangle « tête-à-tête »** :

```
Puissance  ▶ bat ▶  Technique  ▶ bat ▶  Vitesse  ▶ bat ▶  Puissance
```

Tout le rendu est **procédural** (géométrie + matériaux, éclairage directionnel
avec ombres, SSAO/SSIL, glow, ciel procédural, brouillard) : **aucune image ni
modèle externe** n'est requis, le projet tourne tel quel.

## Ouvrir le projet

1. Installer **Godot 4.3** (ou plus récent) — https://godotengine.org
2. *Godot → Importer* → sélectionner `godot/project.godot`
3. Lancer la scène principale (**F5**). L'écran-titre s'ouvre : **Jouer**.

## Commandes

| Contexte | Touche | Action |
|---|---|---|
| Village | `Z Q S D` / flèches | Se déplacer (touches **physiques** : marche aussi en QWERTY = WASD) |
| Village | Clic droit + souris | Pivoter la caméra |
| Village | `E` | Parler / interagir |
| Dialogue | `E` / `Espace` | Réplique suivante |
| Combat | Boutons | Choisir **Puissance / Vitesse / Technique**, cocher *Capacité* |

## Architecture

Le code est découplé par un **bus de signaux** (`SignalBus`) : aucun système ne
dépend directement d'un autre. Les données (dialogues, monstres, rencontres)
vivent dans des **singletons** dédiés, et un `GameManager` orchestre les
transitions village ⇆ combat.

```
godot/
├── project.godot            Config, autoloads, entrées
├── icon.svg
├── scenes/
│   ├── Main.tscn            Écran-titre
│   ├── Village.tscn         Monde (construit par VillageWorld.gd)
│   ├── Player.tscn          Personnage + caméra 3e personne
│   ├── NPC.tscn             Habitant interactif
│   ├── Building.tscn        Bâtiment procédural (Forge / Apothicairerie)
│   ├── WildMonster.tscn     Monstre déclencheur de combat
│   ├── combat/BattleScene.tscn
│   └── ui/                  HUD, DialogueUI, BattleUI
└── scripts/
    ├── autoload/            SignalBus, GameManager, DialogueDatabase,
    │                        MonsterDatabase, BattleManager
    ├── world/              VillageWorld (génère le village), Building
    ├── player/            Player (déplacement), PlayerCamera (spring arm)
    ├── npc/               NPC (dialogue), WildMonster (combat)
    ├── combat/            BattleTypes (triangle), Battler (état), BattleScene
    └── ui/                MainMenu, HUD, DialogueUI, BattleUI
```

### Boucle de jeu

1. **Village** (`VillageWorld.gd`) : instancie sol + décor, deux bâtiments,
   trois habitants et un monstre. Le joueur circule et interagit.
2. **Dialogue** : un PNJ émet ses répliques via `SignalBus`; l'UI les affiche.
   Certains PNJ (le Garde) **déclenchent un combat** à la fin.
3. **Combat** (`BattleManager.gd`) : chaque round, joueur et IA choisissent un
   **type d'attaque**. Quand deux combattants se visent, le **duel** se résout
   par le triangle : le gagnant inflige des dégâts majorés (×1.6) et encaisse
   peu. La vitesse fixe l'ordre d'action.
4. Victoire/défaite → retour au village (`GameManager`), la position du joueur
   est restaurée, l'or mis à jour.

## Étendre facilement

- **Un habitant** : ajouter une entrée dans `DialogueDatabase.DIALOGUES`, puis
  `_add_npc("mon_id", couleur, position)` dans `VillageWorld.gd`.
- **Un monstre / une rencontre** : compléter `MonsterDatabase.MONSTERS` et
  `ENCOUNTERS`.
- **Un bâtiment** : ajouter une valeur à l'enum `Building.Kind` et son preset.
