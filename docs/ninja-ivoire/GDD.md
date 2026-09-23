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
| Inspiration culturelle | **Côte d'Ivoire** : univers ninja d'inspiration ivoirienne et ouest-africaine — Décidé |
| Carte du monde | **La carte de la Côte d'Ivoire** — Décidé |

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

## 2. Concept de combat (Décidé dans les grandes lignes, détails à préciser)

1. **Tour simultané :** les deux camps choisissent leurs actions en secret, puis le tour se résout en même temps. Tout repose sur la lecture de l'adversaire : anticiper, feinter, contrer.
2. **Mudras (signes des mains) :** un jutsu est une suite de signes. Plus la suite est longue, plus le jutsu est puissant, mais plus il prend de tours et plus il peut être **interrompu**.
3. **Interactions élémentaires :** le vent attise le feu, l'eau éteint le feu, la foudre se propage dans l'eau, la terre bloque la foudre… Certaines combinaisons entre coéquipiers créent des **jutsus combinés**.
4. **Positions :** avant, milieu et arrière. La portée des jutsus et les rôles d'équipe en dépendent.
5. **Loadout :** on emporte un nombre limité de jutsus, d'outils et de parchemins. La préparation avant le combat est déjà une décision stratégique.
6. **Chakra et fatigue :** une ressource à gérer sur la durée du combat, et d'une mission à l'autre.

## 3. Le monde : la Côte d'Ivoire (Proposé, à valider)

### Découpage de la carte
- La carte suit la géographie réelle : la côte et les lagunes au sud, la forêt au sud-ouest, les montagnes à l'ouest, la savane au nord, les quatre grands fleuves (Cavally, Sassandra, Bandama, Comoé).
- **Territoires à conquérir :** les **31 régions** du pays, plus les 2 districts autonomes (Abidjan, Yamoussoukro). Chaque région produit des ressources et peut passer d'un village à l'autre.
- Les fleuves servent de frontières naturelles et de points stratégiques (gués, ponts, barrages).

### Cinq villages, cinq terres, cinq éléments

| Village (nom provisoire) | Région d'ancrage | Élément | Identité |
|---|---|---|---|
| Village des Lagunes | Sud-Est : lagune Ébrié, Grand-Bassam | Eau | Commerce, marine, pêcheurs, ruse |
| Village de la Forêt | Sud-Ouest : forêt de Taï | Terre | Pisteurs, poisons, plantes médicinales |
| Village des Montagnes | Ouest : Man, Dent de Man, mont Nimba | Foudre | Masques, échassiers, acrobates, cascades |
| Village de la Savane | Nord : Korhogo, parc de la Comoé | Feu | Initiés, feux de brousse, chasseurs |
| Village du Levant | Est : Bondoukou, Abengourou | Vent | Cités anciennes, lettrés, diplomates, harmattan |

### Zones spéciales
- **Yamoussoukro / le Centre (lac de Kossou, Bandama) :** zone neutre. Grand marché, arène des examens, conseil entre villages.
- **Parc de la Comoé :** grande zone sauvage et dangereuse (PvE).
- **Mont Nimba :** sommet légendaire, contenu de fin de jeu.
- **Forêt de Taï, cœur profond :** secrets, esprits, lignées perdues.

### Principes de respect culturel
- On s'**inspire** des cultures (masques, sociétés d'initiation, tissus, royaumes, contes) sans les caricaturer, en **fictionnalisant** les objets sacrés.
- Les villages ne correspondent **pas** à des ethnies réelles, et aucun n'est « le méchant ».
- On évite de rejouer des conflits réels récents (par exemple une guerre Nord contre Sud).

## 4. Questions ouvertes

- Format et plateforme (navigateur, mobile, PC), rythme de jeu.
- Réutilisation du serveur Go existant (`mmorpg/`).
- Personnage : clans et lignées, statistiques, éléments, spécialisations, mort.
- Monde : nombre de villages, déserteurs, politique, guerres, saisons.
- Monétisation.
