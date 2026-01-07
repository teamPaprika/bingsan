// Package storage provides object storage abstractions for reading Iceberg metadata files.
package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/kimuyb/bingsan/internal/config"
)

// Client provides read-only access to object storage for Iceberg metadata files.
type Client interface {
	// Read opens an object for reading.
	// The caller must close the returned ReadCloser.
	Read(ctx context.Context, path string) (io.ReadCloser, error)

	// Exists checks if an object exists at the given path.
	Exists(ctx context.Context, path string) (bool, error)

	// Type returns the storage type identifier (s3, gcs, local).
	Type() string
}

// NewClient creates a storage client based on the configuration.
// The path parameter is used to determine the appropriate client when multiple
// storage backends are configured (e.g., a path starting with "s3://" uses S3).
func NewClient(cfg *config.StorageConfig, path string) (Client, error) {
	// Determine storage type from path scheme or config
	storageType := detectStorageType(cfg, path)

	switch storageType {
	case "s3":
		return NewS3Client(&cfg.S3)
	case "gcs":
		return NewGCSClient(&cfg.GCS)
	case "local":
		return NewLocalClient(&cfg.Local)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}

// detectStorageType determines the storage backend from a path or config.
func detectStorageType(cfg *config.StorageConfig, path string) string {
	// Check path scheme first
	if strings.HasPrefix(path, "s3://") || strings.HasPrefix(path, "s3a://") {
		return "s3"
	}
	if strings.HasPrefix(path, "gs://") {
		return "gcs"
	}
	if strings.HasPrefix(path, "file://") || strings.HasPrefix(path, "/") {
		return "local"
	}

	// Fall back to configured type
	return cfg.Type
}

// ParsePath extracts bucket and key from a storage path.
// Supports s3://bucket/key, gs://bucket/key, and file:///path formats.
func ParsePath(path string) (scheme, bucket, key string, err error) {
	// Handle file paths without scheme
	if strings.HasPrefix(path, "/") {
		return "file", "", path, nil
	}

	u, err := url.Parse(path)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid path: %w", err)
	}

	scheme = u.Scheme
	bucket = u.Host

	// Handle s3a:// (Hadoop S3A filesystem)
	if scheme == "s3a" {
		scheme = "s3"
	}

	// Key is the path without leading slash
	key = strings.TrimPrefix(u.Path, "/")

	return scheme, bucket, key, nil
}
