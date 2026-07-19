# MMORPG de stratégie persistante — Prototype v0.1

Serveur autoritatif d'un MMORPG 2D de stratégie persistante (factions, territoires,
saisons). Ce dossier héberge le **prototype v0.1** construit en suivant le plan de
build `T0 → T8` du plan technique.

- **Client** : Godot 4 (2D) — à venir (T2+).
- **Serveur** : Go, autoritatif intégral.
- **Stockage** : PostgreSQL (vérité durable) + Redis (état volatile, sessions, pub/sub).
- **Transport** : WebSocket (`wss://`) — à venir (T2).

> Principe non négociable : le client n'affiche que ce que le serveur a validé.
> Aucune logique de jeu ne vit dans le client.

## Plan de build (§13 du plan technique)

| Étape | Livrable | État |
|-------|----------|------|
| **T0** | Squelette : repo, docker compose (postgres+redis), première migration, health check. | ✅ **fait** |
| **T1** | Comptes + login HTTP + token de session Redis. | ✅ **fait** |
| **T2** | WebSocket + Gateway : handshake authentifié, écho. | ✅ **fait** |
| **T3** | Personnage persistant + entrée en zone + `zone.snapshot`. | ✅ **fait** |
| **T4** | Boucle de tick + `move.intent` → position serveur → `zone.delta`. | ✅ **fait** |
| **T5** | Combat autoritatif au tour par tour, déclenché par la rencontre. | ✅ **fait** |
| T6 | Inventaire : templates + exemplaires, ramasser/équiper/utiliser. | à venir |
| T7 | Chat (global + faction) via pub/sub Redis + appartenance village. | à venir |
| T8 | Persistance robuste : write-back périodique + flush à la déconnexion + métriques. | à venir |

## T0 — ce qui est livré

- Structure de dépôt Go (`server/cmd`, `server/internal/*`, `server/migrations`).
- `docker-compose.yml` : services `postgres`, `redis`, `server`.
- Première migration `0001_init.sql` : le schéma PostgreSQL v0.1 complet (§6) —
  comptes, factions, villages, zones, personnages, objets (catalogue + exemplaires),
  territoires, saisons, clans, chat, et table `outbox` (reliable outbox).
- Runner de migration interne (versionné, transactionnel, idempotent).
- Health check HTTP : `GET /healthz` (liveness) et `GET /readyz` (DB + Redis).

**Critère de validation T0** : *« Le serveur démarre, se connecte à la DB et à Redis. »* ✅

## T1 — ce qui est livré

Authentification des joueurs (§6), branchée sur le squelette T0 :

- Mots de passe **hachés (bcrypt)**, jamais stockés ni journalisés en clair.
- **Tokens de session opaques** (256 bits, base64url) stockés côté serveur dans
  Redis (`session:<token>`) avec une durée de vie configurable (`SESSION_TTL`).
- Unicité d'email **insensible à la casse** (type `citext`).
- **Anti-énumération** : « email inconnu » et « mauvais mot de passe » renvoient
  la même erreur `invalid_credentials`, avec un temps de réponse uniformisé.
- Découpage en couches : `auth` (règles) → `store` (comptes durables) + `cache`
  (sessions volatiles), le tout testé par des faux en mémoire (`go test ./...`).

### Endpoints

| Méthode & route | Corps / en-tête | Réponse |
|-----------------|-----------------|---------|
| `POST /auth/register` | `{"email","password"}` | `201 {account_id, email}` — `409 email_taken`, `400 invalid_email` / `weak_password` |
| `POST /auth/login` | `{"email","password"}` | `200 {token, token_type, expires_in, account_id}` — `401 invalid_credentials` |
| `GET /auth/me` | `Authorization: Bearer <token>` | `200 {account_id}` — `401 invalid_token` |
| `POST /auth/logout` | `Authorization: Bearer <token>` | `204` (idempotent) |
| `GET /ws` | `?token=` ou `Authorization: Bearer` | Upgrade WebSocket (T2) — `401` si token absent/invalide |

```bash
# Créer un compte, se connecter, récupérer le token, l'utiliser.
curl -X POST localhost:8080/auth/register -d '{"email":"a@b.com","password":"s3cr3t-pass"}'
TOKEN=$(curl -s -X POST localhost:8080/auth/login \
  -d '{"email":"a@b.com","password":"s3cr3t-pass"}' | jq -r .token)
curl localhost:8080/auth/me -H "Authorization: Bearer $TOKEN"   # {"account_id":1}
```

