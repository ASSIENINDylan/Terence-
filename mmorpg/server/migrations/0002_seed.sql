-- Migration 0002 — Données de référence minimales (bootstrap v0.1)
-- Sans faction ni zone de départ, aucun personnage ne peut être créé
-- (characters.faction_id est NOT NULL et pos_zone_id référence zones).
-- Ce seed est le strict minimum jouable ; le vrai contenu viendra plus tard.

-- Rendre les noms de faction uniques : permet un seed idempotent et empêche les
-- doublons de factions.
ALTER TABLE factions ADD CONSTRAINT factions_name_key UNIQUE (name);

-- Deux factions opposées (le cœur du jeu de territoire, §2).
INSERT INTO factions (name, color) VALUES
    ('Lumière', '#3b82f6'),
    ('Ombre',   '#ef4444')
ON CONFLICT (name) DO NOTHING;

-- Zone de départ (sûre, sans PvP) + une zone de plaines (PvP activé).
INSERT INTO zones (code, name, pvp_enabled, is_safe) VALUES
    ('z_village_start', 'Village de départ', false, true),
    ('z_plains',        'Plaines',           true,  false)
ON CONFLICT (code) DO NOTHING;

-- Un village rattaché à la faction Lumière, apparaissant dans la zone de départ.
INSERT INTO villages (name, faction_id, spawn_zone_id)
SELECT 'Bastion de Lumière', f.id, z.id
FROM factions f, zones z
WHERE f.name = 'Lumière' AND z.code = 'z_village_start'
  AND NOT EXISTS (SELECT 1 FROM villages WHERE name = 'Bastion de Lumière');
