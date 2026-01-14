package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/teamPaprika/bingsan/internal/config"
	"github.com/teamPaprika/bingsan/internal/db"
)

// ContextKey is the type for context keys.
type ContextKey string

const (
	// ContextKeyPrincipal is the context key for the authenticated principal.
	ContextKeyPrincipal ContextKey = "principal"
)

// Principal represents an authenticated entity.
type Principal struct {
	ID       string
	Type     string // "api_key", "oauth2"
	Scopes   []string
	ClientID string
}

// Auth returns an authentication middleware.
func Auth(cfg *config.Config, database *db.DB) fiber.Handler {
	// Build skip paths list based on config
	skipPaths := buildSkipPaths(cfg.Compat.PolarisEnabled)

	return func(c *fiber.Ctx) error {
		// Skip auth for certain paths
		if shouldSkipAuth(c.Path(), skipPaths) {
			return c.Next()
		}

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return unauthorizedError(c, "missing authorization header")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			return unauthorizedError(c, "invalid authorization header format")
		}

		scheme := strings.ToLower(parts[0])
		token := parts[1]

		var principal *Principal
		var err error

		switch scheme {
		case "bearer":
			if cfg.Auth.OAuth2.Enabled {
				principal, err = validateBearerToken(c.Context(), database, token)
			} else if cfg.Auth.APIKey.Enabled {
				// Some clients send API keys as Bearer tokens
				principal, err = validateAPIKey(c.Context(), database, token)
			} else {
				return unauthorizedError(c, "bearer authentication not enabled")
			}
		case "x-api-key":
			if cfg.Auth.APIKey.Enabled {
				principal, err = validateAPIKey(c.Context(), database, token)
			} else {
				return unauthorizedError(c, "API key authentication not enabled")
			}
		default:
			return unauthorizedError(c, "unsupported authentication scheme")
		}

		if err != nil {
			return unauthorizedError(c, err.Error())
		}

		// Store principal in context
		c.Locals(string(ContextKeyPrincipal), principal)

		return c.Next()
	}
}

// buildSkipPaths returns the list of paths that should skip authentication.
func buildSkipPaths(polarisEnabled bool) []string {
	skipPaths := []string{
		"/health",
		"/ready",
		"/v1/config",
		"/v1/oauth/tokens",
	}

	// Add Polaris compatibility paths when enabled
	if polarisEnabled {
		skipPaths = append(skipPaths, "/api/catalog/v1/oauth/tokens")
	}

	return skipPaths
}

// shouldSkipAuth returns true if the path should skip authentication.
func shouldSkipAuth(path string, skipPaths []string) bool {
	for _, p := range skipPaths {
		if path == p {
			return true
		}
	}
	return false
}

// validateBearerToken validates an OAuth2 bearer token.
func validateBearerToken(ctx context.Context, database *db.DB, token string) (*Principal, error) {
	// Hash the token for lookup
	tokenHash := hashToken(token)

	var principal Principal
	err := database.Pool.QueryRow(ctx, `
		SELECT client_id, scopes
		FROM oauth_tokens
		WHERE access_token_hash = $1 AND expires_at > $2
	`, tokenHash, time.Now()).Scan(&principal.ClientID, &principal.Scopes)

	if err != nil {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
	}

	principal.ID = principal.ClientID
	principal.Type = "oauth2"

	return &principal, nil
}

// RotationGracePeriod is the duration during which both old and new keys are valid after rotation.
const RotationGracePeriod = 24 * time.Hour

// validateAPIKey validates an API key, supporting key rotation.
// During rotation, both the current key and the previous key are valid
// for a grace period (24 hours from rotation).
func validateAPIKey(ctx context.Context, database *db.DB, key string) (*Principal, error) {
	keyHash := hashToken(key)

	var principal Principal
	var expiresAt *time.Time
	var usedPreviousKey bool

	// First try to match against the current key
	err := database.Pool.QueryRow(ctx, `
		SELECT id, name, scopes, expires_at
		FROM api_keys
		WHERE key_hash = $1
	`, keyHash).Scan(&principal.ID, &principal.ClientID, &principal.Scopes, &expiresAt)

	if err != nil {
		// If not found, try matching against the previous key (rotation grace period)
		err = database.Pool.QueryRow(ctx, `
			SELECT id, name, scopes, expires_at
			FROM api_keys
			WHERE previous_key_hash = $1
			  AND rotated_at IS NOT NULL
			  AND rotated_at > $2
		`, keyHash, time.Now().Add(-RotationGracePeriod)).Scan(
			&principal.ID, &principal.ClientID, &principal.Scopes, &expiresAt)

		if err != nil {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid API key")
		}
		usedPreviousKey = true
	}

	// Check expiration
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return nil, fiber.NewError(fiber.StatusUnauthorized, "API key expired")
	}

	// Update last used timestamp
	_, _ = database.Pool.Exec(ctx, `
		UPDATE api_keys SET last_used_at = $1 WHERE id = $2
	`, time.Now(), principal.ID)

	principal.Type = "api_key"

	// Add indicator if using previous key (for logging/debugging)
	if usedPreviousKey {
		principal.Type = "api_key_rotated"
	}

	return &principal, nil
}

// hashToken creates a SHA-256 hash of a token for storage/lookup.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// unauthorizedError returns a standardized unauthorized error.
func unauthorizedError(c *fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": fiber.Map{
			"message": message,
			"type":    "NotAuthorizedException",
			"code":    401,
		},
	})
}

// GetPrincipal retrieves the authenticated principal from context.
func GetPrincipal(c *fiber.Ctx) *Principal {
	principal, ok := c.Locals(string(ContextKeyPrincipal)).(*Principal)
	if !ok {
		return nil
	}
	return principal
}
