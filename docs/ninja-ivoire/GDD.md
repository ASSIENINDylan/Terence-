# Ninja Ivoire — Document de conception (GDD)

> Document vivant, construit au fil des séances de questions/réponses.
> **Décidé** = validé par Terence. **Proposé** = piste de Claude, à valider.

## 1. Vision

| Sujet | Décision |
|---|---|
| Nom | **Ninja Ivoire** — Décidé |
| Univers | **Univers original** inspiré des ninjas (pas de licence Naruto) — Décidé |
| Ton | **Mélange** : épique, sombre et parfois léger selon les arcs — Décidé |
| Piliers | **Stratégie · Progression · Communauté** — Décidé |
| Modèle de référence | shinobi.fr, en plus moderne, plus complexe, plus intelligent — Décidé |

### Ce qu'on garde de shinobi.fr (le cœur du jeu)
- Des **jutsus complexes**.
- Des **combats stratégiques**.
- La **rivalité entre villages**.

### Inspirations et ce qu'on en retient (Proposé)

| Jeu | Ce qu'on en retient pour Ninja Ivoire |
|---|---|
| **Travian** | Monde persistant, rythme asynchrone, alliances, conquête de territoires, serveurs en saisons |
| **Darkest Dungeon** | Positions en combat (avant/arrière), moral et stress, blessures durables, prise de risque |
| **War Robots** | Équipement modulaire (« loadout ») préparé avant le combat, améliorations |
| **Dota 2** | Rôles d'équipe, contres et synergies, lecture de l'adversaire, combos entre coéquipiers |

## 2. Concept de combat (Proposé, à valider)

1. **Tour simultané :** les deux camps choisissent leurs actions en secret, puis le tour se résout en même temps. Tout repose sur la lecture de l'adversaire : anticiper, feinter, contrer.
2. **Mudras (signes des mains) :** un jutsu est une suite de signes. Plus la suite est longue, plus le jutsu est puissant, mais plus il prend de tours et plus il peut être **interrompu**.
3. **Interactions élémentaires :** le vent attise le feu, l'eau éteint le feu, la foudre se propage dans l'eau, la terre bloque la foudre… Certaines combinaisons entre coéquipiers créent des **jutsus combinés**.
4. **Positions :** avant, milieu et arrière. La portée des jutsus et les rôles d'équipe en dépendent.
5. **Loadout :** on emporte un nombre limité de jutsus, d'outils et de parchemins. La préparation avant le combat est déjà une décision stratégique.
6. **Chakra et fatigue :** une ressource à gérer sur la durée du combat, et d'une mission à l'autre.

## 3. Questions ouvertes

- Sens de « Ivoire » : référence à la Côte d'Ivoire (univers d'inspiration africaine) ou à l'ivoire comme matière/couleur ?
- Format et plateforme (navigateur, mobile, PC), rythme de jeu.
- Réutilisation du serveur Go existant (`mmorpg/`).
- Personnage : clans et lignées, statistiques, éléments, spécialisations, mort.
- Monde : nombre de villages, déserteurs, politique, guerres, saisons.
- Monétisation.
