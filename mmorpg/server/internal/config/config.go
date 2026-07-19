// Package config charge la configuration depuis l'environnement.
//
// Principe (§12 du plan technique) : configuration par variables
// d'environnement uniquement, aucun secret dans le code.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config regroupe tous les paramètres runtime du serveur.
type Config struct {
	// HTTPAddr est l'adresse d'écoute de l'API HTTP (login, health check).
	HTTPAddr string

	// DatabaseURL est l'URL de connexion PostgreSQL (source de vérité).
	DatabaseURL string

	// RedisURL est l'URL de connexion Redis (état volatile, sessions, pub/sub).
	RedisURL string

	// TickHz est la fréquence de la boucle de zone (§4.2). Utilisé dès T4.
	TickHz int

	// StartupTimeout borne le temps d'attente des dépendances au démarrage.
	StartupTimeout time.Duration
}

// Load lit la configuration depuis l'environnement, avec des valeurs par
// défaut adaptées au docker-compose de développement.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:       getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://mmorpg:mmorpg@localhost:5432/mmorpg?sslmode=disable"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6379/0"),
		TickHz:         getEnvInt("TICK_HZ", 12),
		StartupTimeout: getEnvDuration("STARTUP_TIMEOUT", 30*time.Second),
	}

	if cfg.TickHz <= 0 {
		return Config{}, fmt.Errorf("TICK_HZ doit être > 0, reçu %d", cfg.TickHz)
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
