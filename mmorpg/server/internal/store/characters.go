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

// Éléments de départ possibles (un village par élément).
var elements = []string{"feu", "eau", "terre"}

// ValidElement indique si e est un élément de départ connu.
func ValidElement(e string) bool {
	for _, x := range elements {
		if x == e {
			return true
		}
	}
	return false
}

// GetOrCreateForAccount retourne le personnage du compte, en en créant un par
// défaut à la première connexion dans le village de l'élément demandé. Si
// element est vide ou invalide, un élément est attribué de façon déterministe.
func (s *Store) GetOrCreateForAccount(ctx context.Context, accountID int64, element string) (domain.Character, error) {
	c, err := s.firstCharacter(ctx, accountID)
	if err == nil {
		return c, nil
	}
	if !errors.Is(err, ErrNoCharacter) {
		return domain.Character{}, err
	}
	if !ValidElement(element) {
		element = elements[(accountID-1)%int64(len(elements))]
	}
	return s.createCharacter(ctx, accountID, element)
}

const characterCols = `c.id, c.account_id, c.name, c.faction_id,
	f.element, f.bonus_stat, c.level, c.xp,
	c.hp, c.max_hp, c.str, c.def, c.agi,
	c.energy, c.max_energy, c.gold, c.potions,
	c.attr_points, c.perfect_points, c.tech_points, c.learned_techs,
	COALESCE(c.pos_zone_id, 0), c.pos_x, c.pos_y, COALESCE(v.spawn_zone_id, 0)`

func scanCharacter(row pgx.Row) (domain.Character, error) {
	var c domain.Character
	err := row.Scan(&c.ID, &c.AccountID, &c.Name, &c.FactionID,
		&c.Element, &c.SpecStat, &c.Level, &c.XP,
		&c.HP, &c.MaxHP, &c.Str, &c.Def, &c.Agi,
		&c.Energy, &c.MaxEnergy, &c.Gold, &c.Potions,
		&c.AttrPoints, &c.PerfectPoints, &c.TechPoints, &c.LearnedTechs,
		&c.ZoneID, &c.X, &c.Y, &c.HomeZoneID)
	return c, err
}

func (s *Store) firstCharacter(ctx context.Context, accountID int64) (domain.Character, error) {
	c, err := scanCharacter(s.pool.QueryRow(ctx, `
		SELECT `+characterCols+`
		FROM characters c
		JOIN factions f ON f.id = c.faction_id
		LEFT JOIN villages v ON v.id = c.home_village_id
		WHERE c.account_id = $1 AND c.deleted_at IS NULL
		ORDER BY c.created_at
		LIMIT 1`, accountID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Character{}, ErrNoCharacter
		}
		return domain.Character{}, fmt.Errorf("store: lecture du personnage: %w", err)
	}
	return c, nil
}

// createCharacter crée un personnage de départ dans le village de l'élément
// choisi, avec le bonus de statistique de ce village (feu→attaque, eau→agilité,
// terre→défense).
func (s *Store) createCharacter(ctx context.Context, accountID int64, element string) (domain.Character, error) {
	name := fmt.Sprintf("%s-%d", elementTitle(element), accountID)

	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO characters (account_id, name, faction_id, home_village_id, pos_zone_id, pos_x, pos_y, str, def, agi)
		SELECT $1, $2, f.id, v.id, z.id, 0, 0,
		       10 + CASE WHEN f.bonus_stat = 'str' THEN 5 ELSE 0 END,
		       10 + CASE WHEN f.bonus_stat = 'def' THEN 5 ELSE 0 END,
		       10 + CASE WHEN f.bonus_stat = 'agi' THEN 5 ELSE 0 END
		FROM factions f
		JOIN zones z    ON z.code = 'village_' || f.element
		JOIN villages v ON v.spawn_zone_id = z.id AND v.faction_id = f.id
		WHERE f.element = $3
		RETURNING id`,
		accountID, name, element,
	).Scan(&id)
	if err != nil {
		return domain.Character{}, fmt.Errorf("store: création du personnage (%s): %w", element, err)
	}
	return s.firstCharacter(ctx, accountID)
}

func elementTitle(e string) string {
	switch e {
	case "feu":
		return "Feu"
	case "eau":
		return "Eau"
	case "terre":
		return "Terre"
	}
	return "Aventurier"
}

// SaveState persiste la position, la zone, les PV et l'XP d'un personnage. Appelé
// aux transitions de zone et aux morts (persistance périodique complète en T8).
func (s *Store) SaveState(ctx context.Context, c domain.Character) error {
	if _, err := s.pool.Exec(ctx, `
		UPDATE characters
		SET pos_zone_id = $2, pos_x = $3, pos_y = $4, hp = $5, xp = $6, level = $7,
		    str = $8, def = $9, agi = $10, energy = $11, max_energy = $12, max_hp = $13,
		    gold = $14, potions = $15, attr_points = $16, perfect_points = $17,
		    tech_points = $18, learned_techs = $19, last_played_at = now()
		WHERE id = $1`,
		c.ID, c.ZoneID, c.X, c.Y, c.HP, c.XP, c.Level,
		c.Str, c.Def, c.Agi, c.Energy, c.MaxEnergy, c.MaxHP,
		c.Gold, c.Potions, c.AttrPoints, c.PerfectPoints,
		c.TechPoints, c.LearnedTechs,
	); err != nil {
		return fmt.Errorf("store: sauvegarde de l'état du personnage: %w", err)
	}
	return nil
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
