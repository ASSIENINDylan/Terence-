package store

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/assienindylan/terence-/mmorpg/server/migrations"
)

// Migrate applique, dans l'ordre, les migrations non encore appliquées.
//
// Approche volontairement minimale pour la v0.1 (§12 recommande golang-migrate/
// goose/Atlas ; on garde ici un runner interne transparent, sans binaire tiers) :
// chaque fichier migrations/NNNN_nom.sql est joué une seule fois, dans une
// transaction, et enregistré dans la table schema_migrations.
func (s *Store) Migrate(ctx context.Context) (applied []string, err error) {
	if _, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return nil, fmt.Errorf("migrate: table de suivi: %w", err)
	}

	files, err := listMigrations()
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		version := strings.TrimSuffix(f, ".sql")

		var exists bool
		if err := s.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version,
		).Scan(&exists); err != nil {
			return applied, fmt.Errorf("migrate: vérification de %s: %w", version, err)
		}
		if exists {
			continue
		}

		sqlBytes, err := migrations.FS.ReadFile(f)
		if err != nil {
			return applied, fmt.Errorf("migrate: lecture de %s: %w", f, err)
		}

		if err := s.applyOne(ctx, version, string(sqlBytes)); err != nil {
			return applied, err
		}
		applied = append(applied, version)
	}
	return applied, nil
}

func (s *Store) applyOne(ctx context.Context, version, sqlText string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("migrate: début de transaction pour %s: %w", version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, sqlText); err != nil {
		return fmt.Errorf("migrate: exécution de %s: %w", version, err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`, version,
	); err != nil {
		return fmt.Errorf("migrate: enregistrement de %s: %w", version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("migrate: commit de %s: %w", version, err)
	}
	return nil
}

func listMigrations() ([]string, error) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("migrate: lecture du dossier migrations: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files) // NNNN_ garantit l'ordre lexicographique = ordre d'application
	return files, nil
}