**Critère de validation T1** : *« Un joueur peut créer un compte, se connecter,
et obtenir un token de session vérifiable stocké dans Redis. »* ✅ — vérifié de
bout en bout (register → login → `/me` → logout révoque le token).

## T2 — ce qui est livré

La gateway temps réel (§3, §5) : une connexion WebSocket authentifiée, avec
écho.

- **Handshake authentifié** : `GET /ws` valide le token de session (réutilise
  `auth.Authenticate` de T1) **avant** l'upgrade. Token absent ou invalide ⇒
  `401`, aucune socket ouverte. Token accepté via en-tête `Authorization: Bearer`
  (clients natifs) ou paramètre `?token=` (navigateurs).
- **Enveloppe de protocole** (`internal/protocol`) : tout message est un
  `{type, seq, data}` JSON. La (dé)sérialisation est isolée pour pouvoir passer
  au binaire plus tard sans toucher à la logique.
- **Abstraction `Conn`** (`internal/gateway`) : un `writePump` unique possède
  l'écriture, un `readPump` la lecture, keepalive ping/pong et coupure des
  consommateurs lents. C'est la brique que T3+ utilisera pour pousser snapshots
  et deltas.
- **Écho** (comportement T2) : tout message valide revient en `echo` (seq et
  data préservés) ; `ping` → `pong` ; message mal formé → `error`.

```
Client ──ws://…/ws?token=<token>──▶ Gateway
       ◀── {"type":"auth.ok","data":{"account_id":1}}
       ── {"type":"chat.say","seq":9,"data":{"body":"hi"}} ──▶
       ◀── {"type":"echo","seq":9,"data":{"body":"hi"}}
```

**Critère de validation T2** : *« Un client peut se connecter en WebSocket avec
un token valide et échanger des messages ; un token invalide est rejeté. »* ✅ —
vérifié de bout en bout (401 sans token / token invalide ; `auth.ok` + écho +
pong avec un vrai token issu de `/auth/login`).

## T3 — ce qui est livré

Le premier pas « de jeu » : un personnage persistant entre dans une zone et en
reçoit l'état.

- **Première entité métier** (`internal/domain`) : `Character`, projection
  vivante d'une ligne `characters`.
- **Personnage persistant** (`internal/store/characters.go`) : à la connexion,
  le serveur charge le personnage du compte depuis PostgreSQL, ou en **crée un
  par défaut** à la première fois (faction par défaut, zone de départ). Une
  reconnexion recharge le **même** personnage — aucun doublon.
- **Données de référence** (migration `0002_seed.sql`) : deux factions, une zone
  de départ sûre, une zone de plaines PvP, un village — le minimum jouable.
- **Gestionnaire de zones en mémoire** (`internal/zone`) : registre des
  présences par zone (entrée / sortie / snapshot), protégé pour l'accès
  concurrent. C'est l'embryon de l'acteur de zone que T4 dotera d'une boucle de
  tick. Sa perte n'entraîne aucune perte durable (positions reprises de la base).
- **`zone.snapshot`** : à l'entrée, le joueur reçoit l'état cohérent de sa zone —
  la liste des entités présentes (dont lui-même), avec seulement ce qu'il a le
  droit de voir.

À la connexion WebSocket, la séquence est donc : `auth.ok` → chargement du
personnage → entrée en zone → `zone.snapshot`.

```
J1 se connecte ─▶ zone.snapshot { entities: [J1] }
J2 se connecte ─▶ zone.snapshot { entities: [J1, J2] }   (même zone)
```

**Critère de validation T3** : *« Un joueur connecté charge son personnage
persistant, entre dans une zone, et reçoit un zone.snapshot des entités
présentes. »* ✅ — vérifié de bout en bout : deux comptes rejoignent la zone de
départ (le 2ᵉ voit bien les deux personnages), état persisté en base
(`last_played_at` inclus), et reconnexion sur le même personnage.

## T4 — ce qui est livré

Le monde se met à vivre : mouvement autoritatif en temps réel.

- **La zone devient un acteur** (`internal/zone`) : une unique goroutine possède
  tout l'état de la zone (présences, positions) et le fait évoluer à chaque
  **tick** (fréquence `TICK_HZ`, 12 Hz par défaut). Plus de verrou : toutes les
  interactions (entrer, sortir, bouger) sont des commandes traitées en série par
  cette goroutine — pas de course de données (vérifié au détecteur `-race`).
