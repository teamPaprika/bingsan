package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gofiber/fiber/v2"
	"google.golang.org/api/iamcredentials/v1"
	"google.golang.org/api/option"

	"github.com/teamPaprika/bingsan/internal/config"
)

// LoadCredentialsResponse is the response for loading vended credentials.
type LoadCredentialsResponse struct {
	Config map[string]string `json:"config,omitempty"`
}

// LoadCredentials returns vended credentials for accessing table data.
// GET /v1/{prefix}/namespaces/{namespace}/tables/{table}/credentials
//
// This endpoint allows the catalog to vend temporary credentials
// for accessing the underlying storage (S3, GCS, etc.).
func LoadCredentials(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if access delegation is requested
		delegation := c.Get("X-Iceberg-Access-Delegation")
		if delegation == "" {
			// No delegation requested, return empty config
			return c.JSON(LoadCredentialsResponse{
				Config: map[string]string{},
			})
		}

		// Build credentials config based on storage type
		creds := map[string]string{}

		switch cfg.Storage.Type {
		case "s3":
			if delegation == "vended-credentials" {
				var err error
				creds, err = vendS3Credentials(c.Context(), cfg)
				if err != nil {
					slog.Error("failed to vend S3 credentials", "error", err)
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": fiber.Map{
							"message": "failed to vend credentials",
							"type":    "ServerError",
							"code":    500,
						},
					})
				}
			}
		case "gcs":
			if delegation == "vended-credentials" {
				var err error
				creds, err = vendGCSCredentials(c.Context(), cfg)
				if err != nil {
					slog.Error("failed to vend GCS credentials", "error", err)
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": fiber.Map{
							"message": "failed to vend credentials",
							"type":    "ServerError",
							"code":    500,
						},
					})
				}
			}
		case "local":
			// Local storage doesn't need credentials
		}

		return c.JSON(LoadCredentialsResponse{
			Config: creds,
		})
	}
}

// vendS3Credentials returns S3 credentials, using STS AssumeRole if configured.
func vendS3Credentials(ctx context.Context, cfg *config.Config) (map[string]string, error) {
	creds := map[string]string{}

	// If RoleARN is configured, use STS AssumeRole
	if cfg.Storage.S3.RoleARN != "" {
		return assumeRole(ctx, cfg)
	}

	// Otherwise, return static credentials if configured
	if cfg.Storage.S3.AccessKeyID != "" {
		creds["s3.access-key-id"] = cfg.Storage.S3.AccessKeyID
		creds["s3.secret-access-key"] = cfg.Storage.S3.SecretAccessKey
		if cfg.Storage.S3.Endpoint != "" {
			creds["s3.endpoint"] = cfg.Storage.S3.Endpoint
		}
		if cfg.Storage.S3.Region != "" {
			creds["s3.region"] = cfg.Storage.S3.Region
		}
		if cfg.Storage.S3.UsePathStyle {
			creds["s3.path-style-access"] = "true"
		}
		// Add expiration info (static credentials don't expire, but use 1 hour for cache refresh)
		creds["s3.session-token-expiration-ms"] = formatExpirationMs(1 * time.Hour)
	}

	return creds, nil
}

