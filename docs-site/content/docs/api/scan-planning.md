---
title: "Scan Planning"
weight: 5
---

# Scan Planning API

Server-side scan planning allows the catalog to optimize query planning by filtering manifests and data files based on predicates.

## Overview

Scan planning follows an asynchronous workflow:

1. **Submit** a scan plan request
2. **Poll** for plan completion
3. **Fetch** scan tasks containing file references
4. Optionally **cancel** an in-progress plan

```
┌─────────┐    Submit Plan    ┌─────────┐    Poll Status    ┌─────────┐
│ Client  │ ──────────────────▶│ Server  │◀─────────────────▶│ Client  │
└─────────┘                    └─────────┘                   └─────────┘
                                    │
                                    │ Planning Complete
                                    ▼
┌─────────┐    Fetch Tasks    ┌─────────┐
│ Client  │◀──────────────────│ Server  │
└─────────┘                   └─────────┘
```

---

## Submit Scan Plan

Submit a new scan plan request.

### Request

```http
POST /v1/namespaces/{namespace}/tables/{table}/plan
```

### Request Body

```json
{
  "snapshot-id": 123456789,
  "select": ["event_id", "user_id", "event_time"],
  "filter": {
    "type": "and",
    "left": {
      "type": "gte",
      "term": "event_time",
      "value": "2024-01-01T00:00:00Z"
    },
    "right": {
      "type": "lt",
      "term": "event_time",
      "value": "2024-02-01T00:00:00Z"
    }
  },
  "case-sensitive": true,
  "use-snapshot-schema": false,
  "start-snapshot-id": null,
  "end-snapshot-id": null
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `snapshot-id` | long | No | Snapshot to scan (latest if omitted) |
| `select` | array[string] | No | Columns to project |
| `filter` | object | No | Filter expression |
| `case-sensitive` | boolean | No | Case-sensitive column matching (default: true) |
| `use-snapshot-schema` | boolean | No | Use snapshot's schema vs current |
| `start-snapshot-id` | long | No | For incremental scans |
| `end-snapshot-id` | long | No | For incremental scans |

### Filter Expression Types

#### Comparison Operations

```json
{"type": "eq", "term": "column", "value": "value"}
{"type": "neq", "term": "column", "value": "value"}
{"type": "lt", "term": "column", "value": 100}
{"type": "lte", "term": "column", "value": 100}
{"type": "gt", "term": "column", "value": 100}
{"type": "gte", "term": "column", "value": 100}
```

#### Logical Operations

```json
{"type": "and", "left": {...}, "right": {...}}
{"type": "or", "left": {...}, "right": {...}}
{"type": "not", "child": {...}}
```

#### Set Operations

```json
{"type": "in", "term": "column", "values": [1, 2, 3]}
{"type": "not-in", "term": "column", "values": [1, 2, 3]}
```

#### Null Checks

```json
{"type": "is-null", "term": "column"}
{"type": "not-null", "term": "column"}
```

#### String Operations

```json
{"type": "starts-with", "term": "column", "value": "prefix"}
{"type": "not-starts-with", "term": "column", "value": "prefix"}
```

### Response

```json
{
  "plan-id": "plan-550e8400-e29b-41d4-a716-446655440000",
  "status": "submitted"
}
```

### Example

```bash
curl -X POST http://localhost:8181/v1/namespaces/analytics/tables/events/plan \
  -H "Content-Type: application/json" \
  -d '{
    "select": ["event_id", "user_id"],
    "filter": {"type": "eq", "term": "event_type", "value": "click"}
  }'
