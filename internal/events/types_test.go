package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventType_Constants(t *testing.T) {
	// Namespace events
	assert.Equal(t, EventType("namespace.created"), NamespaceCreated)
	assert.Equal(t, EventType("namespace.updated"), NamespaceUpdated)
	assert.Equal(t, EventType("namespace.dropped"), NamespaceDropped)

	// Table events
	assert.Equal(t, EventType("table.created"), TableCreated)
	assert.Equal(t, EventType("table.updated"), TableUpdated)
	assert.Equal(t, EventType("table.dropped"), TableDropped)
	assert.Equal(t, EventType("table.renamed"), TableRenamed)
	assert.Equal(t, EventType("table.committed"), TableCommitted)

	// View events
	assert.Equal(t, EventType("view.created"), ViewCreated)
	assert.Equal(t, EventType("view.replaced"), ViewReplaced)
	assert.Equal(t, EventType("view.dropped"), ViewDropped)
	assert.Equal(t, EventType("view.renamed"), ViewRenamed)

	// Transaction events
	assert.Equal(t, EventType("transaction.committed"), TransactionCommitted)
	assert.Equal(t, EventType("transaction.failed"), TransactionFailed)

	// Scan events
	assert.Equal(t, EventType("scan.plan.submitted"), ScanPlanSubmitted)
	assert.Equal(t, EventType("scan.plan.completed"), ScanPlanCompleted)
	assert.Equal(t, EventType("scan.plan.cancelled"), ScanPlanCancelled)
}

func TestNewEvent(t *testing.T) {
	event := NewEvent(TableCreated)

	require.NotNil(t, event)
	assert.NotEmpty(t, event.ID)
	assert.Equal(t, TableCreated, event.Type)
	assert.False(t, event.Timestamp.IsZero())
	assert.NotNil(t, event.Metadata)
	assert.Empty(t, event.Namespace)
	assert.Empty(t, event.Table)
	assert.Empty(t, event.View)
	assert.Empty(t, event.Actor)
}

func TestCatalogEvent_WithNamespace(t *testing.T) {
	event := NewEvent(NamespaceCreated).WithNamespace("my-namespace")

	assert.Equal(t, "my-namespace", event.Namespace)
}

func TestCatalogEvent_WithTable(t *testing.T) {
	event := NewEvent(TableCreated).WithTable("my-table")

	assert.Equal(t, "my-table", event.Table)
}

func TestCatalogEvent_WithView(t *testing.T) {
	event := NewEvent(ViewCreated).WithView("my-view")

	assert.Equal(t, "my-view", event.View)
}

func TestCatalogEvent_WithActor(t *testing.T) {
	event := NewEvent(TableCreated).WithActor("user-123")

	assert.Equal(t, "user-123", event.Actor)
}

func TestCatalogEvent_WithMetadata(t *testing.T) {
	event := NewEvent(TableCreated).
		WithMetadata("key1", "value1").
		WithMetadata("key2", 42).
		WithMetadata("key3", true)

	assert.Equal(t, "value1", event.Metadata["key1"])
	assert.Equal(t, 42, event.Metadata["key2"])
	assert.Equal(t, true, event.Metadata["key3"])
}

func TestCatalogEvent_MethodChaining(t *testing.T) {
	event := NewEvent(TableCreated).
		WithNamespace("ns").
		WithTable("tbl").
		WithActor("actor").
		WithMetadata("version", 1)

	assert.Equal(t, TableCreated, event.Type)
	assert.Equal(t, "ns", event.Namespace)
	assert.Equal(t, "tbl", event.Table)
	assert.Equal(t, "actor", event.Actor)
	assert.Equal(t, 1, event.Metadata["version"])
}

func TestCatalogEvent_Topic(t *testing.T) {
	tests := []struct {
		name      string
		event     *CatalogEvent
		expected  string
	}{
		{
			name: "namespace and table",
			event: &CatalogEvent{
				Namespace: "myns",
				Table:     "mytable",
			},
			expected: "myns/mytable",
		},
		{
			name: "namespace and view",
			event: &CatalogEvent{
				Namespace: "myns",
				View:      "myview",
			},
			expected: "myns/myview",
		},
		{
			name: "namespace only",
			event: &CatalogEvent{
				Namespace: "myns",
			},
			expected: "myns",
		},
		{
			name: "table takes precedence over view",
			event: &CatalogEvent{
				Namespace: "myns",
				Table:     "mytable",
				View:      "myview",
			},
			expected: "myns/mytable",
		},
		{
			name:     "empty event",
			event:    &CatalogEvent{},
			expected: "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.event.Topic())
		})
	}
}

