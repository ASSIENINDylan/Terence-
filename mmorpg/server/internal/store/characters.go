package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
)

// ErrNoCharacter : le compte ne possède encore aucun personnage.
var ErrNoCharacter = errors.New("store: aucun personnage pour ce compte")

// startingZoneCode est la zone où apparaissent les nouveaux personnages (seed
// 0002). Le spawn est volontairement au centre logique (0,0) en v0.1.
const startingZoneCode = "z_village_start"

// GetOrCreateForAccount retourne le personnage du compte, en en créant un par
// défaut à la première connexion. En v0.1 un compte a un seul personnage ; la
// sélection multi-personnages viendra plus tard.
func (s *Store) GetOrCreateForAccount(ctx context.Context, accountID int64) (domain.Character, error) {
	c, err := s.firstCharacter(ctx, accountID)
	if err == nil {
		return c, nil
	}
	if !errors.Is(err, ErrNoCharacter) {
		return domain.Character{}, err
	}
	return s.createDefaultCharacter(ctx, accountID)
}

// firstCharacter charge le plus ancien personnage non supprimé du compte.
func (s *Store) firstCharacter(ctx context.Context, accountID int64) (domain.Character, error) {
	var c domain.Character
	err := s.pool.QueryRow(ctx, `
		SELECT id, account_id, name, faction_id, level, hp, max_hp, str, def, agi,
		       COALESCE(pos_zone_id, 0), pos_x, pos_y
		FROM characters
		WHERE account_id = $1 AND deleted_at IS NULL
		ORDER BY created_at
		LIMIT 1`,
		accountID,
	).Scan(&c.ID, &c.AccountID, &c.Name, &c.FactionID, &c.Level, &c.HP, &c.MaxHP,
		&c.Str, &c.Def, &c.Agi, &c.ZoneID, &c.X, &c.Y)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Character{}, ErrNoCharacter
		}
		return domain.Character{}, fmt.Errorf("store: lecture du personnage: %w", err)
	}
	return c, nil
}

// createDefaultCharacter crée un personnage de départ : faction par défaut (la
// plus ancienne), placé dans la zone de départ. Le nom est unique par compte.
func (s *Store) createDefaultCharacter(ctx context.Context, accountID int64) (domain.Character, error) {
	name := fmt.Sprintf("Aventurier-%d", accountID)

	var c domain.Character
	err := s.pool.QueryRow(ctx, `
		INSERT INTO characters (account_id, name, faction_id, pos_zone_id, pos_x, pos_y)
		VALUES (
			$1,
			$2,
			(SELECT id FROM factions ORDER BY id LIMIT 1),
			(SELECT id FROM zones WHERE code = $3),
			0, 0
		)
		RETURNING id, account_id, name, faction_id, level, hp, max_hp, str, def, agi,
		          COALESCE(pos_zone_id, 0), pos_x, pos_y`,
		accountID, name, startingZoneCode,
	).Scan(&c.ID, &c.AccountID, &c.Name, &c.FactionID, &c.Level, &c.HP, &c.MaxHP,
		&c.Str, &c.Def, &c.Agi, &c.ZoneID, &c.X, &c.Y)
	if err != nil {
		return domain.Character{}, fmt.Errorf("store: création du personnage: %w", err)
	}
	return c, nil
}

// TouchLastPlayed met à jour l'horodatage de dernière session du personnage.
func (s *Store) TouchLastPlayed(ctx context.Context, characterID string) error {
	if _, err := s.pool.Exec(ctx,
		`UPDATE characters SET last_played_at = now() WHERE id = $1`, characterID,
	); err != nil {
		return fmt.Errorf("store: mise à jour last_played_at: %w", err)
	}
	return nil
}