// assumeRole uses STS AssumeRole to get temporary credentials.
func assumeRole(ctx context.Context, cfg *config.Config) (map[string]string, error) {
	// Build AWS config options
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Storage.S3.Region),
	}

	// If static credentials are configured, use them as the base credentials for assuming the role
	if cfg.Storage.S3.AccessKeyID != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.Storage.S3.AccessKeyID,
				cfg.Storage.S3.SecretAccessKey,
				"",
			),
		))
	}

	// If custom endpoint is configured (MinIO, LocalStack, etc.)
	if cfg.Storage.S3.Endpoint != "" {
		opts = append(opts, awsconfig.WithEndpointResolver(
			aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
				if service == sts.ServiceID {
					return aws.Endpoint{
						URL:               cfg.Storage.S3.Endpoint,
						SigningRegion:     cfg.Storage.S3.Region,
						HostnameImmutable: true,
					}, nil
				}
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			}),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	stsClient := sts.NewFromConfig(awsCfg)

	// Calculate session duration in seconds (AWS STS requires int32)
	sessionDuration := cfg.Storage.S3.SessionDuration
	if sessionDuration == 0 {
		sessionDuration = 1 * time.Hour
	}
	durationSeconds := int32(sessionDuration.Seconds())

	// Build AssumeRole input
	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(cfg.Storage.S3.RoleARN),
		RoleSessionName: aws.String("iceberg-catalog"),
		DurationSeconds: aws.Int32(durationSeconds),
	}

	// Add external ID if configured (for cross-account access)
	if cfg.Storage.S3.ExternalID != "" {
		input.ExternalId = aws.String(cfg.Storage.S3.ExternalID)
	}

	result, err := stsClient.AssumeRole(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("assuming role: %w", err)
	}

	creds := map[string]string{
		"s3.access-key-id":              *result.Credentials.AccessKeyId,
		"s3.secret-access-key":          *result.Credentials.SecretAccessKey,
		"s3.session-token":              *result.Credentials.SessionToken,
		"s3.session-token-expiration-ms": formatExpirationFromTime(*result.Credentials.Expiration),
	}

	// Add region and endpoint info
	if cfg.Storage.S3.Region != "" {
		creds["s3.region"] = cfg.Storage.S3.Region
	}
	if cfg.Storage.S3.Endpoint != "" {
		creds["s3.endpoint"] = cfg.Storage.S3.Endpoint
	}
	if cfg.Storage.S3.UsePathStyle {
		creds["s3.path-style-access"] = "true"
	}

	return creds, nil
}

// vendGCSCredentials returns GCS credentials, using Service Account impersonation if configured.
func vendGCSCredentials(ctx context.Context, cfg *config.Config) (map[string]string, error) {
	creds := map[string]string{}

	if cfg.Storage.GCS.Project != "" {
		creds["gcs.project-id"] = cfg.Storage.GCS.Project
	}

	// If impersonation is configured, generate an access token
	if cfg.Storage.GCS.ImpersonateServiceAccount != "" {
		token, expiration, err := impersonateServiceAccount(ctx, cfg)
		if err != nil {
			return nil, err
		}
		creds["gcs.oauth2.token"] = token
		creds["gcs.oauth2.token-expiration-ms"] = formatExpirationFromTime(expiration)
	}

	return creds, nil
}

// impersonateServiceAccount generates an access token by impersonating a service account.
func impersonateServiceAccount(ctx context.Context, cfg *config.Config) (string, time.Time, error) {
	// Build options for the IAM credentials service
	opts := []option.ClientOption{}

	// If a credentials file is configured, use it
	if cfg.Storage.GCS.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.Storage.GCS.CredentialsFile))
	}

	// Create IAM Credentials service
	iamService, err := iamcredentials.NewService(ctx, opts...)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("creating IAM credentials service: %w", err)
	}

	// Build the resource name for the service account
	name := fmt.Sprintf("projects/-/serviceAccounts/%s", cfg.Storage.GCS.ImpersonateServiceAccount)

	// Calculate token lifetime
	lifetime := cfg.Storage.GCS.TokenLifetime
	if lifetime == 0 {
		lifetime = 1 * time.Hour
	}

	// Generate access token
	req := &iamcredentials.GenerateAccessTokenRequest{
		Scope: []string{
			"https://www.googleapis.com/auth/devstorage.read_write",
		},
		Lifetime: fmt.Sprintf("%ds", int(lifetime.Seconds())),
	}

	resp, err := iamService.Projects.ServiceAccounts.GenerateAccessToken(name, req).Context(ctx).Do()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generating access token: %w", err)
	}

	// Parse expiration time
	expireTime, err := time.Parse(time.RFC3339, resp.ExpireTime)
	if err != nil {
		// If parsing fails, use lifetime from now
		expireTime = time.Now().Add(lifetime)
	}

	return resp.AccessToken, expireTime, nil
}

// formatExpirationMs formats an expiration duration as milliseconds from now.
func formatExpirationMs(d time.Duration) string {
	expiration := time.Now().Add(d).UnixMilli()
	return fmt.Sprintf("%d", expiration)
}

// formatExpirationFromTime formats an expiration time as milliseconds.
func formatExpirationFromTime(t time.Time) string {
	return fmt.Sprintf("%d", t.UnixMilli())
}
