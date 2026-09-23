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
| Carte du monde | **La vraie géographie de la Côte d'Ivoire**, avec des **noms de villages inventés** — Décidé |
| Énergie des ninjas | **Le Souffle** (remplace le chakra) — Décidé |
| Mudras | Restent des **signes de la main** — Décidé |
| Surnaturel | **Place importante** : esprits, lieux mythiques, forces anciennes — Décidé |
| Époque | **Monde hybride** : du mythique au futuriste — Décidé |
| Organisation du monde | **7 régions × 3 villages** (traditionnel, moderne, futuriste) — Décidé |
| Antagonistes | **Trois organisations secrètes** extérieures aux villages — Décidé |
| Économie | **Portée par les joueurs**, notamment par la vente des jutsus découverts — Décidé |
| Monnaie | **Le Djê** — Décidé |
| Plateforme | **Jeu PC** — Décidé |
| Rendu | **Combat** au tour par tour avec des décors à la Darkest Dungeon ; **déplacements sur la carte** façon shinobi.fr — Décidé |
| Code | **On repart de zéro** (sans réutiliser `game.js` ni `mmorpg/`), dans un **nouveau dépôt** — Décidé |
| Graphismes | Pas de dessinateur : **style simple et élégant** — Décidé |
| Rôles | **Terence dirige la conception, Claude développe** — Décidé |
| Niveau maximum | **100** — Décidé |

### Ce qu'on garde de shinobi.fr (le cœur du jeu)
- Des **jutsus complexes**.
- Des **combats stratégiques**.
- La **rivalité entre villages**.

### Inspirations et ce qu'on en retient

| Jeu | Ce qu'on en retient pour Ninja Ivoire |
|---|---|
| **Travian** | Monde persistant, rythme asynchrone, alliances, conquête de territoires, serveurs en saisons |
| **Darkest Dungeon** | Positions en combat (avant/arrière), moral et stress, blessures durables, prise de risque |
| **War Robots** | Équipement modulaire (« loadout ») préparé avant le combat, améliorations |
| **Dota 2** | Rôles d'équipe, contres et synergies, lecture de l'adversaire, combos entre coéquipiers |

## 2. Combat (Décidé dans les grandes lignes, détails à préciser)

1. **Tour simultané :** les deux camps choisissent leurs actions en secret, puis le tour se résout en même temps. Tout repose sur la lecture de l'adversaire : anticiper, feinter, contrer.
2. **Mudras :** un jutsu est une suite de signes de la main. Plus la suite est longue, plus le jutsu est puissant, mais plus il prend de tours et plus il peut être **interrompu**.
3. **Interactions élémentaires :** voir la table des forces et faiblesses (§4). Certaines combinaisons entre coéquipiers créent des **jutsus combinés**.
4. **Positions :** avant, milieu et arrière. La portée des jutsus et les rôles d'équipe en dépendent.
5. **Loadout :** on emporte un nombre limité de jutsus, d'outils et de parchemins.
6. **Le Souffle et la fatigue :** une ressource à gérer pendant le combat et d'une mission à l'autre.

## 3. Mudras et jutsus

### Règles décidées
- **Environ 50 mudras**, combinables entre eux.
- **L'ordre des signes compte** : les mêmes mudras dans un autre ordre donnent un autre jutsu, ou rien.
- **Monter de niveau débloque de nouveaux mudras.**
- **Des milliers de jutsus possibles.** La grande majorité est **inconnue du public** au départ.
- **Les combinaisons sont fixes** (pas de changement par saison ni par joueur), mais **les jutsus les plus puissants ne sont pas évidents à trouver**.
- **Un jutsu découvert reste dans la mémoire du ninja**, pour toujours.
- **Le premier découvreur peut vendre son jutsu** dans le jeu, après l'avoir fait **valider par l'académie de sa région**.
- **La grammaire des mudras** (ci-dessous) est validée.
- **Chaque jutsu s'améliore à l'usage** (niveau de maîtrise).
- **Un parchemin acheté à l'académie donne directement une maîtrise bien plus élevée** qu'une combinaison trouvée sur internet.
- **L'académie enregistre le jutsu au nom du découvreur** et prélève une **taxe qui finance la région du découvreur**.

### La grammaire des mudras (Décidé)
Chaque mudra a un **sens**, et une suite de mudras forme une **phrase**. Une phrase cohérente donne un jutsu. Certaines phrases précises cachent des **jutsus légendaires** faits à la main.

