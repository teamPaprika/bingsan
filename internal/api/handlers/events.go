package handlers

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"github.com/teamPaprika/bingsan/internal/config"
	"github.com/teamPaprika/bingsan/internal/db"
	"github.com/teamPaprika/bingsan/internal/events"
)

// WebSocketUpgrade is middleware that checks for WebSocket upgrade requests.
func WebSocketUpgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// EventStream handles WebSocket connections for streaming catalog events.
// GET /v1/events/stream?token=xxx&namespace=xxx&table=xxx
func EventStream(broker *events.Broker, cfg *config.Config, database *db.DB) fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		// Extract query parameters
		token := c.Query("token")
		namespace := c.Query("namespace")
		table := c.Query("table")

		// Authenticate if auth is enabled
		if cfg.Auth.Enabled {
			if !authenticateWebSocket(c, token, cfg, database) {
				return
			}
		}

		// Determine subscription topic
		topic := determineTopic(namespace, table)

		slog.Info("websocket connected",
			"remote_addr", c.RemoteAddr().String(),
			"topic", topic,
		)

		// Subscribe to events
		sub := broker.Subscribe(topic)
		if sub == nil {
			slog.Error("broker closed, cannot subscribe")
			return
		}
		defer sub.Unsubscribe()

		// Send connection confirmation
		if err := c.WriteJSON(map[string]any{
			"type":    "connected",
			"topic":   topic,
			"message": "Connected to event stream",
		}); err != nil {
			slog.Error("failed to send connection message", "error", err)
			return
		}

		// Create a done channel for cleanup
		done := make(chan struct{})

		// Handle incoming messages (for ping/pong and close detection)
		go func() {
			defer close(done)
			for {
				_, _, err := c.ReadMessage()
				if err != nil {
					// Connection closed
					return
				}
			}
		}()

		// Stream events to client
		for {
			select {
			case event, ok := <-sub.Events():
				if !ok {
					// Channel closed
					return
				}
				if err := c.WriteJSON(event); err != nil {
					slog.Debug("websocket write error", "error", err)
					return
				}
			case <-done:
				// Client disconnected
				slog.Info("websocket disconnected", "remote_addr", c.RemoteAddr().String())
				return
			}
		}
	}, websocket.Config{
		EnableCompression: true,
		Origins:           []string{"*"}, // Configure for production
		RecoverHandler: func(c *websocket.Conn) {
			slog.Error("websocket panic recovered", "remote_addr", c.RemoteAddr().String())
		},
	})
}

// authenticateWebSocket validates the token for WebSocket connections.
func authenticateWebSocket(c *websocket.Conn, token string, cfg *config.Config, database *db.DB) bool {
	if token == "" {
		_ = c.WriteJSON(map[string]any{
			"type":    "error",
			"code":    401,
			"message": "Authentication required. Provide token as query parameter.",
		})
		c.Close()
		return false
	}

	// Remove "Bearer_" prefix if present (URL-safe format)
	token = strings.TrimPrefix(token, "Bearer_")
	token = strings.TrimPrefix(token, "Bearer ")

	// Use a background context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if it's a valid API key
	var keyID string
	err := database.Pool.QueryRow(ctx, `
		SELECT id FROM api_keys
		WHERE key_hash = encode(sha256($1::bytea), 'hex')
		AND (expires_at IS NULL OR expires_at > NOW())
	`, token).Scan(&keyID)

	if err != nil {
		// Try OAuth token
		err = database.Pool.QueryRow(ctx, `
			SELECT id FROM oauth_tokens
			WHERE access_token_hash = encode(sha256($1::bytea), 'hex')
			AND expires_at > NOW()
		`, token).Scan(&keyID)
	}

	if err != nil {
		_ = c.WriteJSON(map[string]any{
			"type":    "error",
			"code":    403,
			"message": "Invalid or expired token",
		})
		c.Close()
		return false
	}

	return true
}

// determineTopic determines the subscription topic from query parameters.
func determineTopic(namespace, table string) string {
	if namespace != "" && table != "" {
		return namespace + "/" + table
	}
	if namespace != "" {
		return namespace
	}
	return "*"
}
