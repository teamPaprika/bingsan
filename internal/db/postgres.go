package db

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kimuyb/bingsan/internal/config"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the PostgreSQL connection pool.
type DB struct {
	Pool   *pgxpool.Pool
	config config.DatabaseConfig
}

// New creates a new database connection pool.
func New(cfg config.DatabaseConfig) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parsing database config: %w", err)
	}

	// Configure pool
	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	slog.Info("connected to database",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
	)

	return &DB{
		Pool:   pool,
		config: cfg,
	}, nil
}

// Close closes the database connection pool.
func (db *DB) Close() {
	db.Pool.Close()
	slog.Info("database connection closed")
}

// Migrate runs database migrations.
func (db *DB) Migrate() error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("creating migration source: %w", err)
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		db.config.User,
		db.config.Password,
		db.config.Host,
		db.config.Port,
		db.config.Database,
		db.config.SSLMode,
	)

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, dsn)
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("running migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("getting migration version: %w", err)
	}

	slog.Info("database migrations complete",
		"version", version,
		"dirty", dirty,
	)

	return nil
}

// Conn returns a connection from the pool.
func (db *DB) Conn(ctx context.Context) (*pgxpool.Conn, error) {
	return db.Pool.Acquire(ctx)
}
