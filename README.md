# Shadow of the Warrior

Un petit jeu d'aventure et de combat au tour par tour, **entièrement dessiné au
code** (pixel-art via `<canvas>`, aucune image ni bibliothèque externe).

## Trois façons de jouer

1. **Dans le navigateur** (le plus simple) : ouvre `index.html` d'un double-clic.
2. **En application de bureau** : `npm install` puis `npm start` (lance Electron).
3. **En vrai `.exe` Windows** : voir ci-dessous.

## Obtenir l'exécutable `.exe` Windows

Le `.exe` est compilé automatiquement par GitHub Actions (sur une machine Windows,
car cela ne peut pas se faire depuis Linux). Deux options :

- **Télécharger le `.exe` prêt à l'emploi**
  1. Va dans l'onglet **Actions** du dépôt sur GitHub.
  2. Ouvre le dernier run réussi **« Build Windows .exe »** (tu peux aussi le
     lancer à la main via **Run workflow**).
  3. En bas, télécharge l'artefact **`ShadowOfTheWarrior-Windows`** : il contient
     `ShadowOfTheWarrior-1.0.0-portable.exe`, un exécutable **portable**
     (double-clic, aucune installation).
  - Astuce : pousse un tag `v1.0.0` (`git tag v1.0.0 && git push origin v1.0.0`)
    et le `.exe` sera aussi publié dans une **Release** GitHub.

- **Compiler soi-même sur une machine Windows**
  ```bash
  npm install
  npm run dist:win           # -> dist/ShadowOfTheWarrior-1.0.0-portable.exe
  # ou, pour un installateur :
  npm run dist:win-installer # -> dist/ShadowOfTheWarrior-1.0.0-Setup.exe
  ```

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
