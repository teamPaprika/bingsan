---
title: "Events"
weight: 7
---

# Events API

Bingsan provides real-time event streaming via WebSocket, allowing clients to receive notifications about catalog changes as they happen.

## Event Stream

Connect to the event stream to receive real-time catalog events.

### Endpoint

```
WebSocket: ws://localhost:8181/v1/events/stream
```

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `token` | string | Yes* | Authentication token (required when auth is enabled) |
| `namespace` | string | No | Filter events to a specific namespace |

### Connection Example

Using `wscat`:

```bash
# Connect to event stream (no auth)
wscat -c "ws://localhost:8181/v1/events/stream"

# Connect with authentication
wscat -c "ws://localhost:8181/v1/events/stream?token=YOUR_API_KEY"

# Connect with namespace filter
wscat -c "ws://localhost:8181/v1/events/stream?token=YOUR_API_KEY&namespace=analytics"
```

Using JavaScript:

```javascript
const ws = new WebSocket('ws://localhost:8181/v1/events/stream?token=YOUR_TOKEN');

ws.onopen = () => {
  console.log('Connected to event stream');
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Received event:', data);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Disconnected from event stream');
};
```

Using Python:

```python
import websocket
import json

def on_message(ws, message):
    event = json.loads(message)
    print(f"Event: {event['type']} - {event}")

def on_error(ws, error):
    print(f"Error: {error}")

def on_close(ws, close_status_code, close_msg):
    print("Connection closed")

def on_open(ws):
    print("Connected to event stream")

ws = websocket.WebSocketApp(
    "ws://localhost:8181/v1/events/stream?token=YOUR_TOKEN",
    on_open=on_open,
    on_message=on_message,
    on_error=on_error,
    on_close=on_close
)

ws.run_forever()
```

---

## Event Types

### Namespace Events

#### namespace_created

Emitted when a new namespace is created.

```json
{
  "type": "namespace_created",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "namespace": ["analytics"],
  "properties": {
    "owner": "data-team"
  }
}
```

#### namespace_updated

Emitted when namespace properties are updated.

```json
{
  "type": "namespace_updated",
  "timestamp": "2024-01-15T10:35:00.000Z",
  "namespace": ["analytics"],
  "updates": {
    "owner": "platform-team"
  },
  "removals": ["description"]
}
```

#### namespace_deleted

Emitted when a namespace is deleted.

```json
{
  "type": "namespace_deleted",
  "timestamp": "2024-01-15T10:40:00.000Z",
  "namespace": ["analytics"]
}
```

### Table Events

#### table_created

Emitted when a new table is created.

```json
{
  "type": "table_created",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "namespace": ["analytics"],
  "table": "events",
  "table_uuid": "550e8400-e29b-41d4-a716-446655440000",
  "metadata_location": "s3://bucket/metadata/00000.metadata.json"
}
```

#### table_updated

Emitted when a table is updated (commit).

```json
{
  "type": "table_updated",
  "timestamp": "2024-01-15T10:35:00.000Z",
  "namespace": ["analytics"],
  "table": "events",
  "table_uuid": "550e8400-e29b-41d4-a716-446655440000",
  "previous_metadata_location": "s3://bucket/metadata/00000.metadata.json",
  "metadata_location": "s3://bucket/metadata/00001.metadata.json",
  "updates": ["add-snapshot", "set-snapshot-ref"]
}
```

#### table_dropped

Emitted when a table is dropped.

```json
{
  "type": "table_dropped",
  "timestamp": "2024-01-15T10:40:00.000Z",
  "namespace": ["analytics"],
  "table": "events",
  "table_uuid": "550e8400-e29b-41d4-a716-446655440000",
  "purge_requested": false
}
```

#### table_renamed

Emitted when a table is renamed.

