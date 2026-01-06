package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		config   DatabaseConfig
		expected string
	}{
		{
			name: "default values",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "iceberg",
				Password: "secret",
				Database: "iceberg_catalog",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=iceberg password=secret dbname=iceberg_catalog sslmode=disable",
		},
		{
			name: "custom host and port",
			config: DatabaseConfig{
				Host:     "db.example.com",
				Port:     5433,
				User:     "admin",
				Password: "p@ss",
				Database: "mydb",
				SSLMode:  "require",
			},
			expected: "host=db.example.com port=5433 user=admin password=p@ss dbname=mydb sslmode=require",
		},
		{
			name: "empty password",
			config: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "",
				Database: "db",
				SSLMode:  "prefer",
			},
			expected: "host=localhost port=5432 user=user password= dbname=db sslmode=prefer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.config.DSN()
			assert.Equal(t, tt.expected, dsn)
		})
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	// Clear environment to ensure defaults are used
	origVars := clearConfigEnvVars()
	defer restoreEnvVars(origVars)

	// Create a temporary directory without config file
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Server defaults
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8181, cfg.Server.Port)
	assert.False(t, cfg.Server.Debug)
	assert.Equal(t, "0.1.0", cfg.Server.Version)
	assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 120*time.Second, cfg.Server.IdleTimeout)
	assert.False(t, cfg.Server.TLS.Enabled)

	// Database defaults
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "iceberg", cfg.Database.User)
	assert.Equal(t, "iceberg", cfg.Database.Password)
	assert.Equal(t, "iceberg_catalog", cfg.Database.Database)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxIdleTime)

	// Storage defaults
	assert.Equal(t, "local", cfg.Storage.Type)
	assert.Equal(t, "/tmp/iceberg/warehouse", cfg.Storage.Warehouse)
	assert.Equal(t, "/tmp/iceberg/data", cfg.Storage.Local.RootPath)
	assert.Equal(t, "us-east-1", cfg.Storage.S3.Region)
	assert.False(t, cfg.Storage.S3.UsePathStyle)
	assert.Equal(t, 1*time.Hour, cfg.Storage.S3.SessionDuration)
	assert.Equal(t, 1*time.Hour, cfg.Storage.GCS.TokenLifetime)

	// Auth defaults
	assert.False(t, cfg.Auth.Enabled)
	assert.False(t, cfg.Auth.OAuth2.Enabled)
	assert.False(t, cfg.Auth.APIKey.Enabled)
	assert.False(t, cfg.Auth.ACL.Enabled)
	assert.Equal(t, "none", cfg.Auth.ACL.DefaultPermission)
	assert.Equal(t, 1*time.Hour, cfg.Auth.TokenExpiry)

	// Audit defaults
	assert.False(t, cfg.Audit.Enabled)
	assert.Equal(t, 90, cfg.Audit.RetentionDays)
	assert.Equal(t, "all", cfg.Audit.LogLevel)

	// Catalog defaults
	assert.Equal(t, "", cfg.Catalog.Prefix)
	assert.Equal(t, 30*time.Second, cfg.Catalog.LockTimeout)
	assert.Equal(t, 100*time.Millisecond, cfg.Catalog.LockRetryInterval)
	assert.Equal(t, 100, cfg.Catalog.MaxLockRetries)

	// Pool defaults
	assert.Equal(t, 4096, cfg.Pool.BufferInitialSize)
	assert.Equal(t, 65536, cfg.Pool.BufferMaxSize)

	// Compat defaults
	assert.False(t, cfg.Compat.PolarisEnabled)

	// Telemetry defaults
	assert.False(t, cfg.Telemetry.Enabled)
	assert.Equal(t, "localhost:4318", cfg.Telemetry.Endpoint)
	assert.Equal(t, 0.1, cfg.Telemetry.SampleRate)
	assert.True(t, cfg.Telemetry.Insecure)
}

func TestLoad_FromConfigFile(t *testing.T) {
	origVars := clearConfigEnvVars()
	defer restoreEnvVars(origVars)

	tmpDir := t.TempDir()
	configContent := `
server:
  host: "127.0.0.1"
  port: 9000
  debug: true

database:
  host: "pghost"
  port: 5433
  user: "testuser"
  password: "testpass"
  database: "testdb"

auth:
  enabled: true
  oauth2:
    enabled: true
    client_id: "my-client"
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 9000, cfg.Server.Port)
	assert.True(t, cfg.Server.Debug)
	assert.Equal(t, "pghost", cfg.Database.Host)
	assert.Equal(t, 5433, cfg.Database.Port)
	assert.Equal(t, "testuser", cfg.Database.User)
	assert.Equal(t, "testpass", cfg.Database.Password)
	assert.Equal(t, "testdb", cfg.Database.Database)
	assert.True(t, cfg.Auth.Enabled)
	assert.True(t, cfg.Auth.OAuth2.Enabled)
	assert.Equal(t, "my-client", cfg.Auth.OAuth2.ClientID)
}

func TestLoad_FromEnvVars(t *testing.T) {
	origVars := clearConfigEnvVars()
	defer restoreEnvVars(origVars)

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	os.Setenv("ICEBERG_SERVER_PORT", "9999")
	os.Setenv("ICEBERG_AUTH_ENABLED", "true")
	os.Setenv("ICEBERG_AUTH_OAUTH2_ENABLED", "true")
	os.Setenv("ICEBERG_AUTH_OAUTH2_CLIENT_ID", "env-client-id")
	os.Setenv("ICEBERG_COMPAT_POLARIS_ENABLED", "true")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 9999, cfg.Server.Port)
	assert.True(t, cfg.Auth.Enabled)
	assert.True(t, cfg.Auth.OAuth2.Enabled)
	assert.Equal(t, "env-client-id", cfg.Auth.OAuth2.ClientID)
	assert.True(t, cfg.Compat.PolarisEnabled)
}

func TestLoad_EnvOverridesConfig(t *testing.T) {
	origVars := clearConfigEnvVars()
	defer restoreEnvVars(origVars)

	tmpDir := t.TempDir()
	configContent := `
server:
  port: 8000

auth:
  enabled: false
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Environment should override config file
	os.Setenv("ICEBERG_SERVER_PORT", "9999")
	os.Setenv("ICEBERG_AUTH_ENABLED", "true")

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 9999, cfg.Server.Port)
	assert.True(t, cfg.Auth.Enabled)
}

