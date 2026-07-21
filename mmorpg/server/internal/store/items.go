package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/assienindylan/terence-/mmorpg/server/internal/domain"
)

// ListItemTemplates charge le catalogue immuable d'objets (§8), une fois au
// démarrage. La colonne jsonb `stats` est décodée en domain.ItemStats.
func (s *Store) ListItemTemplates(ctx context.Context) ([]domain.ItemTemplate, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, code, name, type, stackable, max_stack, stats
		FROM item_templates
		ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("store: lecture du catalogue d'objets: %w", err)
	}
	defer rows.Close()

	var out []domain.ItemTemplate
	for rows.Next() {
		var t domain.ItemTemplate
		var typ string
		var raw []byte
		if err := rows.Scan(&t.ID, &t.Code, &t.Name, &typ, &t.Stackable, &t.MaxStack, &raw); err != nil {
			return nil, fmt.Errorf("store: scan d'un template d'objet: %w", err)
		}
		t.Type = domain.ItemType(typ)
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &t.Stats); err != nil {
				return nil, fmt.Errorf("store: stats d'objet %q illisibles: %w", t.Code, err)
			}
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// LoadInventory charge les exemplaires possédés par un personnage.
func (s *Store) LoadInventory(ctx context.Context, characterID string) ([]domain.InventoryItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, template_id, quantity, equipped
		FROM inventory_items
		WHERE character_id = $1
		ORDER BY slot, id`, characterID)
	if err != nil {
		return nil, fmt.Errorf("store: lecture de l'inventaire: %w", err)
	}
	defer rows.Close()

	var out []domain.InventoryItem
	for rows.Next() {
		var it domain.InventoryItem
		if err := rows.Scan(&it.ID, &it.TemplateID, &it.Quantity, &it.Equipped); err != nil {
			return nil, fmt.Errorf("store: scan d'un exemplaire: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// SaveInventory persiste l'inventaire AUTORITATIF (tenu en mémoire par l'acteur
// de zone) : on remplace en bloc les exemplaires du personnage dans une
// transaction. Chaque exemplaire garde son identifiant unique (anti-duplication).
func (s *Store) SaveInventory(ctx context.Context, characterID string, items []domain.InventoryItem) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: ouverture de transaction inventaire: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM inventory_items WHERE character_id = $1`, characterID); err != nil {
		return fmt.Errorf("store: purge de l'inventaire: %w", err)
	}
	for i, it := range items {
		qty := it.Quantity
		if qty < 1 {
			qty = 1
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO inventory_items (id, character_id, template_id, quantity, slot, equipped)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			it.ID, characterID, it.TemplateID, qty, i, it.Equipped); err != nil {
			return fmt.Errorf("store: insertion d'un exemplaire: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: commit de l'inventaire: %w", err)
	}
	return nil
}