```

---

## Get Scan Plan Status

Poll for scan plan completion.

### Request

```http
GET /v1/namespaces/{namespace}/tables/{table}/plan/{plan-id}
```

### Response (In Progress)

```json
{
  "plan-id": "plan-550e8400-e29b-41d4-a716-446655440000",
  "status": "planning",
  "progress": {
    "manifests-scanned": 50,
    "manifests-total": 100,
    "data-files-matched": 1250
  }
}
```

### Response (Complete)

```json
{
  "plan-id": "plan-550e8400-e29b-41d4-a716-446655440000",
  "status": "complete",
  "statistics": {
    "manifests-scanned": 100,
    "manifests-skipped": 45,
    "data-files-matched": 2500,
    "data-files-skipped": 15000,
    "total-file-size-bytes": 10737418240,
    "planning-duration-ms": 1250
  }
}
```

### Status Values

| Status | Description |
|--------|-------------|
| `submitted` | Plan request received |
| `planning` | Scanning manifests and files |
| `complete` | Planning finished, ready to fetch tasks |
| `failed` | Planning encountered an error |
| `cancelled` | Plan was cancelled |

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 404 | `NoSuchPlanException` | Plan ID not found |

### Example

```bash
curl http://localhost:8181/v1/namespaces/analytics/tables/events/plan/plan-550e8400
```

---

## Fetch Plan Tasks

Retrieve scan tasks for a completed plan.

### Request

```http
POST /v1/namespaces/{namespace}/tables/{table}/tasks
```

### Request Body

```json
{
  "plan-id": "plan-550e8400-e29b-41d4-a716-446655440000",
  "page-token": null,
  "page-size": 100
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `plan-id` | string | Yes | Plan ID from submit response |
| `page-token` | string | No | Pagination token |
| `page-size` | integer | No | Tasks per page (default: 100, max: 1000) |

### Response

```json
{
  "tasks": [
    {
      "task-id": "task-001",
      "data-files": [
        {
          "content": "data",
          "file-path": "s3://bucket/data/part-00001.parquet",
          "file-format": "parquet",
          "file-size-in-bytes": 52428800,
          "record-count": 100000,
          "partition": {"event_day": "2024-01-15"},
          "key-metadata": null,
          "split-offsets": [0, 26214400],
          "sort-order-id": 0
        }
      ],
      "delete-files": [
        {
          "content": "position-deletes",
          "file-path": "s3://bucket/data/delete-00001.parquet",
          "file-format": "parquet",
          "file-size-in-bytes": 1048576,
          "record-count": 500
        }
      ],
      "residual-filter": {"type": "eq", "term": "event_type", "value": "click"}
    }
  ],
  "next-page-token": "eyJvZmZzZXQiOjEwMH0="
}
```

### Task Fields

| Field | Type | Description |
|-------|------|-------------|
| `task-id` | string | Unique task identifier |
| `data-files` | array | Data files to scan |
| `delete-files` | array | Associated delete files |
| `residual-filter` | object | Filter to apply during scan |

### File Entry Fields

| Field | Type | Description |
|-------|------|-------------|
| `content` | string | `data`, `position-deletes`, or `equality-deletes` |
| `file-path` | string | Storage path |
| `file-format` | string | `parquet`, `avro`, `orc` |
| `file-size-in-bytes` | long | File size |
| `record-count` | long | Number of records |
| `partition` | object | Partition values |
| `split-offsets` | array[long] | Offsets for parallel reads |

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | `PlanNotCompleteException` | Plan still in progress |
| 404 | `NoSuchPlanException` | Plan ID not found |

### Example

```bash
curl -X POST http://localhost:8181/v1/namespaces/analytics/tables/events/tasks \
  -H "Content-Type: application/json" \
  -d '{"plan-id": "plan-550e8400-e29b-41d4-a716-446655440000"}'
```

---

## Cancel Scan Plan

Cancel an in-progress scan plan.

### Request

```http
DELETE /v1/namespaces/{namespace}/tables/{table}/plan/{plan-id}
```

### Response

- **204 No Content**: Plan cancelled

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 404 | `NoSuchPlanException` | Plan ID not found |

### Example

```bash
curl -X DELETE http://localhost:8181/v1/namespaces/analytics/tables/events/plan/plan-550e8400
```

---

## Best Practices

### Use Projections

Always specify `select` to reduce data transfer:

```json
{
  "select": ["user_id", "event_time"]
}
```

### Push Down Filters

Provide filters to skip irrelevant partitions:

```json
{
  "filter": {
    "type": "and",
    "left": {"type": "gte", "term": "event_date", "value": "2024-01-01"},
    "right": {"type": "lt", "term": "event_date", "value": "2024-01-08"}
  }
}
```

### Handle Pagination

Large scans may return many tasks:

```python
page_token = None
while True:
    response = fetch_tasks(plan_id, page_token)
    process_tasks(response['tasks'])
    page_token = response.get('next-page-token')
    if not page_token:
        break
```

### Poll Efficiently

Use exponential backoff when polling:

```python
import time

wait = 0.1  # Start with 100ms
max_wait = 5.0

while True:
    status = get_plan_status(plan_id)
    if status['status'] == 'complete':
        break
    time.sleep(wait)
    wait = min(wait * 2, max_wait)
```
