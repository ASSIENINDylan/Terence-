-- Migration 0003 — Monde à paliers, trois villages élémentaires et transitions.
--
-- Refonte du monde v0.1 : remplace le seed provisoire de 0002 (Lumière/Ombre,
-- village + plaines) par la vraie carte.
--   • 3 villages élémentaires (Feu/Eau/Terre), points d'apparition, zones sûres.
--   • Paliers de danger : village → vert (PvE seul) → orange (PvP, −½ XP à la
--     mort) → rouge (PvP, −toute l'XP). PvE partout ; PvP en orange/rouge.
--   • Transitions par portails (village↔vert↔orange) et barrières (orange↔rouge).
--   • Zones orange et rouge PARTAGÉES entre les trois éléments : c'est là qu'ils
--     se rencontrent.
--
-- Prototype pré-lancement : cette migration réinitialise les données de monde et
-- de personnages (aucune donnée de production à préserver).

-- ── Schéma ──────────────────────────────────────────────────────────────────

-- Palier de danger d'une zone.
ALTER TABLE zones ADD COLUMN tier text NOT NULL DEFAULT 'green'
    CHECK (tier IN ('village', 'green', 'orange', 'red'));

-- Élément d'une faction et statistique bonus du village associé.
ALTER TABLE factions ADD COLUMN element    text;
ALTER TABLE factions ADD COLUMN bonus_stat text
    CHECK (bonus_stat IN ('str', 'def', 'agi'));

-- Liens de transition entre zones (portails et barrières). Un lien est
-- unidirectionnel ; on en seede deux par passage pour l'aller-retour.
CREATE TABLE zone_links (
    id           int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    from_zone_id int  NOT NULL REFERENCES zones(id),
    to_zone_id   int  NOT NULL REFERENCES zones(id),
    kind         text NOT NULL CHECK (kind IN ('portal', 'barrier')),
    from_x       int  NOT NULL,     -- position du portail dans la zone de départ
    from_y       int  NOT NULL,
    to_x         int  NOT NULL,     -- point d'arrivée dans la zone de destination
    to_y         int  NOT NULL,
    min_level    int  NOT NULL DEFAULT 0
);
CREATE INDEX idx_zone_links_from ON zone_links(from_zone_id);

-- ── Réinitialisation des données de monde/personnages ───────────────────────
TRUNCATE zones, factions, villages, characters,
         inventory_items, clans, clan_members, chat_messages,
         territories, season_results
    RESTART IDENTITY CASCADE;

-- ── Factions élémentaires ───────────────────────────────────────────────────
INSERT INTO factions (name, color, element, bonus_stat) VALUES
    ('Feu',   '#ef4444', 'feu',   'str'),   -- bonus attaque
    ('Eau',   '#3b82f6', 'eau',   'agi'),   -- bonus agilité
    ('Terre', '#16a34a', 'terre', 'def');   -- bonus défense

-- ── Zones (paliers) ─────────────────────────────────────────────────────────
-- village : sûr (ni PvP). vert : PvE seul. orange/rouge : PvP.
INSERT INTO zones (code, name, tier, pvp_enabled, is_safe) VALUES
    ('village_feu',   'Village du Feu',    'village', false, true),
    ('village_eau',   'Village de l''Eau', 'village', false, true),
    ('village_terre', 'Village de la Terre','village', false, true),
    ('green_feu',     'Terres de Feu',     'green',  false, false),
    ('green_eau',     'Rivages de l''Eau', 'green',  false, false),
    ('green_terre',   'Plaines de Terre',  'green',  false, false),
    ('orange_marches','Marches contestées','orange', true,  false),
    ('red_abysses',   'Abysses',           'red',    true,  false);

-- ── Villages (points d'apparition, un par élément) ──────────────────────────
INSERT INTO villages (name, faction_id, spawn_zone_id)
SELECT v.vname, f.id, z.id
FROM (VALUES
    ('Foyer du Feu',    'feu',   'village_feu'),
    ('Source de l''Eau','eau',   'village_eau'),
    ('Racine de Terre', 'terre', 'village_terre')
) AS v(vname, elem, zcode)
JOIN factions f ON f.element = v.elem
JOIN zones z ON z.code = v.zcode;

-- ── Liens de transition ─────────────────────────────────────────────────────
INSERT INTO zone_links (from_zone_id, to_zone_id, kind, from_x, from_y, to_x, to_y)
SELECT zf.id, zt.id, l.kind, l.fx, l.fy, l.tx, l.ty
FROM (VALUES
    -- village ↔ vert (portails)
    ('village_feu',   'green_feu',      'portal', 100,   0,  0,   0),
    ('green_feu',     'village_feu',    'portal',-100,   0,  0,   0),
    ('village_eau',   'green_eau',      'portal', 100,   0,  0,   0),
    ('green_eau',     'village_eau',    'portal',-100,   0,  0,   0),
    ('village_terre', 'green_terre',    'portal', 100,   0,  0,   0),
    ('green_terre',   'village_terre',  'portal',-100,   0,  0,   0),
    -- vert ↔ orange partagé (portails) ; arrivées espacées par élément
    ('green_feu',     'orange_marches', 'portal', 100,   0,-120,-120),
    ('orange_marches','green_feu',      'portal',-120,-120, 0,   0),
    ('green_eau',     'orange_marches', 'portal', 100,   0,-120, 120),
    ('orange_marches','green_eau',      'portal',-120, 120, 0,   0),
    ('green_terre',   'orange_marches', 'portal', 100,   0, 120,-120),
    ('orange_marches','green_terre',    'portal', 120,-120, 0,   0),
    -- orange ↔ rouge (barrières)
    ('orange_marches','red_abysses',    'barrier',120, 120, 0,-120),
    ('red_abysses',   'orange_marches', 'barrier',  0,-120,120, 120)
) AS l(from_code, to_code, kind, fx, fy, tx, ty)
JOIN zones zf ON zf.code = l.from_code
JOIN zones zt ON zt.code = l.to_code;
