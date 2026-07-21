-- Migration 0005 — Catalogue d'objets (T6, inventaire).
--
-- Peuple item_templates (créée en 0001) : armes (bonus d'attaque), armures
-- (bonus de défense) et consommables (soin). Trois paliers, alignés sur les
-- zones vert / orange / rouge, plus des potions de soin ramassables.
--
-- Les statistiques d'équipement/soin vivent dans la colonne jsonb `stats`
-- (ex. {"atk":6}, {"def":6}, {"heal":40}), lue en domain.ItemStats.

INSERT INTO item_templates (code, name, type, stackable, max_stack, stats) VALUES
    -- Armes (bonus d'attaque).
    ('dague_usee',   'Dague usée',      'weapon',     false, 1,  '{"atk":3}'),
    ('epee_courte',  'Épée courte',     'weapon',     false, 1,  '{"atk":6}'),
    ('lame_ardente', 'Lame ardente',    'weapon',     false, 1,  '{"atk":10}'),
    -- Armures (bonus de défense).
    ('tunique_cuir', 'Tunique de cuir', 'armor',      false, 1,  '{"def":3}'),
    ('cotte_mailles','Cotte de mailles','armor',      false, 1,  '{"def":6}'),
    ('armure_plaques','Armure de plaques','armor',    false, 1,  '{"def":10}'),
    -- Consommables (soin).
    ('potion_soin',  'Potion de soin',  'consumable', true,  10, '{"heal":40}'),
    ('elixir_majeur','Élixir majeur',   'consumable', true,  5,  '{"heal":90}')
ON CONFLICT (code) DO NOTHING;