### Répartition des 50 mudras (Proposé)

| Catégorie | Nombre | Rôle |
|---|---|---|
| **Élément** | 16 | Un mudra par élément de base. Deux mudras d'éléments enchaînés, à très haut niveau, donnent un élément rare. |
| **Forme** | 10 | projectile, lame, mur, zone, clone, lien, armure, piège, invocation, déplacement |
| **Effet** | 10 | brûler, lier, soigner, aveugler, repousser, drainer, briser, dissimuler, renforcer, marquer |
| **Modificateur** | 6 | amplifier, étendre, multiplier, retarder, silence, persistance |
| **Mythique** | 8 | Un par élément mythique, obtenu uniquement par des quêtes |

Les noms des mudras s'inspirent de la faune et des contes ivoiriens (éléphant, panthère, crocodile, caméléon, calao, araignée Ananzè, tortue, python, hippopotame…).

### Combien de jutsus ? (calcul)

Structure d'une phrase : **Élément → Forme → Effet principal → (Effet secondaire) → (Modificateurs)**.
- Un modificateur placé après la Forme agit sur la forme (portée, taille). Placé après l'Effet, il agit sur l'effet (durée, intensité). **L'ordre compte donc vraiment.**
- L'ordre des deux effets compte aussi : le premier est l'effet principal, le second est plus faible.

| Niveau de détail | Calcul | Nombre |
|---|---|---|
| Familles de jutsus, éléments de base seulement | 16 éléments × 10 formes × 10 effets | **1 600** |
| Familles de jutsus, tous éléments (16 + 16 rares + 8 mythiques) | 40 × 10 × 10 | **4 000** |
| Avec un effet secondaire optionnel | 40 × 10 × (10 + 10×9) | **40 000** |
| Avec 0 à 2 modificateurs placés | 40 000 × 103 variantes | **≈ 4,1 millions** |
| Suites de 2 à 6 signes tapées au hasard avec 50 mudras | 50² + … + 50⁶ | ≈ 16 milliards |

- Il y a donc **plus de 4 000 jutsus réellement différents** dans leur nature, **40 000** avec les effets secondaires, et des **millions de variantes**.
- **Environ 1 suite de signes sur 4 000 tapée au hasard est valide.** Sans la résonance et la logique de la grammaire, on ne trouve rien : la découverte se fait par la réflexion.
- En plus, une **centaine de jutsus légendaires** sont conçus à la main.
- Un ninja ne maîtrise que 6 ou 7 éléments (§4). Chacun ne peut donc explorer qu'une partie de l'arbre. Le reste s'obtient par l'**échange** et le **commerce**.

### Découverte (Proposé)
- **Résonance :** un essai raté indique, par l'intensité du Souffle, si l'on s'approche d'une combinaison valide.
- **Risque :** un essai raté coûte du Souffle. Un essai très raté peut se retourner contre le lanceur.
- **Chroniques :** le nom du premier découvreur d'un jutsu légendaire est inscrit dans l'histoire du serveur.
- **Conditions cachées :** les jutsus les plus puissants demandent plus que la bonne suite de signes, par exemple un niveau minimum, la maîtrise d'un élément, un lieu, une heure, une phase de lune. Une suite divulguée sur internet ne suffit donc pas à les lancer.
- **Maîtrise :** chaque jutsu connu a un niveau de maîtrise qui progresse à l'usage (puissance, coût en Souffle, vitesse de lancement).

### Validation et vente à l'académie (Décidé ; détails Proposés)
1. Le découvreur présente son jutsu à l'**académie de sa région**, qui le teste et l'enregistre à son nom.
2. Il fixe un prix en Djê et vend des **parchemins d'enseignement**. Chaque parchemin apprend le jutsu à l'acheteur avec une **maîtrise de départ élevée**.
3. L'académie prélève une **taxe**, qui finance la **région du découvreur**.
4. Le jutsu apparaît dans le **catalogue** de l'académie (nom et effets visibles, suite de mudras cachée).

## 4. Éléments

### Acquisition par le ninja (Décidé)
- **Niveau maximum : 100.**
- **1 élément de départ**, donné par la **région** du ninja.
- **1 élément au choix tous les 20 niveaux** (niveaux 20, 40, 60, 80, 100), soit **6 éléments au total**.
- **Chaque jutsu légendaire connu** permet d'apprendre **un élément de plus**, au choix.
- **On ne peut pas acheter le parchemin d'un jutsu dont on ne maîtrise pas l'élément.**
- Une fusion (élément rare) exige de maîtriser les **deux** éléments qui la composent.

