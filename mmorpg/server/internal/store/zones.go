package store

import (
	"context"
	"fmt"
)

// ZoneInfo est la projection des attributs statiques d'une zone (§6). Ils
// gouvernent notamment le PvP : un combat ne peut s'engager que dans une zone où
// il est activé (jamais dans une zone sûre — anti-griefing, §10).
type ZoneInfo struct {
	ID         int
	Code       string
	Name       string
	PvPEnabled bool
	IsSafe     bool
}

// ListZones charge toutes les zones. Données de référence peu nombreuses et
// stables : chargées une fois au démarrage pour alimenter le gestionnaire de
// zones.
func (s *Store) ListZones(ctx context.Context) ([]ZoneInfo, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, code, name, pvp_enabled, is_safe FROM zones ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("store: lecture des zones: %w", err)
	}
	defer rows.Close()

	var zones []ZoneInfo
	for rows.Next() {
		var z ZoneInfo
		if err := rows.Scan(&z.ID, &z.Code, &z.Name, &z.PvPEnabled, &z.IsSafe); err != nil {
			return nil, fmt.Errorf("store: décodage d'une zone: %w", err)
		}
		zones = append(zones, z)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: parcours des zones: %w", err)
	}
	return zones, nil
}
