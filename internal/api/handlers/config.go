package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/kimuyb/bingsan/internal/config"
)

// CatalogConfig represents the catalog configuration response.
type CatalogConfig struct {
	Defaults  map[string]string `json:"defaults"`
	Overrides map[string]string `json:"overrides"`
	Endpoints []string          `json:"endpoints,omitempty"`
}

// GetConfig returns the catalog configuration.
// GET /v1/config
func GetConfig(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Build default configuration that clients should use
		defaults := map[string]string{
			"prefix": cfg.Catalog.Prefix,
		}

		// Add warehouse if configured
		if cfg.Storage.Warehouse != "" {
			defaults["warehouse"] = cfg.Storage.Warehouse
		}

		// Add storage-specific defaults
		switch cfg.Storage.Type {
		case "s3":
			if cfg.Storage.S3.Endpoint != "" {
				defaults["s3.endpoint"] = cfg.Storage.S3.Endpoint
			}
			if cfg.Storage.S3.Region != "" {
				defaults["s3.region"] = cfg.Storage.S3.Region
			}
			if cfg.Storage.S3.UsePathStyle {
				defaults["s3.path-style-access"] = "true"
			}
		case "gcs":
			if cfg.Storage.GCS.Project != "" {
				defaults["gcs.project-id"] = cfg.Storage.GCS.Project
			}
		}

		// Overrides take precedence over client config
		overrides := map[string]string{}

		// List of supported endpoints
		endpoints := []string{
			"GET /v1/config",
			"POST /v1/oauth/tokens",
			// Namespace endpoints
			"GET /v1/{prefix}/namespaces",
			"POST /v1/{prefix}/namespaces",
			"GET /v1/{prefix}/namespaces/{namespace}",
			"HEAD /v1/{prefix}/namespaces/{namespace}",
			"DELETE /v1/{prefix}/namespaces/{namespace}",
			"POST /v1/{prefix}/namespaces/{namespace}/properties",
			// Table endpoints
			"GET /v1/{prefix}/namespaces/{namespace}/tables",
			"POST /v1/{prefix}/namespaces/{namespace}/tables",
			"GET /v1/{prefix}/namespaces/{namespace}/tables/{table}",
			"POST /v1/{prefix}/namespaces/{namespace}/tables/{table}",
			"DELETE /v1/{prefix}/namespaces/{namespace}/tables/{table}",
			"HEAD /v1/{prefix}/namespaces/{namespace}/tables/{table}",
			"POST /v1/{prefix}/namespaces/{namespace}/register",
			"POST /v1/{prefix}/tables/rename",
			"GET /v1/{prefix}/namespaces/{namespace}/tables/{table}/credentials",
			"POST /v1/{prefix}/namespaces/{namespace}/tables/{table}/metrics",
			// Scan planning endpoints
			"POST /v1/{prefix}/namespaces/{namespace}/tables/{table}/plan",
			"GET /v1/{prefix}/namespaces/{namespace}/tables/{table}/plan/{plan-id}",
			"DELETE /v1/{prefix}/namespaces/{namespace}/tables/{table}/plan/{plan-id}",
			"POST /v1/{prefix}/namespaces/{namespace}/tables/{table}/tasks",
			// Transaction endpoint
			"POST /v1/{prefix}/transactions/commit",
			// View endpoints
			"GET /v1/{prefix}/namespaces/{namespace}/views",
			"POST /v1/{prefix}/namespaces/{namespace}/views",
			"GET /v1/{prefix}/namespaces/{namespace}/views/{view}",
			"POST /v1/{prefix}/namespaces/{namespace}/views/{view}",
			"DELETE /v1/{prefix}/namespaces/{namespace}/views/{view}",
			"HEAD /v1/{prefix}/namespaces/{namespace}/views/{view}",
			"POST /v1/{prefix}/views/rename",
		}

		return c.JSON(CatalogConfig{
			Defaults:  defaults,
			Overrides: overrides,
			Endpoints: endpoints,
		})
	}
}
