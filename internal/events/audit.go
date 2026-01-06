package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/kimuyb/bingsan/internal/db"
)

// AuditLogger logs audit events to the database.
type AuditLogger struct {
	db      *db.DB
	broker  *Broker
	enabled bool
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(database *db.DB, broker *Broker, enabled bool) *AuditLogger {
	return &AuditLogger{
		db:      database,
		broker:  broker,
		enabled: enabled,
	}
}

// Log records an audit event to the database and publishes it to the broker.
func (a *AuditLogger) Log(ctx context.Context, event *AuditEvent) error {
	if !a.enabled {
		// Still publish to broker for real-time streaming
		if a.broker != nil {
			a.broker.Publish(event.CatalogEvent)
		}
		return nil
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	// Insert audit log entry
	_, err = a.db.Pool.Exec(ctx, `
		INSERT INTO audit_log (
			event_type, event_id, timestamp, namespace, table_name, view_name,
			actor, actor_type, ip_address, user_agent, request_id, response_code, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`,
		string(event.Type),
		event.ID,
		event.Timestamp,
		nullIfEmpty(event.Namespace),
		nullIfEmpty(event.Table),
		nullIfEmpty(event.View),
		event.Actor,
		event.ActorType,
		nullIfEmpty(event.IPAddress),
		nullIfEmpty(event.UserAgent),
		nullIfEmpty(event.RequestID),
		nullIfZero(event.ResponseCode),
		metadataJSON,
	)

	if err != nil {
		slog.Error("failed to write audit log", "error", err, "event_type", event.Type)
		// Don't fail the request, just log the error
	}

	// Also publish to broker for real-time streaming
	if a.broker != nil {
		a.broker.Publish(event.CatalogEvent)
	}

	return nil
}

// nullIfEmpty returns nil if the string is empty, otherwise the string.
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// nullIfZero returns nil if the int is zero, otherwise the int.
func nullIfZero(i int) interface{} {
	if i == 0 {
		return nil
	}
	return i
}
