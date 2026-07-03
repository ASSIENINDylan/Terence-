# Shadow of the Warrior

Un petit jeu d'aventure et de combat au tour par tour, **entièrement dessiné au
code** (pixel-art via `<canvas>`, aucune image ni bibliothèque externe). Il suffit
d'ouvrir `index.html` dans un navigateur pour jouer.

## Comment jouer

- **Déplacement** : `Z Q S D` ou les **flèches**
- **Interagir** (portes / marchands) : `E` ou `Entrée`
- **En combat** : `1` Attaquer · `2` Potion · `3` Fuir (ou `↑ ↓` + `E`)
- **En boutique** : `↑ ↓` pour choisir, `E` pour acheter, `X` pour sortir

## La carte

Le héros — un guerrier de l'ombre — se promène librement sur une carte et peut,
en s'approchant d'un bâtiment et en appuyant sur `E` :

- **Donjon** (en haut) → entrer pour **combattre** un ennemi et gagner or + XP.
- **Apothicaire** (à droite) → acheter des **potions** et des élixirs de stats.
- **Forgeron** (en bas) → acheter des **armes** et des **armures**.

## Effets spéciaux des statistiques

Les stats ne sont pas de simples nombres : chacune a un effet concret en combat.

| Stat        | Effet spécial                                                        |
|-------------|----------------------------------------------------------------------|
| **Force**   | Augmente l'**armure** (réduction des dégâts subis) et un peu les dégâts. |
| **Vitesse** | Augmente l'**esquive** (chance d'éviter totalement un coup) et la fuite. |

- Armure totale = défense de l'équipement + `Force × 0.6`
- Esquive = `Vitesse × 2.2 %` (plafonnée à 60 %)

## Progression

Chaque ennemi vaincu rapporte de l'or et de l'XP. À chaque niveau, les PV Max,
la Force et la Vitesse augmentent, et les ennemis deviennent plus coriaces.
En cas de défaite, le héros renaît en perdant la moitié de son or.
