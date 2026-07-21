-- Migration 0006 — Village : portail de sortie repositionné à l'extrémité.
--
-- Le portail village → zone verte se trouvait au centre (100, 0). On le place à
-- l'extrémité nord du village (0, 440) pour dégager la place centrale et
-- répartir les services sur toute l'étendue du village (aménagement client).

UPDATE zone_links
SET from_x = 0, from_y = 440
WHERE kind = 'portal'
  AND from_zone_id IN (SELECT id FROM zones WHERE tier = 'village');
