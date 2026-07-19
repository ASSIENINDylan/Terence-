// Commande server : point d'entrée du serveur autoritatif MMORPG (v0.1).
//
// Étape T0 du plan de build (§13) : le serveur démarre, applique les migrations,
// se connecte à PostgreSQL et à Redis, puis expose un health check.
// Critère de validation T0 : « Le serveur démarre, se connecte à la DB et à Redis. »
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/assienindylan/terence-/mmorpg/server/internal/auth"
	"github.com/assienindylan/terence-/mmorpg/server/internal/cache"
	"github.com/assienindylan/terence-/mmorpg/server/internal/config"
	"github.com/assienindylan/terence-/mmorpg/server/internal/httpapi"
	"github.com/assienindylan/terence-/mmorpg/server/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("arrêt sur erreur", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger.Info("configuration chargée", "http_addr", cfg.HTTPAddr, "tick_hz", cfg.TickHz)

	// Contexte de démarrage borné : on attend DB et Redis, mais pas indéfiniment.
	startCtx, cancelStart := context.WithTimeout(context.Background(), cfg.StartupTimeout)
	defer cancelStart()

	// PostgreSQL : source de vérité durable (§3).
	db, err := store.Connect(startCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	logger.Info("PostgreSQL connecté")

	// Migrations de schéma versionnées (§12).
	applied, err := db.Migrate(startCtx)
	if err != nil {
		return err
	}
	if len(applied) > 0 {
		logger.Info("migrations appliquées", "versions", applied)
	} else {
		logger.Info("schéma à jour, aucune migration à appliquer")
	}

	// Redis : état volatile (sessions, présence, pub/sub) (§7).
	rc, err := cache.Connect(startCtx, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	logger.Info("Redis connecté")

	// Service d'authentification (T1) : comptes durables en PostgreSQL, tokens
	// de session volatils en Redis.
	authSvc := auth.NewService(db, rc, cfg.SessionTTL)

	// API HTTP (health check T0, authentification T1 ; WebSocket en T2).
	api := httpapi.New(db, rc, authSvc)
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Arrêt propre sur SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("serveur HTTP en écoute", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("signal d'arrêt reçu, fermeture en cours")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	logger.Info("arrêt propre terminé")
	return nil
}
