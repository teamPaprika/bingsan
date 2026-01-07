package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the catalog.
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Audit     AuditConfig     `mapstructure:"audit"`
	Catalog   CatalogConfig   `mapstructure:"catalog"`
	Pool      PoolConfig      `mapstructure:"pool"`
	Compat    CompatConfig    `mapstructure:"compat"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
	Scan      ScanConfig      `mapstructure:"scan"`
}

// TelemetryConfig holds OpenTelemetry configuration.
type TelemetryConfig struct {
	Enabled    bool    `mapstructure:"enabled"`
	Endpoint   string  `mapstructure:"endpoint"`    // OTLP HTTP endpoint
	SampleRate float64 `mapstructure:"sample_rate"` // 0.0-1.0 sampling rate
	Insecure   bool    `mapstructure:"insecure"`    // Use HTTP instead of HTTPS
}

// ScanConfig holds server-side scan planning configuration.
type ScanConfig struct {
	Enabled                bool          `mapstructure:"enabled"`                  // Enable scan planning
	AsyncThreshold         int           `mapstructure:"async_threshold"`          // Files above which async planning is used
	MaxConcurrentManifests int           `mapstructure:"max_concurrent_manifests"` // Parallel manifest reads
	PlanTaskBatchSize      int           `mapstructure:"plan_task_batch_size"`     // Files per plan-task
	PlanExpiry             time.Duration `mapstructure:"plan_expiry"`              // How long to keep plans
}

// ToScanPlannerConfig converts ScanConfig to scan.Config.
func (c ScanConfig) ToScanPlannerConfig() map[string]any {
	return map[string]any{
		"async_threshold":          c.AsyncThreshold,
		"max_concurrent_manifests": c.MaxConcurrentManifests,
		"plan_task_batch_size":     c.PlanTaskBatchSize,
		"plan_expiry":              c.PlanExpiry,
	}
}

// AuditConfig holds audit logging configuration.
type AuditConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	RetentionDays int    `mapstructure:"retention_days"` // Days to keep audit logs
	LogLevel      string `mapstructure:"log_level"`      // "all", "write", "manage"
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
	TLS          TLSConfig     `mapstructure:"tls"`
}

// TLSConfig holds TLS configuration.
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"` // Path to TLS certificate
	KeyFile  string `mapstructure:"key_file"`  // Path to TLS private key
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
	Endpoint        string        `mapstructure:"endpoint"`
	Region          string        `mapstructure:"region"`
	AccessKeyID     string        `mapstructure:"access_key_id"`
	SecretAccessKey string        `mapstructure:"secret_access_key"`
	Bucket          string        `mapstructure:"bucket"`
	UsePathStyle    bool          `mapstructure:"use_path_style"` // For MinIO
	RoleARN         string        `mapstructure:"role_arn"`       // For STS AssumeRole
	ExternalID      string        `mapstructure:"external_id"`    // Optional external ID for cross-account access
	SessionDuration time.Duration `mapstructure:"session_duration"` // Duration for assumed role (default 1h)
}

// GCSConfig holds Google Cloud Storage configuration.
type GCSConfig struct {
	Project                    string        `mapstructure:"project"`
	CredentialsFile            string        `mapstructure:"credentials_file"`
	Bucket                     string        `mapstructure:"bucket"`
	ImpersonateServiceAccount  string        `mapstructure:"impersonate_service_account"` // SA email for impersonation
	TokenLifetime              time.Duration `mapstructure:"token_lifetime"`              // Duration for impersonated tokens
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
	ACL          ACLConfig     `mapstructure:"acl"`
	TokenExpiry  time.Duration `mapstructure:"token_expiry"`
	SigningKey   string        `mapstructure:"signing_key"`
}

// ACLConfig holds namespace-level access control configuration.
type ACLConfig struct {
	Enabled           bool   `mapstructure:"enabled"`
	DefaultPermission string `mapstructure:"default_permission"` // "none", "read", "write", "manage"
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
	v.SetDefault("server.tls.enabled", false)

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
	v.SetDefault("storage.s3.session_duration", 1*time.Hour)
	v.SetDefault("storage.gcs.token_lifetime", 1*time.Hour)

	// Auth defaults
	v.SetDefault("auth.enabled", false)
	v.SetDefault("auth.oauth2.enabled", false)
	v.SetDefault("auth.api_key.enabled", false)
	v.SetDefault("auth.acl.enabled", false)
	v.SetDefault("auth.acl.default_permission", "none")
	v.SetDefault("auth.token_expiry", 1*time.Hour)

	// Audit defaults
	v.SetDefault("audit.enabled", false)
	v.SetDefault("audit.retention_days", 90)
	v.SetDefault("audit.log_level", "all")

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

	// Telemetry defaults (OpenTelemetry disabled by default)
	v.SetDefault("telemetry.enabled", false)
	v.SetDefault("telemetry.endpoint", "localhost:4318")
	v.SetDefault("telemetry.sample_rate", 0.1)
	v.SetDefault("telemetry.insecure", true)

	// Scan planning defaults
	v.SetDefault("scan.enabled", true)
	v.SetDefault("scan.async_threshold", 100)           // Files above which async is used
	v.SetDefault("scan.max_concurrent_manifests", 10)   // Parallel manifest reads
	v.SetDefault("scan.plan_task_batch_size", 50)       // Files per plan-task
	v.SetDefault("scan.plan_expiry", 1*time.Hour)       // Plans expire after 1 hour
}