func TestLoad_InvalidConfigFile(t *testing.T) {
	origVars := clearConfigEnvVars()
	defer restoreEnvVars(origVars)

	tmpDir := t.TempDir()
	configContent := `
server:
  port: "not a number"  # Invalid: should be int
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	require.Error(t, err)
	assert.Nil(t, cfg)
}

func TestConfig_Structs(t *testing.T) {
	t.Run("TelemetryConfig", func(t *testing.T) {
		cfg := TelemetryConfig{
			Enabled:    true,
			Endpoint:   "localhost:4317",
			SampleRate: 0.5,
			Insecure:   false,
		}
		assert.True(t, cfg.Enabled)
		assert.Equal(t, "localhost:4317", cfg.Endpoint)
		assert.Equal(t, 0.5, cfg.SampleRate)
		assert.False(t, cfg.Insecure)
	})

	t.Run("AuditConfig", func(t *testing.T) {
		cfg := AuditConfig{
			Enabled:       true,
			RetentionDays: 30,
			LogLevel:      "write",
		}
		assert.True(t, cfg.Enabled)
		assert.Equal(t, 30, cfg.RetentionDays)
		assert.Equal(t, "write", cfg.LogLevel)
	})

	t.Run("TLSConfig", func(t *testing.T) {
		cfg := TLSConfig{
			Enabled:  true,
			CertFile: "/path/to/cert.pem",
			KeyFile:  "/path/to/key.pem",
		}
		assert.True(t, cfg.Enabled)
		assert.Equal(t, "/path/to/cert.pem", cfg.CertFile)
		assert.Equal(t, "/path/to/key.pem", cfg.KeyFile)
	})

	t.Run("S3Config", func(t *testing.T) {
		cfg := S3Config{
			Endpoint:        "http://localhost:9000",
			Region:          "us-west-2",
			AccessKeyID:     "access",
			SecretAccessKey: "secret",
			Bucket:          "mybucket",
			UsePathStyle:    true,
			RoleARN:         "arn:aws:iam::123456789012:role/MyRole",
			ExternalID:      "ext-id",
			SessionDuration: 2 * time.Hour,
		}
		assert.Equal(t, "http://localhost:9000", cfg.Endpoint)
		assert.Equal(t, "us-west-2", cfg.Region)
		assert.Equal(t, "access", cfg.AccessKeyID)
		assert.Equal(t, "secret", cfg.SecretAccessKey)
		assert.Equal(t, "mybucket", cfg.Bucket)
		assert.True(t, cfg.UsePathStyle)
		assert.Equal(t, "arn:aws:iam::123456789012:role/MyRole", cfg.RoleARN)
		assert.Equal(t, "ext-id", cfg.ExternalID)
		assert.Equal(t, 2*time.Hour, cfg.SessionDuration)
	})

	t.Run("GCSConfig", func(t *testing.T) {
		cfg := GCSConfig{
			Project:                   "my-project",
			CredentialsFile:           "/path/to/creds.json",
			Bucket:                    "mybucket",
			ImpersonateServiceAccount: "sa@project.iam.gserviceaccount.com",
			TokenLifetime:             2 * time.Hour,
		}
		assert.Equal(t, "my-project", cfg.Project)
		assert.Equal(t, "/path/to/creds.json", cfg.CredentialsFile)
		assert.Equal(t, "mybucket", cfg.Bucket)
		assert.Equal(t, "sa@project.iam.gserviceaccount.com", cfg.ImpersonateServiceAccount)
		assert.Equal(t, 2*time.Hour, cfg.TokenLifetime)
	})

	t.Run("ACLConfig", func(t *testing.T) {
		cfg := ACLConfig{
			Enabled:           true,
			DefaultPermission: "read",
		}
		assert.True(t, cfg.Enabled)
		assert.Equal(t, "read", cfg.DefaultPermission)
	})
}

// Helper functions

func clearConfigEnvVars() map[string]string {
	envVars := []string{
		"ICEBERG_SERVER_HOST",
		"ICEBERG_SERVER_PORT",
		"ICEBERG_AUTH_ENABLED",
		"ICEBERG_AUTH_OAUTH2_ENABLED",
		"ICEBERG_AUTH_OAUTH2_CLIENT_ID",
		"ICEBERG_AUTH_OAUTH2_CLIENT_SECRET",
		"ICEBERG_AUTH_TOKEN_EXPIRY",
		"ICEBERG_AUTH_SIGNING_KEY",
		"ICEBERG_COMPAT_POLARIS_ENABLED",
		"ICEBERG_DATABASE_HOST",
		"ICEBERG_DATABASE_PORT",
	}

	orig := make(map[string]string)
	for _, v := range envVars {
		orig[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	return orig
}

func restoreEnvVars(vars map[string]string) {
	for k, v := range vars {
		if v == "" {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, v)
		}
	}
}
