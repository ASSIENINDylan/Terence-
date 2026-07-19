package store

import (
	"context"
	"fmt"
)

// ZoneInfo est la projection des attributs statiques d'une zone (§6). Le palier
// (tier) gouverne le PvP et la pénalité d'XP à la mort.
type ZoneInfo struct {
	ID         int
	Code       string
	Name       string
	Tier       string // village | green | orange | red
	PvPEnabled bool
	IsSafe     bool
}

// ZoneLink est un point de transition (portail ou barrière) d'une zone vers une
// autre.
type ZoneLink struct {
	ID         int
	FromZoneID int
	ToZoneID   int
	Kind       string // portal | barrier
	FromX      int
	FromY      int
	ToX        int
	ToY        int
	MinLevel   int
}

// ListZones charge toutes les zones (données de référence, au démarrage).
func (s *Store) ListZones(ctx context.Context) ([]ZoneInfo, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, code, name, tier, pvp_enabled, is_safe FROM zones ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("store: lecture des zones: %w", err)
	}
	defer rows.Close()

	var zones []ZoneInfo
	for rows.Next() {
		var z ZoneInfo
		if err := rows.Scan(&z.ID, &z.Code, &z.Name, &z.Tier, &z.PvPEnabled, &z.IsSafe); err != nil {
			return nil, fmt.Errorf("store: décodage d'une zone: %w", err)
		}
		zones = append(zones, z)
	}
	return zones, rows.Err()
}

// ListZoneLinks charge tous les liens de transition (au démarrage).
func (s *Store) ListZoneLinks(ctx context.Context) ([]ZoneLink, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, from_zone_id, to_zone_id, kind, from_x, from_y, to_x, to_y, min_level
		 FROM zone_links ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("store: lecture des liens de zone: %w", err)
	}
	defer rows.Close()

	var links []ZoneLink
	for rows.Next() {
		var l ZoneLink
		if err := rows.Scan(&l.ID, &l.FromZoneID, &l.ToZoneID, &l.Kind,
			&l.FromX, &l.FromY, &l.ToX, &l.ToY, &l.MinLevel); err != nil {
			return nil, fmt.Errorf("store: décodage d'un lien: %w", err)
		}
		links = append(links, l)
	}
	return links, rows.Err()
}
