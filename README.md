# Shadow of the Warrior

Un jeu d'aventure et de combat au tour par tour, **entièrement dessiné au code**
(canvas HTML5 : dégradés, lumières, particules, animations — aucune image ni
bibliothèque externe).

## Trois façons de jouer

1. **Dans le navigateur** (le plus simple) : ouvre `index.html` d'un double-clic.
2. **En application de bureau** : `npm install` puis `npm start` (Electron).
3. **En vrai `.exe` Windows** : voir plus bas.

## Comment jouer

- **Choix de classe** au lancement : `← →` puis `E`
- **Déplacement** : `Z Q S D` ou les **flèches**
- **Interagir** (portes / marchands) : `E` ou `Entrée`
- **En combat** : `1` Attaquer · `2` Capacité spéciale · `3` Potion · `4` Fuir
- **En boutique** : `↑ ↓` choisir, `E` acheter, `X` sortir

## La carte

- **Donjon** (au nord) → franchir la porte pour **combattre** (or + XP).
- **Forgeron** (en haut à l'ouest) → **armes** et **armures**.
- **Apothicaire** (à l'est) → **potions** et élixirs de stats permanents.

## Les classes (3 → 9 évolutions)

Chaque classe a ses **stats de base**, sa **capacité spéciale**, et évolue au
**niveau 5** en l'une de **3 classes avancées** :

| Classe | Profil | Capacité | Évolutions (niv. 5) |
|---|---|---|---|
| **Guerrier** | PV 60 · FOR 9 · VIT 3 · ESP 2 | Colère du Titan — 220% imparable | Berserker · Paladin · Seigneur de Guerre |
| **Lame des Ombres** | PV 45 · FOR 5 · VIT 9 · ESP 3 | Danse des Ombres — 3 frappes à 80% | Maître des Ombres · Duelliste · Traqueur |
| **Mage de Guerre** | PV 40 · FOR 3 · VIT 4 · ESP 10 | Boule de Feu — magie imparable | Archimage · Nécromancien · Sage |

Chaque évolution apporte des bonus de stats **et une nouvelle capacité**
(Rage Sanglante, Jugement Sacré, Étendard de Guerre, Voile d'Ombre, Estocade
Fulgurante, Lame Empoisonnée, Météore, Drain de Vie, Bénédiction).

## Effets spéciaux des statistiques

| Stat | Effets |
|---|---|
| **Force** | + armure (`FOR × 0.6`) et + dégâts (`FOR × 0.5`) |
| **Vitesse** | + esquive (`VIT × 2.2 %`, max 70) · chance de **double frappe** (`VIT × 1.2 %`) · + fuite |
| **Esprit** | + puissance des capacités magiques · + énergie max et régénération · petit soin après victoire |

Les valeurs dérivées sont affichées en direct dans le HUD.

## Obtenir l'application Windows

L'application est compilée automatiquement par GitHub Actions à chaque push,
sous forme d'un **`.zip` contenant l'application décompressée** (et non un exe
auto-extractible, qui déclenche souvent une fausse alerte antivirus) :

1. Onglet **Actions** du dépôt → dernier run **« Build Windows .exe »** vert.
2. Télécharger l'artefact **`ShadowOfTheWarrior-Windows`**.
3. Dézipper le dossier `ShadowOfTheWarrior-2.0.0-win64.zip`.
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
