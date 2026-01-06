package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kimuyb/bingsan/internal/api"
	"github.com/kimuyb/bingsan/internal/background"
	"github.com/kimuyb/bingsan/internal/config"
	"github.com/kimuyb/bingsan/internal/db"
	"github.com/kimuyb/bingsan/internal/leader"
	"github.com/kimuyb/bingsan/internal/telemetry"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Set log level from config
	if cfg.Server.Debug {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	}

	slog.Info("starting iceberg rest catalog",
		"version", cfg.Server.Version,
		"port", cfg.Server.Port,
	)

	// Initialize OpenTelemetry (if enabled)
	ctx := context.Background()
	telemetryProvider, err := telemetry.NewProvider(ctx, telemetry.Config{
		Enabled:    cfg.Telemetry.Enabled,
		Endpoint:   cfg.Telemetry.Endpoint,
		SampleRate: cfg.Telemetry.SampleRate,
		Insecure:   cfg.Telemetry.Insecure,
	}, "bingsan", cfg.Server.Version)
	if err != nil {
		return fmt.Errorf("initializing telemetry: %w", err)
	}
	if telemetryProvider != nil {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if shutdownErr := telemetryProvider.Shutdown(shutdownCtx); shutdownErr != nil {
				slog.Error("error shutting down telemetry", "error", shutdownErr)
			}
		}()
	}

	// Initialize database connection
	database, err := db.New(cfg.Database)
	if err != nil {
		return err
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(); err != nil {
		return err
	}

	// Initialize leader election for background tasks
	nodeID := generateNodeID()
	elector := leader.NewElector(database.Pool, nodeID)
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = elector.ReleaseAll(releaseCtx)
	}()

	// Initialize and start background task runner
	taskRunner := background.NewRunner(elector, database)
	taskRunner.Start(ctx)
	defer taskRunner.Stop()

	// Initialize and start HTTP server
	server := api.NewServer(cfg, database)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := server.Start(); err != nil {
			slog.Error("server error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server...")

	if err := server.Shutdown(); err != nil {
		slog.Error("error during shutdown", "error", err)
	}

	slog.Info("server stopped")
	return nil
}


// generateNodeID creates a unique identifier for this node.
func generateNodeID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s-%d", hostname, os.Getpid())
}
