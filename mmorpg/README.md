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
| T1 | Comptes + login HTTP + token de session Redis. | à venir |
| T2 | WebSocket + Gateway : handshake authentifié, écho. | à venir |
| T3 | Personnage persistant + entrée en zone + `zone.snapshot`. | à venir |
| T4 | Boucle de tick + `move.intent` → position serveur → `zone.delta`. | à venir |
| T5 | Combat autoritatif (auto-attaque + PV + mort + respawn). | à venir |
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
      store/                    # accès PostgreSQL (pool pgx) + runner de migration
      cache/                    # accès Redis (sessions, présence, pub/sub)
      httpapi/                  # API HTTP : health check (login T1, WS T2)
      gateway/                  # WebSocket, auth, routage        (T2)
      zone/                     # acteur de zone, tick, combat     (T3+)
      domain/                   # entités : character, item…       (T3+)
      protocol/                 # enveloppes + (dé)sérialisation   (T2)
    migrations/                 # SQL versionné, embarqué dans le binaire
      0001_init.sql
```

## Migrations

Chaque changement de schéma est un fichier `migrations/NNNN_nom.sql`, joué une seule
fois, dans l'ordre, à l'intérieur d'une transaction, et tracé dans la table
`schema_migrations`. Les fichiers sont **embarqués dans le binaire** (`go:embed`) :
rien à copier au déploiement. Jamais d'`ALTER` manuel en base.
