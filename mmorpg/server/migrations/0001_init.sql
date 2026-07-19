-- Migration 0001 — Schéma initial v0.1
-- Traduit le modèle de données du §6 du plan technique.
-- Conventions : PK bigint/int à identité (ou uuid pour les entités exposées au
-- client), timestamptz partout, deleted_at pour la suppression logique, JSONB
-- pour le semi-structuré. Aucune logique de jeu ici : la vérité durable seulement.

-- citext : emails et noms insensibles à la casse.
CREATE EXTENSION IF NOT EXISTS citext;
-- pgcrypto : génération d'UUID côté base (gen_random_uuid).
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ── Comptes joueurs ─────────────────────────────────────────────────────────
CREATE TABLE accounts (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         citext UNIQUE NOT NULL,
    password_hash text   NOT NULL,                       -- Argon2id/bcrypt, jamais en clair
    status        text   NOT NULL DEFAULT 'active'
                         CHECK (status IN ('active', 'banned', 'pending')),
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_login_at timestamptz
);

-- ── Structure du monde : factions ───────────────────────────────────────────
CREATE TABLE factions (
    id    int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name  text NOT NULL,
    color text NOT NULL DEFAULT '#888888'
);

-- ── Carte : zones ───────────────────────────────────────────────────────────
CREATE TABLE zones (
    id          int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code        text UNIQUE NOT NULL,                    -- ex. z_forest
    name        text NOT NULL,
    pvp_enabled boolean NOT NULL DEFAULT false,
    is_safe     boolean NOT NULL DEFAULT true            -- zone sûre anti-griefing (§10)
);

-- ── Villages (points d'apparition rattachés à une faction) ──────────────────
CREATE TABLE villages (
    id            int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          text NOT NULL,
    faction_id    int  NOT NULL REFERENCES factions(id),
    spawn_zone_id int  REFERENCES zones(id)              -- point d'apparition d'origine
);

-- ── Saisons (cycles du monde) ───────────────────────────────────────────────
CREATE TABLE seasons (
    id         int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    number     int  NOT NULL,
    started_at timestamptz NOT NULL DEFAULT now(),
    ends_at    timestamptz,
    status     text NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'ended'))
);

-- ── Personnages (cœur du jeu, exposés au client par uuid) ───────────────────
CREATE TABLE characters (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      bigint NOT NULL REFERENCES accounts(id),
    name            citext UNIQUE NOT NULL,               -- nom unique dans le monde
    faction_id      int    NOT NULL REFERENCES factions(id),
    home_village_id int    REFERENCES villages(id),
    level           int    NOT NULL DEFAULT 1,
    xp              bigint NOT NULL DEFAULT 0,
    hp              int    NOT NULL DEFAULT 100,
    max_hp          int    NOT NULL DEFAULT 100,
    str             int    NOT NULL DEFAULT 10,
    def             int    NOT NULL DEFAULT 10,
    agi             int    NOT NULL DEFAULT 10,
    pos_zone_id     int    REFERENCES zones(id),
    pos_x           int    NOT NULL DEFAULT 0,
    pos_y           int    NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),
    last_played_at  timestamptz,
    deleted_at      timestamptz                            -- suppression logique
);
CREATE INDEX idx_characters_account ON characters(account_id);
CREATE INDEX idx_characters_zone    ON characters(pos_zone_id);

-- ── Objets : catalogue immuable vs exemplaires possédés ─────────────────────
CREATE TABLE item_templates (
    id        int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code      text UNIQUE NOT NULL,                       -- ex. sword_short
    name      text NOT NULL,
    type      text NOT NULL
                   CHECK (type IN ('weapon', 'armor', 'consumable')),
    stackable boolean NOT NULL DEFAULT false,
    max_stack int     NOT NULL DEFAULT 1,
    stats     jsonb   NOT NULL DEFAULT '{}'::jsonb         -- ex. {"atk":5}
);

-- Chaque exemplaire a un id unique : pilier de l'anti-duplication (§8).
CREATE TABLE inventory_items (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    character_id uuid NOT NULL REFERENCES characters(id),  -- propriétaire actuel
    template_id  int  NOT NULL REFERENCES item_templates(id),
    quantity     int  NOT NULL DEFAULT 1,
    slot         int  NOT NULL DEFAULT 0,
    equipped     boolean NOT NULL DEFAULT false
);
CREATE INDEX idx_inventory_character          ON inventory_items(character_id);
CREATE INDEX idx_inventory_character_equipped ON inventory_items(character_id, equipped);

-- ── Territoires (portions contrôlables d'une zone) ──────────────────────────
CREATE TABLE territories (
    id                    int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    zone_id               int NOT NULL REFERENCES zones(id),
    controlling_faction_id int REFERENCES factions(id),    -- NULL = neutre
    control_points        int NOT NULL DEFAULT 0,          -- jauge de contrôle
    season_id             int REFERENCES seasons(id)
);
CREATE INDEX idx_territories_zone ON territories(zone_id);

-- ── Bilan de fin de saison ──────────────────────────────────────────────────
CREATE TABLE season_results (
    season_id  int NOT NULL REFERENCES seasons(id),
    faction_id int NOT NULL REFERENCES factions(id),
    score      int NOT NULL DEFAULT 0,
    rank       int,
    PRIMARY KEY (season_id, faction_id)
);

-- ── Social : clans ──────────────────────────────────────────────────────────
CREATE TABLE clans (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name                citext UNIQUE NOT NULL,
    leader_character_id uuid REFERENCES characters(id),
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE clan_members (
    clan_id      bigint NOT NULL REFERENCES clans(id),
    character_id uuid   NOT NULL REFERENCES characters(id),
    role         text   NOT NULL DEFAULT 'member',        -- extension future
    joined_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (clan_id, character_id)
);

-- ── Chat (conservé pour la modération, optionnel) ───────────────────────────
CREATE TABLE chat_messages (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    channel_type        text NOT NULL
                             CHECK (channel_type IN ('global', 'faction', 'clan')),
    channel_id          bigint,                            -- id de faction/clan selon le canal
    sender_character_id uuid REFERENCES characters(id),
    body                text NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_chat_channel ON chat_messages(channel_type, channel_id, created_at);

-- ── Reliable outbox (§6) : événements à propager de façon fiable ────────────
-- Écrits dans la même transaction que le changement d'état, puis publiés via
-- Redis pub/sub. Garantit qu'un événement n'est jamais perdu.
CREATE TABLE outbox (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    topic        text  NOT NULL,
    payload      jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz                               -- NULL = pas encore publié
);
CREATE INDEX idx_outbox_unpublished ON outbox(created_at) WHERE published_at IS NULL;