- **`move.intent`** (client → serveur) : le client n'envoie qu'une **direction**
  (`{dx, dy}`, chaque axe borné à −1/0/1). Il ne transmet jamais de position.
- **Mouvement autoritatif** : à chaque tick, le serveur applique la direction à
  sa propre vitesse, borne la position aux limites de la zone, et reste seul
  maître des coordonnées. Un client ne peut pas « accélérer » en trichant sur
  l'amplitude.
- **`zone.delta`** (serveur → clients) : à chaque tick, la zone diffuse ses
  changements — **déplacements**, **arrivées** et **départs**. C'est aussi ce qui
  fait qu'un joueur voit désormais les autres entrer, bouger et quitter la zone
  en direct (ce que le `zone.snapshot` de T3 ne donnait qu'à l'entrée).

```
Client ── {"type":"move.intent","data":{"dx":1,"dy":0}} ──▶ serveur
   (à chaque tick, positions recalculées côté serveur)
serveur ── {"type":"zone.delta","data":{"tick":N,"moved":[{"character_id":"…","x":8,"y":0}]}} ──▶ tous les clients de la zone
```

> Note : les positions vivent en mémoire pendant la session. Leur persistance
> périodique (write-back) et le flush à la déconnexion arrivent en **T8** ; pour
> l'instant, une reconnexion repart du point d'apparition.

**Critère de validation T4** : *« Un joueur envoie move.intent ; le serveur
applique le déplacement de façon autoritative à chaque tick et diffuse un
zone.delta aux joueurs de la zone. »* ✅ — vérifié de bout en bout : J1 envoie une
intention, le serveur avance sa position tick par tick (x = 4, 8, 12, …), et J2
reçoit les `zone.delta` correspondants en continu.

## T5 — ce qui est livré

