# Shadow of the Warrior

Un action-RPG au tour par tour à l'ambiance gothique. Le **combat est rendu en
WebGL via PixiJS** (pénombre, halos de torche, ombres portées, brouillard,
étalonnage sombre et **giclées de sang**) pour évoquer *Darkest Dungeon* ; la
carte-hub et les menus sont rendus sur un canvas 2D. Tous les personnages sont
**peints procéduralement** (couches d'ombrage, contour-lumière) — aucune image
externe, tout est embarqué (PixiJS est vendorisé dans `libs/`), donc le jeu
tourne hors-ligne.

## Trois façons de jouer

1. **Dans le navigateur** : ouvre `index.html` (sert `game.js` + `libs/`).
2. **En application de bureau** : `npm install` puis `npm start` (Electron).
3. **En application Windows** : voir plus bas.

## Comment jouer

- **Choix de classe** : `← →` puis `E`
- **Déplacement sur la carte** : `Z Q S D` / flèches · **Interagir** : `E`
- **En combat** : `1` Attaquer · `2` Capacité · `3` Potion · `4` Fuir (ou `↑↓ E`)
- **En boutique** : `↑ ↓` choisir, `E` acheter, `X` sortir

## Les Cinq Portails

La carte-hub donne accès à **5 portails**, chacun de **10 étages** terminés par
un **boss unique**. La difficulté augmente avec le portail, l'étage et votre
niveau. Un portail se déverrouille en atteignant son niveau recommandé ou en
purifiant le précédent.

| Portail | Niv. | Gardien (boss) |
|---|---|---|
| Les Catacombes | 1 | Le Croque-Os |
| La Forêt Maudite | 6 | La Sylve Affamée |
| Le Sanctuaire Brisé | 11 | L'Inquisiteur Déchu |
| Les Abysses Gelés | 16 | Le Roi Gelé |
| Le Trône de Cendres | 22 | Malphas, l'Embrasé |

Entre deux étages, un **feu de camp** permet de se soigner un peu et de
continuer, ou de se replier au village (en gardant or et niveaux).

## Classes uniques (3 → 9 évolutions)

Chaque classe a des **stats de base**, une **capacité active**, un **passif
unique** et une apparence propre. Au **niveau 5**, elle évolue en l'une de
**3 classes avancées** (bonus de stats + nouveau passif + nouvelle capacité).

| Classe | Passif | Capacité | Évolutions |
|---|---|---|---|
| **Guerrier** | Peau de fer (−20% dégâts subis) | Colère du Titan | Berserker · Paladin · Seigneur de Guerre |
| **Lame des Ombres** | Célérité (chance de rejouer) | Danse des Ombres | Maître des Ombres · Duelliste · Traqueur |
| **Mage de Guerre** | Flux mystique (+énergie/tour) | Trait de Feu | Archimage · Nécromancien · Sage |

## Effets spéciaux des statistiques

| Stat | Effets |
|---|---|
| **Force** | + armure (`FOR × 0.6`) et + dégâts (`FOR × 0.55`) |
| **Vitesse** | + esquive (`VIT × 2.2 %`, max 72) · **double frappe** (`VIT × 1.2 %`) · + fuite |
| **Esprit** | + puissance magique · + énergie max & régénération · soin après victoire |

## Marchands qui évoluent

Le **forgeron** (armes/armures) et l'**apothicaire** (potions/élixirs)
**enrichissent leur stock à mesure que le héros monte en niveau** : de
nouveaux paliers d'équipement apparaissent, et les potions soignent davantage.

## Obtenir l'application Windows

L'application est compilée automatiquement par GitHub Actions à chaque push,
sous forme d'un **`.zip` contenant l'application décompressée** (et non un exe
auto-extractible, qui déclenche souvent une fausse alerte antivirus) :

1. Onglet **Actions** du dépôt → dernier run **« Build Windows .exe »** vert.
2. Télécharger l'artefact **`ShadowOfTheWarrior-Windows`**.
3. Dézipper le dossier `ShadowOfTheWarrior-3.0.0-win64.zip`.
4. Ouvrir le dossier et double-cliquer sur **`Shadow of the Warrior.exe`**
   (`F11` = plein écran).

Ou compiler soi-même sous Windows : `npm install && npm run dist:win`.
Pousser un tag `v2.0.0` publie aussi le `.zip` dans une **Release** GitHub.

### En cas d'alerte antivirus (faux positif)

L'application n'est pas signée numériquement (un certificat de signature de code
coûte cher), donc Windows SmartScreen ou un antivirus peut afficher un
avertissement. Le jeu est 100 % local (aucun accès réseau) — c'est un faux
positif dû à l'absence de signature. Solutions :

- **SmartScreen** : « Informations complémentaires » → « Exécuter quand même ».
- **Windows Defender** : *Sécurité Windows → Protection contre les virus →
  Gérer les paramètres → Exclusions* → ajouter le dossier du jeu.
- **Test immédiat sans exe** : ouvre simplement `index.html` dans un navigateur,
  le jeu est identique et ne peut être bloqué par aucun antivirus.
