// Package cache isole l'accès à Redis : état volatile et rapide (§7 du plan) —
// tokens de session, présence en ligne, pub/sub inter-zones, rate-limiting.
//
// Principe de sûreté : rien de durable ne vit uniquement dans Redis ; sa perte
// force au pire des reconnexions, jamais la destruction du monde.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache encapsule le client Redis.
type Cache struct {
	client *redis.Client
}

// Connect ouvre un client Redis et attend qu'il réponde, en réessayant jusqu'à
// l'expiration du contexte (démarrage simultané sous docker compose).
func Connect(ctx context.Context, redisURL string) (*Cache, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("cache: URL Redis invalide: %w", err)
	}
	client := redis.NewClient(opt)

	if err := waitReady(ctx, client); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &Cache{client: client}, nil
}

func waitReady(ctx context.Context, client *redis.Client) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		if err := client.Ping(ctx).Err(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("cache: Redis injoignable: %w (dernière erreur: %v)", ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

// Ping vérifie que Redis répond (utilisé par le health check).
func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Client expose le client sous-jacent pour les modules internes (sessions, pub/sub).
func (c *Cache) Client() *redis.Client {
	return c.client
}

// Close ferme proprement la connexion.
func (c *Cache) Close() error {
	return c.client.Close()
}
