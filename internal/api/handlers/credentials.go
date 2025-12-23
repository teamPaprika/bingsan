package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/kimuyb/bingsan/internal/config"
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
		credentials := map[string]string{}

		switch cfg.Storage.Type {
		case "s3":
			// For S3, we could return temporary credentials via STS
			// For now, return the configured credentials if access delegation is vended-credentials
			if delegation == "vended-credentials" && cfg.Storage.S3.AccessKeyID != "" {
				credentials["s3.access-key-id"] = cfg.Storage.S3.AccessKeyID
				credentials["s3.secret-access-key"] = cfg.Storage.S3.SecretAccessKey
				if cfg.Storage.S3.Endpoint != "" {
					credentials["s3.endpoint"] = cfg.Storage.S3.Endpoint
				}
				if cfg.Storage.S3.Region != "" {
					credentials["s3.region"] = cfg.Storage.S3.Region
				}
				if cfg.Storage.S3.UsePathStyle {
					credentials["s3.path-style-access"] = "true"
				}
				// Add expiration info
				credentials["s3.session-token-expiration-ms"] = formatExpirationMs(1 * time.Hour)
			}
		case "gcs":
			// For GCS, we could return temporary credentials via Service Account impersonation
			if delegation == "vended-credentials" && cfg.Storage.GCS.CredentialsFile != "" {
				credentials["gcs.project-id"] = cfg.Storage.GCS.Project
				// Note: Actual credential vending would involve generating short-lived tokens
			}
		case "local":
			// Local storage doesn't need credentials
		}

		return c.JSON(LoadCredentialsResponse{
			Config: credentials,
		})
	}
}

// formatExpirationMs formats an expiration duration as milliseconds from now.
func formatExpirationMs(d time.Duration) string {
	expiration := time.Now().Add(d).UnixMilli()
	return fmt.Sprintf("%d", expiration)
}