Le combat, **au tour par tour, déclenché par la rencontre** (choix de design par
rapport au plan initial qui prévoyait de l'auto-attaque temps réel).

- **Rencontre = engagement** : à chaque tick, la zone détecte deux personnages
  de **factions opposées** suffisamment proches et engage un combat 1v1 —
  uniquement dans une **zone PvP** (jamais dans une zone sûre : anti-griefing,
  §10). Les alliés ne se combattent pas.
- **Tour par tour** : initiative à l'**agilité** ; à son tour, le joueur choisit
  **Attaquer** (dégâts gouvernés par force/défense) ou **Fuir** (réussite selon
  l'agilité). Un tour non joué dans le délai déclenche une attaque automatique
  (pas de blocage). Le mouvement est suspendu pendant le combat.
- **Autoritatif** : le serveur calcule seul les dégâts, les PV et l'issue ; le
  client n'envoie qu'un choix d'action. Tout vit dans l'acteur de zone (pas de
  verrou ; vérifié au détecteur `-race`).
- **Mort & réapparition** : à 0 PV, le perdant réapparaît en pleine santé au
  point d'apparition, et les deux combattants gagnent une brève immunité
  (anti-re-engagement immédiat).
- **Messages** : `combat.start` (participants, à qui de jouer) → `combat.action`
  (client) → `combat.event` (dégâts, PV, tour suivant) → `combat.end` (mort +
  réapparition, ou fuite).

> Portée : les personnages apparaissent au village (zone **sûre**) et la
> **transition entre zones** n'est pas encore implémentée — atteindre une zone
> PvP en jeu normal viendra plus tard. Le combat est donc validé en plaçant des
> personnages dans la zone de plaines ; la maquette navigateur, elle, rend le
> combat directement jouable.

**Critère de validation T5** : *« Deux personnages hostiles se rencontrent en se
déplaçant, un combat au tour par tour s'engage, l'un meurt et réapparaît. »* ✅ —
vérifié de bout en bout : Lumière vs Ombre en zone de plaines, tours alternés à
20 PV de dégâts, mort après 5 coups (100 PV) et réapparition à 100 PV.

## Démarrage rapide

### Avec docker compose (recommandé)

```bash
cd mmorpg
docker compose up --build
```

Le serveur applique les migrations puis expose son API sur `:8080`.

```bash
curl localhost:8080/readyz
# {"status":"ok","checks":{"postgres":"ok","redis":"ok"}}
```

### Essayer dans le navigateur (client de test)

Une **page de test** est servie à la racine pour exercer le serveur à la main
sans écrire de code — c'est un harnais de développement, pas le client de jeu
final.

1. `cd mmorpg && docker compose up --build`
2. Ouvrir **http://localhost:8080** dans le navigateur.
3. « Créer un compte » → « Se connecter » → « Entrer en jeu ».
4. La page affiche le `zone.snapshot` reçu : la zone, ton personnage, et la
   liste des entités présentes (avec une mini-carte).

Astuce : ouvre la page dans **deux onglets** avec deux emails différents pour
voir deux personnages se rejoindre dans la même zone. Le bouton « Ping » et le
champ d'écho permettent aussi de tester le transport WebSocket (T2).

### Tester sans navigateur (smoketest)

Si le navigateur ne convient pas, une commande rejoue tout le parcours T0→T3 et
affiche un rapport ✓/✗ étape par étape. Le serveur doit déjà tourner.

```bash
# Terminal 1 — le serveur (via docker compose, ou go run ./cmd/server)
cd mmorpg && docker compose up --build

# Terminal 2 — le test (depuis mmorpg/server)
cd mmorpg/server
go run ./cmd/smoketest                    # cible http://localhost:8080
# ou, si le serveur écoute ailleurs :
go run ./cmd/smoketest http://localhost:8080
```

Sortie attendue :

```
  ✓ Le serveur est prêt (GET /readyz)
  ✓ Créer un compte (POST /auth/register)
  ✓ Se connecter (POST /auth/login)
  ✓ Vérifier la session (GET /auth/me)
  ✓ Entrer en jeu en WebSocket + recevoir auth.ok et zone.snapshot
  ✓ Ping applicatif → pong
  ✓ Écho d'un message
✅ Tout fonctionne : 7/7 étapes OK.
```

La première étape en échec indique où ça bloque : si même `/readyz` échoue, le
serveur n'est pas démarré ou pas joignable (problème Docker / port), pas un
problème applicatif.

### En local (serveur hors conteneur)

Nécessite un PostgreSQL et un Redis joignables (via `docker compose up -d postgres redis`
ou des instances locales).

```bash
cd mmorpg/server
cp ../.env.example ../.env   # ajuster si besoin
DATABASE_URL="postgres://mmorpg:mmorpg@localhost:5432/mmorpg?sslmode=disable" \
REDIS_URL="redis://localhost:6379/0" \
go run ./cmd/server
```

## Configuration (§12 : par variables d'environnement, aucun secret dans le code)

| Variable | Défaut | Rôle |
|----------|--------|------|
| `HTTP_ADDR` | `:8080` | Adresse d'écoute de l'API HTTP. |
| `DATABASE_URL` | `postgres://mmorpg:mmorpg@localhost:5432/mmorpg?sslmode=disable` | Connexion PostgreSQL. |
| `REDIS_URL` | `redis://localhost:6379/0` | Connexion Redis. |
| `TICK_HZ` | `12` | Fréquence de la boucle de zone (utilisé dès T4). |
| `STARTUP_TIMEOUT` | `30s` | Délai d'attente des dépendances au démarrage. |
| `SESSION_TTL` | `24h` | Durée de vie d'un token de session dans Redis (T1). |

## Structure

```
mmorpg/
  docker-compose.yml            # postgres + redis + server
  .env.example
  server/
    Dockerfile                  # build statique Go, image alpine minimale
    go.mod
    cmd/server/main.go          # point d'entrée : connexions, migrations, HTTP
    internal/
      config/                   # chargement de la config depuis l'env
      store/                    # accès PostgreSQL (pool pgx, migrations, comptes)
      cache/                    # accès Redis (sessions, présence, pub/sub)
      auth/                     # comptes, login, tokens de session (T1)
      httpapi/                  # API HTTP : health check + auth + route /ws
      gateway/                  # WebSocket : handshake, entrée en jeu, Conn
      protocol/                 # enveloppe {type,seq,data} + (dé)sérialisation
      zone/                     # zones-acteurs : tick, mouvement, deltas (T3/T4), combat (T5)
      domain/                   # entités métier : character (T3), item… (T6)
    migrations/                 # SQL versionné, embarqué dans le binaire
      0001_init.sql
```

## Migrations

Chaque changement de schéma est un fichier `migrations/NNNN_nom.sql`, joué une seule
fois, dans l'ordre, à l'intérieur d'une transaction, et tracé dans la table
`schema_migrations`. Les fichiers sont **embarqués dans le binaire** (`go:embed`) :
rien à copier au déploiement. Jamais d'`ALTER` manuel en base.
