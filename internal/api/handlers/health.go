package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/teamPaprika/bingsan/internal/db"
)

// HealthCheck returns a simple health check response.
func HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "healthy",
	})
}

// ReadyCheck checks if the service is ready to accept requests.
func ReadyCheck(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if database is available
		if database == nil || database.Pool == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "not ready",
				"checks": fiber.Map{
					"database": "not configured",
				},
			})
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		// Check database connectivity
		if err := database.Pool.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "not ready",
				"checks": fiber.Map{
					"database": "failed",
				},
			})
		}

		return c.JSON(fiber.Map{
			"status": "ready",
			"checks": fiber.Map{
				"database": "healthy",
			},
		})
	}
}
