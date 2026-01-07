package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	"github.com/kimuyb/bingsan/internal/config"
)

// S3Client implements Client for Amazon S3 and S3-compatible storage (MinIO, R2).
type S3Client struct {
	client *s3.Client
	cfg    *config.S3Config
}

// NewS3Client creates a new S3 storage client.
func NewS3Client(cfg *config.S3Config) (*S3Client, error) {
	ctx := context.Background()

	// Build AWS config options
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}

	// Use static credentials if configured
	if cfg.AccessKeyID != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	// Build S3 client options
	s3Opts := []func(*s3.Options){}

	// Custom endpoint (for MinIO, LocalStack, R2, etc.)
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	// Path-style access (required for MinIO)
	if cfg.UsePathStyle {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)

	return &S3Client{
		client: client,
		cfg:    cfg,
	}, nil
}

// Read opens an S3 object for reading.
func (c *S3Client) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	_, bucket, key, err := ParsePath(path)
	if err != nil {
		return nil, err
	}

	// Use configured bucket if path doesn't specify one
	if bucket == "" {
		bucket = c.cfg.Bucket
	}

	result, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("reading s3://%s/%s: %w", bucket, key, err)
	}

	return result.Body, nil
}

// Exists checks if an S3 object exists.
func (c *S3Client) Exists(ctx context.Context, path string) (bool, error) {
	_, bucket, key, err := ParsePath(path)
	if err != nil {
		return false, err
	}

	// Use configured bucket if path doesn't specify one
	if bucket == "" {
		bucket = c.cfg.Bucket
	}

	_, err = c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a "not found" error
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			if apiErr.ErrorCode() == "NotFound" || apiErr.ErrorCode() == "NoSuchKey" {
				return false, nil
			}
		}
		return false, fmt.Errorf("checking s3://%s/%s: %w", bucket, key, err)
	}

	return true, nil
}

// Type returns the storage type identifier.
func (c *S3Client) Type() string {
	return "s3"
}