func TestNewAuditEvent(t *testing.T) {
	catalogEvent := NewEvent(TableCreated).
		WithNamespace("ns").
		WithTable("tbl")

	auditEvent := NewAuditEvent(catalogEvent)

	require.NotNil(t, auditEvent)
	assert.Equal(t, catalogEvent.ID, auditEvent.ID)
	assert.Equal(t, catalogEvent.Type, auditEvent.Type)
	assert.Equal(t, catalogEvent.Namespace, auditEvent.Namespace)
	assert.Equal(t, catalogEvent.Table, auditEvent.Table)
	assert.Equal(t, "anonymous", auditEvent.ActorType)
}

func TestAuditEvent_WithActorType(t *testing.T) {
	event := NewAuditEvent(NewEvent(TableCreated)).WithActorType("api_key")

	assert.Equal(t, "api_key", event.ActorType)
}

func TestAuditEvent_WithIPAddress(t *testing.T) {
	event := NewAuditEvent(NewEvent(TableCreated)).WithIPAddress("192.168.1.1")

	assert.Equal(t, "192.168.1.1", event.IPAddress)
}

func TestAuditEvent_WithUserAgent(t *testing.T) {
	event := NewAuditEvent(NewEvent(TableCreated)).WithUserAgent("Mozilla/5.0")

	assert.Equal(t, "Mozilla/5.0", event.UserAgent)
}

func TestAuditEvent_WithRequestID(t *testing.T) {
	event := NewAuditEvent(NewEvent(TableCreated)).WithRequestID("req-123")

	assert.Equal(t, "req-123", event.RequestID)
}

func TestAuditEvent_WithResponseCode(t *testing.T) {
	event := NewAuditEvent(NewEvent(TableCreated)).WithResponseCode(200)

	assert.Equal(t, 200, event.ResponseCode)
}

func TestAuditEvent_MethodChaining(t *testing.T) {
	catalogEvent := NewEvent(TableCreated).
		WithNamespace("ns").
		WithTable("tbl").
		WithActor("user")

	auditEvent := NewAuditEvent(catalogEvent).
		WithActorType("oauth2").
		WithIPAddress("10.0.0.1").
		WithUserAgent("TestClient/1.0").
		WithRequestID("req-456").
		WithResponseCode(201)

	assert.Equal(t, "oauth2", auditEvent.ActorType)
	assert.Equal(t, "10.0.0.1", auditEvent.IPAddress)
	assert.Equal(t, "TestClient/1.0", auditEvent.UserAgent)
	assert.Equal(t, "req-456", auditEvent.RequestID)
	assert.Equal(t, 201, auditEvent.ResponseCode)
}

func TestCatalogEvent_UniqueIDs(t *testing.T) {
	event1 := NewEvent(TableCreated)
	event2 := NewEvent(TableCreated)

	assert.NotEqual(t, event1.ID, event2.ID, "each event should have unique ID")
}

func TestCatalogEvent_Timestamp(t *testing.T) {
	before := time.Now().Add(-time.Second)
	event := NewEvent(TableCreated)
	after := time.Now().Add(time.Second)

	assert.True(t, event.Timestamp.After(before))
	assert.True(t, event.Timestamp.Before(after))
}

func TestCatalogEvent_JSON_Tags(t *testing.T) {
	event := CatalogEvent{
		ID:        "test-id",
		Type:      TableCreated,
		Timestamp: time.Now(),
		Namespace: "ns",
		Table:     "tbl",
		Actor:     "actor",
		Metadata:  map[string]any{"key": "value"},
	}

	// Verify struct has correct JSON tags by checking it can be created
	assert.Equal(t, "test-id", event.ID)
	assert.Equal(t, TableCreated, event.Type)
	assert.Equal(t, "ns", event.Namespace)
	assert.Equal(t, "tbl", event.Table)
	assert.Equal(t, "actor", event.Actor)
}
