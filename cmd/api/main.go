// Command api runs the cloudforge REST API server.
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

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/alirezawaskari/cloudforge/internal/config"
	"github.com/alirezawaskari/cloudforge/internal/handlers"
	appmw "github.com/alirezawaskari/cloudforge/internal/middleware"
	"github.com/alirezawaskari/cloudforge/internal/store"
	"github.com/alirezawaskari/cloudforge/internal/telemetry"
	"github.com/alirezawaskari/cloudforge/internal/version"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("fatal startup error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := telemetry.Init(ctx, cfg.OTelServiceName, cfg.OTelExporterEndpoint, cfg.OTelEnabled)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracing(shutdownCtx); err != nil {
			logger.Warn("tracer shutdown error", "error", err)
		}
	}()

	db, err := store.NewPostgres(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return err
	}
	defer db.Close()

	cache := store.NewRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer cache.Close()

	health := &handlers.HealthAPI{DB: db, Cache: cache}
	items := &handlers.ItemsAPI{DB: db, Cache: cache, Logger: logger}

	router := newRouter(logger, health, items)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           otelhttp.NewHandler(router, cfg.OTelServiceName),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting",
			"port", cfg.Port,
			"env", cfg.Env,
			"version", version.Version,
			"commit", version.Commit,
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Connecting dependencies happens in the background rather than
	// blocking process startup: /livez answers immediately so kubelet
	// never mistakes a slow-starting Postgres/Redis for a crashed
	// process, and the Kubernetes startup probe polls /startupz (backed
	// by health.ready) until this goroutine flips it true.
	depsErr := make(chan error, 1)
	go func() {
		depsErr <- connectDependencies(ctx, logger, cfg, db, cache, health)
	}()

	select {
	case err := <-serverErr:
		return err
	case err := <-depsErr:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining connections")
		return shutdown(srv, health, cfg, logger)
	}

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining connections")
	}

	return shutdown(srv, health, cfg, logger)
}

// connectDependencies retries the initial Postgres/Redis connection with
// backoff instead of failing fast, since dependencies (e.g. a StatefulSet
// doing initdb) commonly take longer to become reachable than the
// container itself takes to start. It gives up after
// cfg.DependencyReadyTimeout has elapsed with no successful connection.
func connectDependencies(ctx context.Context, logger *slog.Logger, cfg *config.Config, db *store.Postgres, cache *store.Redis, health *handlers.HealthAPI) error {
	deadline := time.Now().Add(cfg.DependencyReadyTimeout)
	backoff := 500 * time.Millisecond

	for {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		dbErr := db.Ping(pingCtx)
		cancel()

		var cacheErr error
		if dbErr == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			cacheErr = cache.Ping(pingCtx)
			cancel()
		}

		if dbErr == nil && cacheErr == nil {
			break
		}

		if time.Now().After(deadline) {
			if dbErr != nil {
				return dbErr
			}
			return cacheErr
		}

		logger.Warn("dependencies not ready yet, retrying", "database_error", dbErr, "cache_error", cacheErr, "retry_in", backoff)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		if backoff < 5*time.Second {
			backoff *= 2
		}
	}

	if err := db.Migrate(ctx); err != nil {
		return err
	}

	health.SetReady(true)
	logger.Info("dependencies connected, ready to serve")
	return nil
}

func shutdown(srv *http.Server, health *handlers.HealthAPI, cfg *config.Config, logger *slog.Logger) error {
	// Mark not-ready immediately so the readiness probe fails and the load
	// balancer stops routing new traffic while in-flight requests finish.
	health.SetReady(false)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed, forcing close", "error", err)
		return srv.Close()
	}

	logger.Info("server shut down cleanly")
	return nil
}

func newRouter(logger *slog.Logger, health *handlers.HealthAPI, items *handlers.ItemsAPI) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(appmw.RequestLogger(logger))
	r.Use(appmw.Metrics)
	r.Use(chimw.Timeout(30 * time.Second))

	r.Get("/livez", health.Livez)
	r.Get("/readyz", health.Readyz)
	r.Get("/startupz", health.Startupz)
	r.Get("/version", handlers.Version)
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1/items", items.Routes)

	return r
}
