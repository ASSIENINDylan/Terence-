-- Migration 0004 — Progression : attributs flottants, énergie, or, points.
--
-- Ajoute les champs nécessaires à la progression (§ techniques, académie,
-- boutique) : statistiques en flottant (l'académie fait varier de 0,5), énergie
-- pour les techniques, or, potions, et les différents points (attribut, parfait,
-- technique) plus les techniques apprises.

-- Statistiques en flottant (double precision : scan natif en float64, demi-points).
ALTER TABLE characters ALTER COLUMN str TYPE double precision;
ALTER TABLE characters ALTER COLUMN def TYPE double precision;
ALTER TABLE characters ALTER COLUMN agi TYPE double precision;

ALTER TABLE characters ADD COLUMN energy         int    NOT NULL DEFAULT 50;
ALTER TABLE characters ADD COLUMN max_energy     int    NOT NULL DEFAULT 50;
ALTER TABLE characters ADD COLUMN gold           bigint NOT NULL DEFAULT 0;
ALTER TABLE characters ADD COLUMN potions        int    NOT NULL DEFAULT 0;
ALTER TABLE characters ADD COLUMN attr_points    int    NOT NULL DEFAULT 0;
ALTER TABLE characters ADD COLUMN perfect_points int    NOT NULL DEFAULT 0;
ALTER TABLE characters ADD COLUMN tech_points    int    NOT NULL DEFAULT 0;
ALTER TABLE characters ADD COLUMN learned_techs  text[] NOT NULL DEFAULT '{}';
