package events

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of catalog event.
type EventType string

const (
	// Namespace events
	NamespaceCreated EventType = "namespace.created"
	NamespaceUpdated EventType = "namespace.updated"
	NamespaceDropped EventType = "namespace.dropped"

	// Table events
	TableCreated   EventType = "table.created"
	TableUpdated   EventType = "table.updated"
	TableDropped   EventType = "table.dropped"
	TableRenamed   EventType = "table.renamed"
	TableCommitted EventType = "table.committed"

	// View events
	ViewCreated  EventType = "view.created"
	ViewReplaced EventType = "view.replaced"
	ViewDropped  EventType = "view.dropped"
	ViewRenamed  EventType = "view.renamed"

	// Transaction events
	TransactionCommitted EventType = "transaction.committed"
	TransactionFailed    EventType = "transaction.failed"

	// Scan events
	ScanPlanSubmitted EventType = "scan.plan.submitted"
	ScanPlanCompleted EventType = "scan.plan.completed"
	ScanPlanCancelled EventType = "scan.plan.cancelled"
)

// CatalogEvent represents a catalog change event.
type CatalogEvent struct {
	ID        string         `json:"id"`
	Type      EventType      `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Namespace string         `json:"namespace,omitempty"`
	Table     string         `json:"table,omitempty"`
	View      string         `json:"view,omitempty"`
	Actor     string         `json:"actor,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// NewEvent creates a new catalog event.
func NewEvent(eventType EventType) *CatalogEvent {
	return &CatalogEvent{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}
}

// WithNamespace sets the namespace for the event.
func (e *CatalogEvent) WithNamespace(namespace string) *CatalogEvent {
	e.Namespace = namespace
	return e
}

// WithTable sets the table for the event.
func (e *CatalogEvent) WithTable(table string) *CatalogEvent {
	e.Table = table
	return e
}

// WithView sets the view for the event.
func (e *CatalogEvent) WithView(view string) *CatalogEvent {
	e.View = view
	return e
}

// WithActor sets the actor for the event.
func (e *CatalogEvent) WithActor(actor string) *CatalogEvent {
	e.Actor = actor
	return e
}

// WithMetadata adds metadata to the event.
func (e *CatalogEvent) WithMetadata(key string, value any) *CatalogEvent {
	e.Metadata[key] = value
	return e
}

// Topic returns the topic string for this event.
// Format: namespace or namespace/table or namespace/view
func (e *CatalogEvent) Topic() string {
	if e.Table != "" {
		return e.Namespace + "/" + e.Table
	}
	if e.View != "" {
		return e.Namespace + "/" + e.View
	}
	if e.Namespace != "" {
		return e.Namespace
	}
	return "*"
}

// AuditEvent extends CatalogEvent with additional audit information.
type AuditEvent struct {
	CatalogEvent
	ActorType    string `json:"actor_type,omitempty"`    // "api_key", "oauth2", "anonymous"
	IPAddress    string `json:"ip_address,omitempty"`
	UserAgent    string `json:"user_agent,omitempty"`
	RequestID    string `json:"request_id,omitempty"`
	ResponseCode int    `json:"response_code,omitempty"`
}

// NewAuditEvent creates a new audit event from a catalog event.
func NewAuditEvent(event *CatalogEvent) *AuditEvent {
	return &AuditEvent{
		CatalogEvent: *event,
		ActorType:    "anonymous",
	}
}

// WithActorType sets the actor type for the audit event.
func (e *AuditEvent) WithActorType(actorType string) *AuditEvent {
	e.ActorType = actorType
	return e
}

// WithIPAddress sets the IP address for the audit event.
func (e *AuditEvent) WithIPAddress(ip string) *AuditEvent {
	e.IPAddress = ip
	return e
}

// WithUserAgent sets the user agent for the audit event.
func (e *AuditEvent) WithUserAgent(ua string) *AuditEvent {
	e.UserAgent = ua
	return e
}

// WithRequestID sets the request ID for the audit event.
func (e *AuditEvent) WithRequestID(reqID string) *AuditEvent {
	e.RequestID = reqID
	return e
}

// WithResponseCode sets the response code for the audit event.
func (e *AuditEvent) WithResponseCode(code int) *AuditEvent {
	e.ResponseCode = code
	return e
}
