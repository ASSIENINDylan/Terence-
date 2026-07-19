// Package store isole tout l'accès à PostgreSQL, la source de vérité durable
// (§3, §6 du plan technique). Toutes les requêtes passent par ici et sont
// paramétrées (jamais de concaténation SQL — §10).
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store encapsule le pool de connexions PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// Connect ouvre un pool vers PostgreSQL et attend qu'il réponde, en réessayant
// jusqu'à ce que le timeout du contexte expire (utile quand le serveur démarre
// en même temps que la base sous docker compose).
func Connect(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: configuration du pool: %w", err)
	}

	if err := waitReady(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func waitReady(ctx context.Context, pool *pgxpool.Pool) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		if err := pool.Ping(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("store: PostgreSQL injoignable: %w (dernière erreur: %v)", ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

// Ping vérifie que la base répond (utilisé par le health check).
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Pool expose le pool sous-jacent pour les modules internes (migrations, etc.).
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// Close ferme proprement le pool.
func (s *Store) Close() {
	s.pool.Close()
}