### Élément de départ par région (Proposé)

| Région | Élément de départ | Attribut favorisé |
|---|---|---|
| Lagunes | Eau | Manhis (agilité) |
| Côte Ouest | Végétal | Gnanga (technique) |
| Montagnes | Foudre | Manhis (agilité) |
| Hautes Savanes | Vent | Fangan (force) |
| Savanes du Nord | Feu | Fangan (force) |
| Levant | Terre | Gnanga (technique) |

### 16 éléments de base (Proposé)

Chaque élément est **fort contre deux éléments** et **faible contre deux autres**. Chacun a un horizon d'utilité : **court terme** (effet immédiat), **moyen terme** (contrôle sur plusieurs tours), **long terme** (effet qui grandit avec le temps, en combat ou hors combat).

| Élément | Rôle principal | Horizon | Fort contre | Faible contre |
|---|---|---|---|---|
| **Feu** | Gros dégâts, brûlure | Court | Végétal, Essaim | Eau, Sable |
| **Eau** | Polyvalence, soins, changement de forme | Moyen | Feu, Sable | Terre, Foudre |
| **Vent** | Vitesse, initiative, portée, dévie les projectiles | Court | Brume, Essaim | Venin, Gravité |
| **Terre** | Murs, armure, fortifications de territoire | Long | Foudre, Eau | Végétal, Métal |
| **Foudre** | Perce les défenses, paralysie, agit en premier | Court | Eau, Métal | Terre, Sable |
| **Végétal** | Pièges, entraves, croissance à chaque tour, récoltes | Long | Terre, Sable | Feu, Métal |
| **Métal** | Armes, armures, forge et artisanat | Moyen / long | Végétal, Terre | Foudre, Son |
| **Son** | Interrompt les mudras adverses, détection, tambours qui renforcent l'équipe | Moyen | Métal, Lune | Brume, Venin |
| **Sable** | Aveuglement, érosion des défenses, pièges | Moyen | Feu, Foudre | Eau, Végétal |
| **Venin** | Dégâts sur la durée, cumulatifs, affaiblissement | Long | Son, Vent | Sel, Soleil |
| **Brume** | Cache ses actions à l'adversaire, esquive, infiltration et espionnage | Moyen | Son, Sel | Vent, Soleil |
| **Sel** | Purification : annule les effets, soigne les altérations, repousse les esprits, conserve les ressources | Moyen / long | Venin, Lune | Brume, Gravité |
| **Essaim** | Nuées d'insectes, éclaireurs sur la carte, harcèlement qui grossit | Long | Soleil, Gravité | Feu, Vent |
| **Soleil** | Puissance selon l'heure réelle (max à midi, décidé), aveuglement, recharge le Souffle des alliés | Cyclique | Brume, Venin | Lune, Essaim |
| **Lune** | Illusions, sommeil, puissance la nuit et selon les phases lunaires (décidé) | Cyclique | Soleil, Gravité | Son, Sel |
| **Gravité** | Déplace les ennemis entre les rangs, ralentit, écrase | Court / moyen | Vent, Sel | Lune, Essaim |

### 16 éléments rares, par fusion à très haut niveau (Proposé)

| Fusion | Élément rare | Idée |
|---|---|---|
| Feu + Terre | **Lave** | Dégâts et terrain brûlant durable |
| Eau + Vent | **Glace** | Gel, entrave, armure de glace |
| Feu + Eau | **Vapeur** | Brûlure et dissimulation |
| Foudre + Métal | **Magnétisme** | Désarme, attire et repousse le métal, neutralise la technologie |
| Vent + Foudre | **Tempête** | Dégâts de zone sur tout le terrain |
| Sel + Soleil | **Cristal** | Renvoie les jutsus, stocke du Souffle |
| Venin + Eau | **Acide** | Détruit armures et équipements |
| Végétal + Soleil | **Bois sacré** | Soins massifs, croissance explosive |
| Son + Vent | **Onde de choc** | Repousse et étourdit tout un rang |
| Feu + Vent | **Cendre** | Aveugle et étouffe sur la durée |
| Soleil + Lune | **Crépuscule** | Change le cycle jour/nuit du combat |
| Brume + Lune | **Songe** | Illusions profondes, contrôle de l'esprit |
| Essaim + Venin | **Fléau** | Épidémie qui se propage d'un ennemi à l'autre |
| Terre + Gravité | **Séisme** | Brise les positions et les fortifications |
| Eau + Lune | **Marée** | Vagues qui montent de tour en tour |
| Sable + Vent | **Harmattan** | Tempête de poussière qui affaiblit tout le camp adverse |

