package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrSessionNotFound : le token est inconnu ou a expiré.
var ErrSessionNotFound = errors.New("cache: session introuvable ou expirée")

// sessionKey préfixe les clés de session pour isoler leur espace de noms dans
// Redis. Un token opaque unique par session ⇒ pas de collision.
func sessionKey(token string) string {
	return "session:" + token
}

// CreateSession associe un token de session à un compte, avec une durée de vie.
// La session vit uniquement dans Redis (§7) : sa perte n'impose qu'une nouvelle
// connexion, jamais une perte de données durables.
func (c *Cache) CreateSession(ctx context.Context, token string, accountID int64, ttl time.Duration) error {
	if err := c.client.Set(ctx, sessionKey(token), accountID, ttl).Err(); err != nil {
		return fmt.Errorf("cache: création de session: %w", err)
	}
	return nil
}

// GetSession résout un token vers l'identifiant de compte associé.
// Renvoie ErrSessionNotFound si le token est absent ou expiré.
func (c *Cache) GetSession(ctx context.Context, token string) (int64, error) {
	v, err := c.client.Get(ctx, sessionKey(token)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, ErrSessionNotFound
		}
		return 0, fmt.Errorf("cache: lecture de session: %w", err)
	}
	accountID, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cache: session corrompue (%q): %w", v, err)
	}
	return accountID, nil
}

// DeleteSession révoque un token (déconnexion). Idempotent : supprimer un token
// déjà absent n'est pas une erreur.
func (c *Cache) DeleteSession(ctx context.Context, token string) error {
	if err := c.client.Del(ctx, sessionKey(token)).Err(); err != nil {
		return fmt.Errorf("cache: suppression de session: %w", err)
	}
	return nil
}
