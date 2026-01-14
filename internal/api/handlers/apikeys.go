package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/teamPaprika/bingsan/internal/api/middleware"
	"github.com/teamPaprika/bingsan/internal/db"
)

// APIKeyResponse represents an API key in responses.
type APIKeyResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Key        string     `json:"key,omitempty"` // Only returned on creation/rotation
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RotatedAt  *time.Time `json:"rotated_at,omitempty"`
}

// CreateAPIKeyRequest is the request body for creating an API key.
type CreateAPIKeyRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresIn *string  `json:"expires_in,omitempty"` // e.g., "30d", "1y"
}

// ListAPIKeys returns all API keys for the authenticated user.
// GET /v1/api-keys
func ListAPIKeys(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		principal := middleware.GetPrincipal(c)
		if principal == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"message": "unauthorized", "type": "NotAuthorizedException", "code": 401},
			})
		}

		rows, err := database.Pool.Query(c.Context(), `
			SELECT id, name, scopes, expires_at, created_at, last_used_at, rotated_at
			FROM api_keys
			WHERE (expires_at IS NULL OR expires_at > NOW())
			ORDER BY created_at DESC
		`)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fiber.Map{"message": "failed to list API keys", "type": "ServerError", "code": 500},
			})
		}
		defer rows.Close()

		var keys []APIKeyResponse
		for rows.Next() {
			var key APIKeyResponse
			if err := rows.Scan(&key.ID, &key.Name, &key.Scopes, &key.ExpiresAt,
				&key.CreatedAt, &key.LastUsedAt, &key.RotatedAt); err != nil {
				continue
			}
			keys = append(keys, key)
		}

		if keys == nil {
			keys = []APIKeyResponse{}
		}

		return c.JSON(fiber.Map{"keys": keys})
	}
}

// CreateAPIKey creates a new API key.
// POST /v1/api-keys
func CreateAPIKey(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		principal := middleware.GetPrincipal(c)
		if principal == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"message": "unauthorized", "type": "NotAuthorizedException", "code": 401},
			})
		}

		var req CreateAPIKeyRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"message": "invalid request body", "type": "BadRequestException", "code": 400},
			})
		}

		if req.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"message": "name is required", "type": "BadRequestException", "code": 400},
			})
		}

		// Generate a secure random key
		rawKey, err := generateAPIKey()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fiber.Map{"message": "failed to generate API key", "type": "ServerError", "code": 500},
			})
		}

		// Hash the key for storage
		keyHash := hashAPIKey(rawKey)

		// Parse expiration
		var expiresAt *time.Time
		if req.ExpiresIn != nil {
			dur, err := parseDuration(*req.ExpiresIn)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": fiber.Map{"message": "invalid expires_in format", "type": "BadRequestException", "code": 400},
				})
			}
			t := time.Now().Add(dur)
			expiresAt = &t
		}

		// Default scopes
		scopes := req.Scopes
		if len(scopes) == 0 {
			scopes = []string{"catalog"}
		}

		id := uuid.New().String()
		now := time.Now()

		_, err = database.Pool.Exec(c.Context(), `
			INSERT INTO api_keys (id, name, key_hash, scopes, expires_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, id, req.Name, keyHash, scopes, expiresAt, now)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fiber.Map{"message": "failed to create API key", "type": "ServerError", "code": 500},
			})
		}

		return c.Status(fiber.StatusCreated).JSON(APIKeyResponse{
			ID:        id,
			Name:      req.Name,
			Key:       rawKey, // Only returned on creation
			Scopes:    scopes,
			ExpiresAt: expiresAt,
			CreatedAt: now,
		})
	}
}

// RotateAPIKey generates a new key for an existing API key.
// The old key remains valid for a grace period (24 hours).
// POST /v1/api-keys/:id/rotate
func RotateAPIKey(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		principal := middleware.GetPrincipal(c)
		if principal == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"message": "unauthorized", "type": "NotAuthorizedException", "code": 401},
			})
		}

		keyID := c.Params("id")
		if keyID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"message": "key ID is required", "type": "BadRequestException", "code": 400},
			})
		}

		// Get current key hash
		var currentKeyHash string
		var name string
		var scopes []string
		var expiresAt *time.Time
		var createdAt time.Time

		err := database.Pool.QueryRow(c.Context(), `
			SELECT key_hash, name, scopes, expires_at, created_at
			FROM api_keys
			WHERE id = $1
		`, keyID).Scan(&currentKeyHash, &name, &scopes, &expiresAt, &createdAt)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fiber.Map{"message": "API key not found", "type": "NotFoundException", "code": 404},
			})
		}

		// Generate new key
		newRawKey, err := generateAPIKey()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fiber.Map{"message": "failed to generate new API key", "type": "ServerError", "code": 500},
			})
		}

		newKeyHash := hashAPIKey(newRawKey)
		now := time.Now()

		// Update key: move current hash to previous, set new hash
		_, err = database.Pool.Exec(c.Context(), `
			UPDATE api_keys
			SET key_hash = $1,
			    previous_key_hash = $2,
			    rotated_at = $3
			WHERE id = $4
		`, newKeyHash, currentKeyHash, now, keyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fiber.Map{"message": "failed to rotate API key", "type": "ServerError", "code": 500},
			})
		}

		return c.JSON(APIKeyResponse{
			ID:        keyID,
			Name:      name,
			Key:       newRawKey, // Return the new key
			Scopes:    scopes,
			ExpiresAt: expiresAt,
			CreatedAt: createdAt,
			RotatedAt: &now,
		})
	}
}

// DeleteAPIKey deletes an API key.
// DELETE /v1/api-keys/:id
func DeleteAPIKey(database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		principal := middleware.GetPrincipal(c)
		if principal == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{"message": "unauthorized", "type": "NotAuthorizedException", "code": 401},
			})
		}

		keyID := c.Params("id")
		if keyID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{"message": "key ID is required", "type": "BadRequestException", "code": 400},
			})
		}

		result, err := database.Pool.Exec(c.Context(), `
			DELETE FROM api_keys WHERE id = $1
		`, keyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fiber.Map{"message": "failed to delete API key", "type": "ServerError", "code": 500},
			})
		}

		if result.RowsAffected() == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": fiber.Map{"message": "API key not found", "type": "NotFoundException", "code": 404},
			})
		}

		return c.SendStatus(fiber.StatusNoContent)
	}
}

// generateAPIKey generates a secure random API key.
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "bingsan_" + base64.RawURLEncoding.EncodeToString(bytes), nil
}

// hashAPIKey creates a SHA-256 hash of an API key.
func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// parseDuration parses a duration string like "30d", "1y", "24h".
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid duration")
	}

	unit := s[len(s)-1]
	value := s[:len(s)-1]

	var multiplier time.Duration
	switch unit {
	case 'd':
		multiplier = 24 * time.Hour
	case 'w':
		multiplier = 7 * 24 * time.Hour
	case 'm':
		multiplier = 30 * 24 * time.Hour
	case 'y':
		multiplier = 365 * 24 * time.Hour
	case 'h':
		multiplier = time.Hour
	default:
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid duration unit")
	}

	var n int
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0, fiber.NewError(fiber.StatusBadRequest, "invalid duration")
		}
		n = n*10 + int(c-'0')
	}

	return time.Duration(n) * multiplier, nil
}