### 8 éléments mythiques, par quêtes surnaturelles (Proposé)

| Élément | Idée |
|---|---|
| **Ivoire** | L'élément légendaire qui donne son nom au jeu : le Souffle originel, blanc et pur |
| **Esprit** | Lien avec les ancêtres et les génies |
| **Lumière** | Révélation, guérison, jugement |
| **Ombre** | Absorption, disparition, peur |
| **Vie** | Régénération, résurrection |
| **Temps** | Accélérer, ralentir, rejouer un tour |
| **Vide** | Effacer un jutsu, annuler un élément |
| **Astre** | Puissance cosmique, liée aux étoiles et aux éclipses |

Chaque élément mythique est lié à un **lieu mythique** et à un **mythe**, et s'obtient par une longue quête.

## 5. Le ninja

### Appartenance (Décidé)
- **Impossible de changer de village ou de région** une fois le ninja créé.

### Progression (Décidé)
- On gagne de l'**expérience** en combattant des **PNJ**, des **joueurs**, et en accomplissant des **missions**.
- **Certaines zones ne sont accessibles qu'à partir de certains niveaux.**

### Défaite hors siège (Décidé)
- Le ninja **perd tous ses objets** et **renaît dans son village**.

### Trois attributs (Décidé ; effets Proposés)
Chaque région a un **avantage dans un attribut** (voir §4).

| Attribut | Sens | Effets proposés |
|---|---|---|
| **Fangan** | Force | Dégâts des armes, points de vie, résistance aux coups, port des armures lourdes |
| **Gnanga** | Technique | Puissance des jutsus, réserve de Souffle, **nombre de mudras formés par tour** (un ninja technique lance plus vite les longs jutsus) |
| **Manhis** | Agilité | Initiative, esquive, coups critiques, changement de rang en combat |

Les jutsus **influencent** le combat ; les attributs et l'équipement en sont la base.

### Équipement (Décidé ; détails Proposés)

| Emplacement | Exemples |
|---|---|
| **Arme principale** | sabre, lance, dagues jumelles, bâton, gantelets, arc ; lames à énergie pour les villages futuristes |
| **Arme secondaire** | couteaux de lancer, sarbacane, bouclier, fumigènes, drone, talisman |
| **Armure de tête** | masque, casque, bandeau, visière tactique |
| **Armure de corps** | tunique, cuirasse, combinaison renforcée |
| **Armure de pieds** | sandales, bottes, jambières |

- Chaque pièce donne des **bonus de combat** et favorise un attribut.
- Le style de l'équipement dépend du type de village (traditionnel, moderne, futuriste).
- L'équipement est **fabriqué et vendu par les joueurs** (Proposé), dans la logique d'une économie portée par eux.

## 6. Le monde : la Côte d'Ivoire

### 7 régions (Décidé : 6 en guerre et 1 centrale sûre ; découpage Proposé)

| Région (nom provisoire) | Géographie réelle | Paysage |
|---|---|---|
| **Lagunes** | Sud-Est : Abidjan, Grand-Bassam, Assinie, Aboisso | Littoral, lagunes, mangroves |
| **Côte Ouest** | Sud-Ouest : San-Pédro, Sassandra, Soubré, forêt de Taï | Forêt primaire, côte sauvage, port |
| **Montagnes** | Ouest : Man, Danané, Touba, mont Nimba | Montagnes, cascades, forêts d'altitude |
| **Hautes Savanes** | Nord-Ouest : Odienné, Séguéla, Mankono | Savane arborée, plateaux |
| **Savanes du Nord** | Nord : Korhogo, Ferkessédougou, Boundiali, Kong | Savane, collines, cités anciennes |
| **Levant** | Est : Bondoukou, Bouna, Abengourou, parc de la Comoé | Forêt et savane, grande réserve sauvage |
| **Cœur** (zone sûre) | Centre : Yamoussoukro, Bouaké, lac de Kossou | Terre neutre, lacs, fleuve Bandama |

