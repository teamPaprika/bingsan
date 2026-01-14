package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/teamPaprika/bingsan/internal/api"
	"github.com/teamPaprika/bingsan/internal/config"
	"github.com/teamPaprika/bingsan/internal/db"
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
