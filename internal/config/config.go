package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the catalog.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Catalog  CatalogConfig  `mapstructure:"catalog"`
	Pool     PoolConfig     `mapstructure:"pool"`
	Compat   CompatConfig   `mapstructure:"compat"`
}

// CompatConfig holds compatibility layer configuration.
type CompatConfig struct {
	// PolarisEnabled enables Apache Polaris API compatibility.
	// When enabled, adds path rewriting for /api/catalog/v1/{catalog}/...
	// and mock Management API endpoints at /api/management/v1/catalogs.
	// This is useful for running polaris-tools benchmarks.
	PolarisEnabled bool `mapstructure:"polaris_enabled"`
}

// PoolConfig holds object pool configuration for performance optimization.
type PoolConfig struct {
	BufferInitialSize int `mapstructure:"buffer_initial_size"` // Initial buffer size in bytes
	BufferMaxSize     int `mapstructure:"buffer_max_size"`     // Max buffer size before discard
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Debug        bool          `mapstructure:"debug"`
	Version      string        `mapstructure:"version"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// DatabaseConfig holds PostgreSQL configuration.
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// DSN returns the PostgreSQL connection string.
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// StorageConfig holds object storage configuration.
type StorageConfig struct {
	Type      string      `mapstructure:"type"` // s3, gcs, local
	S3        S3Config    `mapstructure:"s3"`
	GCS       GCSConfig   `mapstructure:"gcs"`
	Local     LocalConfig `mapstructure:"local"`
	Warehouse string      `mapstructure:"warehouse"` // Base path for tables
}

// S3Config holds S3/MinIO configuration.
type S3Config struct {
	Endpoint        string `mapstructure:"endpoint"`
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Bucket          string `mapstructure:"bucket"`
	UsePathStyle    bool   `mapstructure:"use_path_style"` // For MinIO
}

// GCSConfig holds Google Cloud Storage configuration.
type GCSConfig struct {
	Project         string `mapstructure:"project"`
	CredentialsFile string `mapstructure:"credentials_file"`
	Bucket          string `mapstructure:"bucket"`
}

// LocalConfig holds local filesystem configuration.
type LocalConfig struct {
	RootPath string `mapstructure:"root_path"`
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	OAuth2       OAuth2Config  `mapstructure:"oauth2"`
	APIKey       APIKeyConfig  `mapstructure:"api_key"`
	TokenExpiry  time.Duration `mapstructure:"token_expiry"`
	SigningKey   string        `mapstructure:"signing_key"`
}

// OAuth2Config holds OAuth2 configuration.
type OAuth2Config struct {
	Enabled      bool   `mapstructure:"enabled"`
	Issuer       string `mapstructure:"issuer"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

// APIKeyConfig holds API key configuration.
type APIKeyConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// CatalogConfig holds catalog-specific configuration.
type CatalogConfig struct {
	Prefix             string        `mapstructure:"prefix"` // URL prefix for catalog
	DefaultWarehouse   string        `mapstructure:"default_warehouse"`
	LockTimeout        time.Duration `mapstructure:"lock_timeout"`
	LockRetryInterval  time.Duration `mapstructure:"lock_retry_interval"`
	MaxLockRetries     int           `mapstructure:"max_lock_retries"`
}

// Load reads configuration from file and environment variables.
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configuration file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/iceberg-catalog/")
	v.AddConfigPath("$HOME/.iceberg-catalog/")

	// Environment variables
	v.SetEnvPrefix("ICEBERG")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Explicitly bind nested environment variables (Viper doesn't auto-bind nested structs well)
	_ = v.BindEnv("auth.enabled", "ICEBERG_AUTH_ENABLED")
	_ = v.BindEnv("auth.oauth2.enabled", "ICEBERG_AUTH_OAUTH2_ENABLED")
	_ = v.BindEnv("auth.oauth2.client_id", "ICEBERG_AUTH_OAUTH2_CLIENT_ID")
	_ = v.BindEnv("auth.oauth2.client_secret", "ICEBERG_AUTH_OAUTH2_CLIENT_SECRET")
	_ = v.BindEnv("auth.token_expiry", "ICEBERG_AUTH_TOKEN_EXPIRY")
	_ = v.BindEnv("auth.signing_key", "ICEBERG_AUTH_SIGNING_KEY")
	_ = v.BindEnv("compat.polaris_enabled", "ICEBERG_COMPAT_POLARIS_ENABLED")

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
		// Config file not found is acceptable, use defaults + env vars
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8181)
	v.SetDefault("server.debug", false)
	v.SetDefault("server.version", "0.1.0")
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)
	v.SetDefault("server.idle_timeout", 120*time.Second)

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "iceberg")
	v.SetDefault("database.password", "iceberg")
	v.SetDefault("database.database", "iceberg_catalog")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 5*time.Minute)
	v.SetDefault("database.conn_max_idle_time", 5*time.Minute)

	// Storage defaults
	v.SetDefault("storage.type", "local")
	v.SetDefault("storage.warehouse", "/tmp/iceberg/warehouse")
	v.SetDefault("storage.local.root_path", "/tmp/iceberg/data")
	v.SetDefault("storage.s3.region", "us-east-1")
	v.SetDefault("storage.s3.use_path_style", false)

	// Auth defaults
	v.SetDefault("auth.enabled", false)
	v.SetDefault("auth.oauth2.enabled", false)
	v.SetDefault("auth.api_key.enabled", false)
	v.SetDefault("auth.token_expiry", 1*time.Hour)

	// Catalog defaults
	v.SetDefault("catalog.prefix", "")
	v.SetDefault("catalog.lock_timeout", 30*time.Second)
	v.SetDefault("catalog.lock_retry_interval", 100*time.Millisecond)
	v.SetDefault("catalog.max_lock_retries", 100)

	// Pool defaults (for sync.Pool optimization)
	v.SetDefault("pool.buffer_initial_size", 4096)  // 4KB - typical JSON metadata size
	v.SetDefault("pool.buffer_max_size", 65536)     // 64KB - discard oversized buffers

	// Compat defaults (compatibility layers disabled by default)
	v.SetDefault("compat.polaris_enabled", false)
}
