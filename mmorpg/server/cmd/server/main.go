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
	"github.com/assienindylan/terence-/mmorpg/server/internal/gateway"
	"github.com/assienindylan/terence-/mmorpg/server/internal/httpapi"
	"github.com/assienindylan/terence-/mmorpg/server/internal/store"
	"github.com/assienindylan/terence-/mmorpg/server/internal/zone"
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

	// Monde à paliers : métadonnées de zones + liens de transition, chargés une
	// fois au démarrage (données de référence stables).
	zones, err := db.ListZones(startCtx)
	if err != nil {
		return err
	}
	links, err := db.ListZoneLinks(startCtx)
	if err != nil {
		return err
	}
	metas, zoneLinks := buildWorld(zones, links)

	// Monde en mémoire : zones-acteurs avec boucle de tick à TICK_HZ.
	world := zone.NewManager(cfg.TickHz, metas, zoneLinks)
	defer world.Close()

	// Gateway temps réel (T2/T3) : handshake authentifié, puis chargement du
	// personnage persistant, entrée en zone et envoi du zone.snapshot.
	gw := gateway.New(authSvc, db, world, logger)

	// API HTTP (health check T0, authentification T1, WebSocket T2/T3).
	api := httpapi.New(db, rc, authSvc, gw)
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

// buildWorld convertit les données de référence (zones + liens) en métadonnées
// de zones (palier → PvP, pénalité d'XP, portails) et en carte des liens de
// transition pour le gestionnaire de zones.
func buildWorld(zones []store.ZoneInfo, links []store.ZoneLink) (map[int]zone.ZoneMeta, map[int]zone.Link) {
	names := make(map[int]string, len(zones))
	for _, z := range zones {
		names[z.ID] = z.Name
	}

	metas := make(map[int]zone.ZoneMeta, len(zones))
	for _, z := range zones {
		metas[z.ID] = zone.ZoneMeta{
			ID:          z.ID,
			Code:        z.Code,
			Name:        z.Name,
			Tier:        z.Tier,
			PvP:         z.PvPEnabled,
			DeathXPLoss: deathXPLoss(z.Tier),
		}
	}

	zoneLinks := make(map[int]zone.Link, len(links))
	for _, l := range links {
		zoneLinks[l.ID] = zone.Link{
			ID: l.ID, FromZoneID: l.FromZoneID, ToZoneID: l.ToZoneID, Kind: l.Kind,
			FromX: l.FromX, FromY: l.FromY, ToX: l.ToX, ToY: l.ToY, MinLevel: l.MinLevel,
		}
		// Le lien apparaît comme un portail dans sa zone de départ.
		m := metas[l.FromZoneID]
		m.Portals = append(m.Portals, zone.Portal{
			LinkID: l.ID, X: l.FromX, Y: l.FromY, Kind: l.Kind, ToZone: names[l.ToZoneID],
		})
		metas[l.FromZoneID] = m
	}
	return metas, zoneLinks
}

// deathXPLoss retourne la fraction d'XP perdue à la mort selon le palier :
// orange = la moitié, rouge = la totalité, sûr/vert = rien.
func deathXPLoss(tier string) float64 {
	switch tier {
	case "orange":
		return 0.5
	case "red":
		return 1.0
	default:
		return 0
	}
}
