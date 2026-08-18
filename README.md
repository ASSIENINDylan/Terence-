# 忍 Shinobi no Yuukan

Un **jeu de ninja par navigateur** inspiré de [shinobi.fr](https://www.shinobi.net/) :
on incarne un shinobi de l'un des **villages cachés** du pays de **Yuukan**, on
**explore la carte**, on améliore ses caractéristiques (**taijutsu, ninjutsu,
genjutsu**), on **apprend des jutsus**, on **équipe** son ninja, on prend des
**missions** et on combat au **tour par tour**.

Le jeu reprend la **dynamique de déplacement** clé de shinobi.fr : on se déplace
de zone en zone sur la carte, **chaque pas consomme du chakra** (qui régénère au
repos), et **s'aventurer dans les zones sauvages déclenche des rencontres**.

Tout est **100 % local, sans dépendance et sans build** : peinture procédurale
sur `<canvas>`, aucune image ni bibliothèque externe. Il suffit d'ouvrir le
fichier `index.html`.

## Jouer

Double-cliquez sur **`index.html`** (ou glissez-le dans un navigateur). Aucune
installation, aucun serveur.

La **première page est un écran de connexion / inscription** (comptes locaux,
stockés dans le navigateur). À l'inscription, on crée son personnage :
**nom**, **apparence personnalisable** (peau, tenue, cheveux, bandeau) et
**village d'appartenance**. Une reconnexion reprend directement la partie.

- **Se déplacer** : `Z Q S D` (ou `W A S D`) / flèches
- **Courir** : `Maj` (plus rapide, mais consomme plus de chakra)
- **Entrer dans un village** : `E` (ou `Espace`) quand on est dessus
- **Fiche du ninja & jutsus** : `C`
- **Sauvegarder** : bouton *Sauvegarder* (la partie est aussi sauvegardée
  automatiquement après chaque combat, achat, montée de niveau…)

La progression est stockée dans le navigateur (`localStorage`) — le bouton
**Continuer** de l'écran-titre reprend la dernière partie.

## Le pays de Yuukan

La carte est **générée** (reproductible via une graine) : plaines, forêts,
montagnes, désert, lacs et **routes** qui relient les villages. On y trouve les
**trois villages cachés** — et un **quatrième, secret**, à découvrir en explorant.

| Village | Kanji | Spécialité | Style |
|---|---|---|---|
| **Zambakro** | 力 | Taijutsu | Le corps à corps, la Force |
| **Abidjan** | 魔 | Ninjutsu | Les sceaux et les éléments |
| **Akradjo** | 幻 | Genjutsu | L'illusion, briser l'esprit |
| *Yami* | 闇 | *secret* | *le village de l'Ombre, caché* |

## Compte et personnage

- **Connexion / inscription** en première page. Les comptes sont **locaux**
  (`localStorage`) — le mot de passe n'est que haché, ce n'est pas une sécurité
  réseau, juste un profil par joueur sur le navigateur.
- À l'**inscription**, on choisit **nom**, **apparence** et **village**.
- L'**apparence** est un **portrait de face personnalisable** : forme du
  **visage**, **peau**, **coiffure** et **couleur de cheveux**, **yeux** et
  **couleur des yeux**, **nez**, **bouche**, **bandeau** et **tenue**. Elle se
  personnalise à la création et se **modifie à tout moment** depuis la fiche du
  ninja (`C` → *Apparence*), et se reflète sur la carte et en combat.

## Dynamique de déplacement (le cœur du jeu)

- Le monde est une **grille de zones**. On avance case par case ; le rendu
  interpole pour une marche fluide, la caméra suit le ninja.
- **Chaque pas coûte du chakra**, selon le terrain : route `1`, plaine `2`,
  forêt `3`, désert `4`, montagne `5`. Les villages sont gratuits.
- Le **chakra régénère** — vite à l'arrêt, lentement en marchant. À sec, on ne
  peut plus traverser les terrains coûteux : il faut se **reposer** (au village
  ou en revenant sur une route).
- **L'eau est infranchissable** ; les **routes** sont sûres et rapides.
- Marcher dans une **zone sauvage** a une chance de déclencher une **rencontre**
  qui bascule en combat. Une brève **immunité** suit chaque combat.

## Combat au tour par tour

L'ordre de départ dépend de la **vitesse**. À votre tour :

- **Taijutsu** — attaque de base, gratuite, gouvernée par le Taijutsu.
- **Jutsu** — puise dans le **chakra**, gouverné par la stat associée
  (nin/gen/tai). Effets : dégâts, **soin**, **paralysie**, baisse de défense
  ennemie, boost de vitesse.
- **Objet** — potions de soin / pilules de chakra.
- **Fuir** — réussite selon la vitesse ; échouer coûte le tour.

Victoire ⇒ **XP + ryō** (+ butin éventuel) et progression de mission. Défaite ⇒
réveil à votre village, un peu de ryō en moins.

## Progression

- **XP → niveau** : chaque niveau augmente PV et chakra max, remet en forme, et
  donne **3 points de caractéristique** à répartir (Taijutsu / Ninjutsu /
  Genjutsu / Vitesse) au **dojo d'entraînement** du village.
- **Grades** selon le niveau : Genin → Chunin → Jonin → Anbu → Kage.
- **Boutique** : armes (+taijutsu), armures (+PV/défense), bandeaux (+chakra),
  et consommables. Les objets ont un **niveau requis**.
- **Jutsus** : chaque village enseigne son propre répertoire, appris contre des
  ryō (niveau requis selon la puissance).
- **Missions** : contrats de chasse (rangs D → B) pour votre village, récompensés
  en XP et ryō.

## Menus du village (touche `E`)

Sur une tuile de village, `E` ouvre le hub : **Repos** (soin complet),
**Entraînement** (répartir les points), **Boutique**, **Jutsus** et **Missions**.

## Structure du code

Aucun bundler : des scripts classiques partageant l'espace de noms global `SH`,
chargés dans l'ordre des dépendances par `index.html`.

```
index.html          # page + HUD + ordre de chargement des scripts
css/style.css       # thème sombre, HUD, fenêtres modales, combat
js/
  rng.js            # générateur pseudo-aléatoire déterministe (mulberry32)
  data.js           # contenu : villages, jutsus, monstres, objets, terrains, missions
  world.js          # génération de la carte du pays de Yuukan
  state.js          # état du joueur : stats dérivées, XP, inventaire, sauvegarde
  combat.js         # moteur de combat au tour par tour
  ui.js             # HUD, menus de village, fiche perso, fenêtre de combat
  render.js         # rendu canvas (carte + sprites procéduraux)
  game.js           # boucle de jeu, déplacement/chakra, rencontres, entrées
```

## Développement

Le jeu n'a **aucune dépendance d'exécution**. Pour vérifier la logique hors
navigateur (génération de carte, combat, progression, sauvegarde), un harnais de
test Node stubbe `window`/`localStorage` et charge les modules non-DOM — voir
l'historique de développement. La syntaxe de chaque fichier se vérifie avec
`node --check js/<fichier>.js`.