- **Les six régions périphériques** se font la guerre.
- **Le Cœur** est une zone sûre : pas de combat entre joueurs, grand marché, arène des examens, conseil entre régions.
- **Principe :** les six régions s'affrontent toutes entre elles. Il n'y a pas de bloc Nord contre Sud.

### Le Cœur (Décidé)
- **Trois villages aux habitants neutres**, uniquement des **PNJ** : aucun joueur n'y naît.
- Des **lieux introuvables dans les six autres régions**.
- Aucun combat entre joueurs.

### 3 villages par région (Décidé : 21 villages ; avantages Proposés)

Chaque région a un village **traditionnel**, un **moderne** et un **futuriste**. Aucune région n'est donc plus développée qu'une autre.

| Type | Avantages | Inconvénients |
|---|---|---|
| **Traditionnel** | Souffle plus puissant, accès privilégié aux lieux mythiques et aux esprits, bonus de découverte des jutsus, soins par les plantes | Peu d'équipement, économie et déplacements plus lents |
| **Moderne** | Équilibre, commerce, infrastructures, formation plus rapide, bonus de production | Pas d'excellence dans un domaine précis |
| **Futuriste** | Gadgets, implants, drones, renseignement sur la carte, production rapide | Souffle affaibli par la technologie, vulnérable à la Foudre et au Magnétisme, entretien coûteux, peu d'accès au surnaturel |

### Lieux mythiques (Proposé)
Le cœur de la forêt de Taï, le sommet du mont Nimba, les profondeurs des lagunes, le parc de la Comoé, les cascades de Man, le lac de Kossou…

### Principes de respect culturel
- On s'**inspire** des cultures (masques, sociétés d'initiation, tissus, royaumes, contes) sans les caricaturer, en **fictionnalisant** les objets sacrés.
- Les villages ne correspondent **pas** à des ethnies réelles, et aucun n'est « le méchant ».
- On évite de rejouer des conflits réels récents.

## 7. La guerre entre régions (Décidé)

- **Ce sont les régions qui se font la guerre.** Les trois villages d'une région sont alliés.
- Chaque région est découpée en **8 zones** qui donnent des **bonus** à qui les occupe.

### Déroulement d'un assaut
1. **Chaque vendredi soir**, les **portails** des régions deviennent attaquables.
2. Chaque portail est gardé par un **PNJ très puissant**. Il faut être bien équipé et bien organisé pour le battre.
3. Une fois le garde battu, une **annonce** est faite dans la région attaquée, qui dispose d'**une heure pour se préparer** et envoyer ses soldats.
4. Les assaillants **choisissent la zone** qu'ils veulent assiéger.
5. Si l'assaut réussit, la zone passe à la région attaquante, qui **s'agrandit** et récupère les **bonus de la zone**.
6. **Une seule zone peut être prise par week-end.**
7. **Au maximum deux régions** peuvent attaquer une même région.
8. Le siège se joue en **combats d'équipes simultanés**.

### Réinitialisation
- Une région peut **perdre toutes ses zones**.
- Dans ce cas, **après deux semaines d'occupation, la carte globale se réinitialise**. Aucune région ne garde trop longtemps les avantages des zones d'une autre.

## 8. Les trois organisations secrètes (Décidé, noms provisoires)

| Organisation | Philosophie | Ce qu'elle veut | Style |
|---|---|---|---|
| **Le Cercle d'Acier** | La technologie doit remplacer le Souffle | Contrôler les villes futuristes et créer des ninjas cybernétiques | Implants, drones, Métal et Foudre |
| **Les Sans-Visage** | Libérer les forces anciennes | Réveiller des esprits scellés et maîtriser les éléments mythiques | Rituels, possession, Ombre |
| **La Main d'Or** | Tout s'achète | Contrôler les richesses (or, cacao, ports) et manipuler les régions entre elles | Espions, assassins, corruption |

## 9. Technique (Proposé)

- **Client PC** : Godot 4 (gratuit, excellent en 2D, export Windows / Mac / Linux).
- **Serveur autoritatif** : obligatoire. Les **recettes de jutsus ne doivent jamais être dans le client**, sinon elles seraient extraites en quelques heures.
- **Base de données** : PostgreSQL.

## 10. Questions ouvertes

- Taille des équipes pendant un siège, et comment on gagne un siège.
- Défaite : les objets perdus vont-ils au vainqueur ? L'équipement porté est-il inclus ?
- Lignées.
- Monétisation.
