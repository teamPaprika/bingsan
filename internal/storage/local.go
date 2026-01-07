package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kimuyb/bingsan/internal/config"
)

// LocalClient implements Client for local filesystem storage.
// Useful for development and testing.
type LocalClient struct {
	rootPath string
}

// NewLocalClient creates a new local filesystem storage client.
func NewLocalClient(cfg *config.LocalConfig) (*LocalClient, error) {
	// Ensure root path exists
	if cfg.RootPath != "" {
		if err := os.MkdirAll(cfg.RootPath, 0755); err != nil {
			return nil, fmt.Errorf("creating root path: %w", err)
		}
	}

	return &LocalClient{
		rootPath: cfg.RootPath,
	}, nil
}

// Read opens a local file for reading.
func (c *LocalClient) Read(_ context.Context, path string) (io.ReadCloser, error) {
	filePath := c.resolvePath(path)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}

	return file, nil
}

// Exists checks if a local file exists.
func (c *LocalClient) Exists(_ context.Context, path string) (bool, error) {
	filePath := c.resolvePath(path)

	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking %s: %w", filePath, err)
	}

	return true, nil
}

// Type returns the storage type identifier.
func (c *LocalClient) Type() string {
	return "local"
}

// resolvePath converts a storage path to a local filesystem path.
func (c *LocalClient) resolvePath(path string) string {
	// Strip file:// scheme if present
	path = strings.TrimPrefix(path, "file://")

	// If path is absolute, use it directly
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	// Otherwise, resolve relative to root path
	if c.rootPath != "" {
		return filepath.Join(c.rootPath, path)
	}

	return filepath.Clean(path)
}
