package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// PolarisCreateCatalog is a mock endpoint for Polaris Management API compatibility.
// Bingsan is a single-catalog system, so this just returns success.
// POST /api/management/v1/catalogs
func PolarisCreateCatalog(c *fiber.Ctx) error {
	// Parse the request to extract catalog name for response
	var req struct {
		Catalog struct {
			Type       string            `json:"type"`
			Name       string            `json:"name"`
			Properties map[string]string `json:"properties"`
			StorageConfigInfo struct {
				StorageType      string   `json:"storageType"`
				AllowedLocations []string `json:"allowedLocations,omitempty"`
			} `json:"storageConfigInfo"`
		} `json:"catalog"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    fiber.StatusBadRequest,
				"message": "invalid request body",
				"type":    "BadRequestException",
			},
		})
	}

	// Store the default base location
	defaultBaseLocation := req.Catalog.Properties["default-base-location"]
	if defaultBaseLocation == "" {
		defaultBaseLocation = "file:///tmp/iceberg/warehouse"
	}

	// Return a response that matches what Polaris returns
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"type":       req.Catalog.Type,
		"name":       req.Catalog.Name,
		"properties": req.Catalog.Properties,
		"storageConfigInfo": fiber.Map{
			"storageType":      req.Catalog.StorageConfigInfo.StorageType,
			"allowedLocations": []string{defaultBaseLocation},
		},
		"createTimestamp":     0,
		"lastUpdateTimestamp": 0,
		"entityVersion":       1,
	})
}

// PolarisGetCatalog is a mock endpoint for Polaris Management API compatibility.
// GET /api/management/v1/catalogs/:name
func PolarisGetCatalog(c *fiber.Ctx) error {
	name := c.Params("name")

	// Return a mock catalog response
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"type": "INTERNAL",
		"name": name,
		"properties": fiber.Map{
			"default-base-location": "file:///tmp/iceberg/warehouse",
		},
		"storageConfigInfo": fiber.Map{
			"storageType":      "FILE",
			"allowedLocations": []string{"file:///tmp/iceberg/warehouse"},
		},
		"createTimestamp":     0,
		"lastUpdateTimestamp": 0,
		"entityVersion":       1,
	})
}

// PolarisListCatalogs is a mock endpoint for Polaris Management API compatibility.
// GET /api/management/v1/catalogs
func PolarisListCatalogs(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"catalogs": []fiber.Map{
			{
				"type": "INTERNAL",
				"name": "default",
				"properties": fiber.Map{
					"default-base-location": "file:///tmp/iceberg/warehouse",
				},
				"storageConfigInfo": fiber.Map{
					"storageType":      "FILE",
					"allowedLocations": []string{"file:///tmp/iceberg/warehouse"},
				},
				"createTimestamp":     0,
				"lastUpdateTimestamp": 0,
				"entityVersion":       1,
			},
		},
	})
}
