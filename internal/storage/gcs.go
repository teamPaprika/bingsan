package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/kimuyb/bingsan/internal/config"
)

// GCSClient implements Client for Google Cloud Storage.
type GCSClient struct {
	client *storage.Client
	cfg    *config.GCSConfig
}

// NewGCSClient creates a new GCS storage client.
func NewGCSClient(cfg *config.GCSConfig) (*GCSClient, error) {
	ctx := context.Background()

	// Build client options
	opts := []option.ClientOption{}

	// Use credentials file if configured
	if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating GCS client: %w", err)
	}

	return &GCSClient{
		client: client,
		cfg:    cfg,
	}, nil
}

// Read opens a GCS object for reading.
func (c *GCSClient) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	_, bucket, key, err := ParsePath(path)
	if err != nil {
		return nil, err
	}

	// Use configured bucket if path doesn't specify one
	if bucket == "" {
		bucket = c.cfg.Bucket
	}

	reader, err := c.client.Bucket(bucket).Object(key).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading gs://%s/%s: %w", bucket, key, err)
	}

	return reader, nil
}

// Exists checks if a GCS object exists.
func (c *GCSClient) Exists(ctx context.Context, path string) (bool, error) {
	_, bucket, key, err := ParsePath(path)
	if err != nil {
		return false, err
	}

	// Use configured bucket if path doesn't specify one
	if bucket == "" {
		bucket = c.cfg.Bucket
	}

	_, err = c.client.Bucket(bucket).Object(key).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("checking gs://%s/%s: %w", bucket, key, err)
	}

	return true, nil
}

// Type returns the storage type identifier.
func (c *GCSClient) Type() string {
	return "gcs"
}

// Close releases resources held by the GCS client.
func (c *GCSClient) Close() error {
	return c.client.Close()
}
