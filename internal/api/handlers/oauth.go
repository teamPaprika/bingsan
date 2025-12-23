package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/kimuyb/bingsan/internal/config"
	"github.com/kimuyb/bingsan/internal/db"
)

// TokenRequest represents an OAuth2 token exchange request.
type TokenRequest struct {
	GrantType    string `json:"grant_type" form:"grant_type"`
	Scope        string `json:"scope" form:"scope"`
	ClientID     string `json:"client_id" form:"client_id"`
	ClientSecret string `json:"client_secret" form:"client_secret"`
}

// TokenResponse represents an OAuth2 token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// ExchangeToken handles OAuth2 token exchange.
// POST /v1/oauth/tokens
// Note: This endpoint is marked as deprecated in the Iceberg spec
// but is still required for compatibility with some clients.
func ExchangeToken(cfg *config.Config, database *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req TokenRequest

		// Parse form data (OAuth2 uses form encoding)
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "invalid request body",
					"type":    "BadRequestException",
					"code":    400,
				},
			})
		}

		// Validate grant type
		if req.GrantType != "client_credentials" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "unsupported grant_type, only client_credentials is supported",
					"type":    "BadRequestException",
					"code":    400,
				},
			})
		}

		// Validate client credentials
		if req.ClientID == "" || req.ClientSecret == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "client_id and client_secret are required",
					"type":    "NotAuthorizedException",
					"code":    401,
				},
			})
		}

		// Verify client credentials against configured OAuth2 settings
		if !cfg.Auth.OAuth2.Enabled {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "OAuth2 authentication is not enabled",
					"type":    "NotAuthorizedException",
					"code":    401,
				},
			})
		}

		// For simple setup, check against configured client
		// In production, you'd want to look up clients from a database
		if req.ClientID != cfg.Auth.OAuth2.ClientID || req.ClientSecret != cfg.Auth.OAuth2.ClientSecret {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "invalid client credentials",
					"type":    "NotAuthorizedException",
					"code":    401,
				},
			})
		}

		// Generate access token
		accessToken, err := generateToken(32)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "failed to generate token",
					"type":    "ServerError",
					"code":    500,
				},
			})
		}

		// Calculate expiry
		expiresIn := int(cfg.Auth.TokenExpiry.Seconds())
		expiresAt := time.Now().Add(cfg.Auth.TokenExpiry)

		// Store token in database
		_, err = database.Pool.Exec(c.Context(), `
			INSERT INTO oauth_tokens (access_token_hash, client_id, scopes, expires_at)
			VALUES ($1, $2, $3, $4)
		`, accessToken, req.ClientID, []string{req.Scope}, expiresAt)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fiber.Map{
					"message": "failed to store token",
					"type":    "ServerError",
					"code":    500,
				},
			})
		}

		return c.JSON(TokenResponse{
			AccessToken: accessToken,
			TokenType:   "bearer",
			ExpiresIn:   expiresIn,
			Scope:       req.Scope,
		})
	}
}

// generateToken generates a random hex token.
func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