```json
{
  "type": "table_renamed",
  "timestamp": "2024-01-15T10:45:00.000Z",
  "source_namespace": ["analytics"],
  "source_table": "old_name",
  "destination_namespace": ["analytics"],
  "destination_table": "new_name",
  "table_uuid": "550e8400-e29b-41d4-a716-446655440000"
}
```

### View Events

#### view_created

```json
{
  "type": "view_created",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "namespace": ["analytics"],
  "view": "daily_summary",
  "view_uuid": "660e8400-e29b-41d4-a716-446655440001",
  "metadata_location": "s3://bucket/views/metadata/00000.metadata.json"
}
```

#### view_updated

```json
{
  "type": "view_updated",
  "timestamp": "2024-01-15T10:35:00.000Z",
  "namespace": ["analytics"],
  "view": "daily_summary",
  "view_uuid": "660e8400-e29b-41d4-a716-446655440001",
  "previous_version_id": 1,
  "version_id": 2
}
```

#### view_dropped

```json
{
  "type": "view_dropped",
  "timestamp": "2024-01-15T10:40:00.000Z",
  "namespace": ["analytics"],
  "view": "daily_summary",
  "view_uuid": "660e8400-e29b-41d4-a716-446655440001"
}
```

#### view_renamed

```json
{
  "type": "view_renamed",
  "timestamp": "2024-01-15T10:45:00.000Z",
  "source_namespace": ["analytics"],
  "source_view": "old_view",
  "destination_namespace": ["analytics"],
  "destination_view": "new_view",
  "view_uuid": "660e8400-e29b-41d4-a716-446655440001"
}
```

### Transaction Events

#### transaction_committed

```json
{
  "type": "transaction_committed",
  "timestamp": "2024-01-15T10:30:00.000Z",
  "commit_id": "txn-770e8400-e29b-41d4-a716-446655440002",
  "tables_updated": [
    {"namespace": ["analytics"], "name": "events"},
    {"namespace": ["analytics"], "name": "aggregates"}
  ]
}
```

---

## Event Schema

All events share a common structure:

```json
{
  "type": "event_type",
  "timestamp": "ISO-8601 timestamp",
  ...event-specific fields
}
```

### Common Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Event type identifier |
| `timestamp` | string | ISO-8601 timestamp when event occurred |

---

## Connection Management

### Heartbeat

The server sends periodic ping frames to keep the connection alive. Clients should respond with pong frames.

### Reconnection

Clients should implement reconnection logic with exponential backoff:

```python
import time
import websocket

def connect_with_retry():
    max_retries = 10
    base_delay = 1  # seconds

    for attempt in range(max_retries):
        try:
            ws = websocket.create_connection(
                "ws://localhost:8181/v1/events/stream"
            )
            return ws
        except Exception as e:
            delay = base_delay * (2 ** attempt)
            print(f"Connection failed, retrying in {delay}s: {e}")
            time.sleep(delay)

    raise Exception("Failed to connect after max retries")
```

### Connection Limits

| Limit | Value | Description |
|-------|-------|-------------|
| Max connections per IP | 100 | Maximum concurrent connections |
| Idle timeout | 5 minutes | Connection closed if no activity |
| Max message size | 64KB | Maximum event message size |

---

## Use Cases

### Cache Invalidation

Invalidate local caches when tables change:

```python
def on_message(ws, message):
    event = json.loads(message)
    if event['type'] in ['table_updated', 'table_dropped']:
        cache_key = f"{event['namespace']}.{event['table']}"
        cache.invalidate(cache_key)
```

### Audit Logging

Stream events to an audit log:

```python
def on_message(ws, message):
    event = json.loads(message)
    audit_log.append({
        'timestamp': event['timestamp'],
        'event_type': event['type'],
        'details': event
    })
```

### Change Data Capture

Trigger downstream processing when data changes:

```python
def on_message(ws, message):
    event = json.loads(message)
    if event['type'] == 'table_updated':
        if 'add-snapshot' in event.get('updates', []):
            trigger_etl_pipeline(event['namespace'], event['table'])
```
