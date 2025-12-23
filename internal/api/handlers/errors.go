package handlers

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL error codes
const (
	PgErrUniqueViolation = "23505"
)

// IsDuplicateError checks if the error is a PostgreSQL unique constraint violation.
func IsDuplicateError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == PgErrUniqueViolation
	}
	return false
}

// alreadyExistsError returns an AlreadyExistsException response.
func alreadyExistsError(c *fiber.Ctx, entityType, name string) error {
	return c.Status(fiber.StatusConflict).JSON(fiber.Map{
		"error": fiber.Map{
			"message": fmt.Sprintf("%s already exists: %s", entityType, name),
			"type":    "AlreadyExistsException",
			"code":    409,
		},
	})
}

// getStringFromMap safely extracts a string value from a map.
func getStringFromMap(m map[string]any, key string) (string, bool) {
	if m == nil {
		return "", false
	}
	val, ok := m[key]
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// validatePageSize validates and constrains pageSize to safe bounds.
func validatePageSize(pageSize int) int {
	if pageSize < 1 {
		return 1
	}
	if pageSize > 1000 {
		return 1000
	}
	return pageSize
}
