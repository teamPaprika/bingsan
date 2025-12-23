---
title: "Namespaces"
weight: 2
---

# Namespaces API

Namespaces are containers for tables and views, similar to databases or schemas in traditional systems.

## List Namespaces

List all namespaces, optionally filtered by parent.

### Request

```http
GET /v1/namespaces
GET /v1/namespaces?parent={parent}
GET /v1/namespaces?pageToken={token}&pageSize={size}
```

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `parent` | string | No | Parent namespace to filter children (URL-encoded, e.g., `analytics%00events`) |
| `pageToken` | string | No | Pagination token from previous response |
| `pageSize` | integer | No | Maximum number of results (default: 100) |

### Response

```json
{
  "namespaces": [
    ["analytics"],
    ["analytics", "events"],
    ["raw"]
  ],
  "next-page-token": "eyJvZmZzZXQiOjEwMH0="
}
```

### Example

```bash
# List all namespaces
curl http://localhost:8181/v1/namespaces

# List children of "analytics"
curl "http://localhost:8181/v1/namespaces?parent=analytics"
```

---

## Create Namespace

Create a new namespace.

### Request

```http
POST /v1/namespaces
```

### Request Body

```json
{
  "namespace": ["analytics", "events"],
  "properties": {
    "owner": "data-team",
    "description": "Event analytics data"
  }
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | array[string] | Yes | Namespace identifier as array of name parts |
| `properties` | object | No | Key-value properties for the namespace |

### Response

```json
{
  "namespace": ["analytics", "events"],
  "properties": {
    "owner": "data-team",
    "description": "Event analytics data"
  }
}
```

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 400 | `BadRequestException` | Invalid namespace format |
| 409 | `AlreadyExistsException` | Namespace already exists |

### Example

```bash
curl -X POST http://localhost:8181/v1/namespaces \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": ["analytics"],
    "properties": {
      "owner": "platform-team"
    }
  }'
```

---

## Get Namespace

Load a namespace's metadata including properties.

### Request

```http
GET /v1/namespaces/{namespace}
```

### Path Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `namespace` | string | URL-encoded namespace (use `%1F` for nested, e.g., `analytics%1Fevents`) |

### Response

```json
{
  "namespace": ["analytics", "events"],
  "properties": {
    "owner": "data-team",
    "description": "Event analytics data",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 404 | `NoSuchNamespaceException` | Namespace does not exist |

### Example

```bash
# Simple namespace
curl http://localhost:8181/v1/namespaces/analytics

# Nested namespace (URL-encode the separator)
curl http://localhost:8181/v1/namespaces/analytics%1Fevents
```

---

## Check Namespace Exists

Check if a namespace exists without loading its metadata.

### Request

```http
HEAD /v1/namespaces/{namespace}
```

### Response

- **200 OK**: Namespace exists
- **404 Not Found**: Namespace does not exist

### Example

```bash
curl -I http://localhost:8181/v1/namespaces/analytics
```

---

## Update Namespace Properties

Update or remove properties from a namespace.

### Request

```http
POST /v1/namespaces/{namespace}/properties
```

### Request Body

```json
{
  "updates": {
    "owner": "new-team",
    "retention": "90d"
  },
  "removals": ["description", "deprecated_key"]
}
```

### Request Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `updates` | object | No | Properties to add or update |
| `removals` | array[string] | No | Property keys to remove |

### Response

```json
{
  "updated": ["owner", "retention"],
  "removed": ["description"],
  "missing": ["deprecated_key"]
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `updated` | array[string] | Properties that were updated |
| `removed` | array[string] | Properties that were removed |
| `missing` | array[string] | Properties in `removals` that didn't exist |

### Example

```bash
curl -X POST http://localhost:8181/v1/namespaces/analytics/properties \
  -H "Content-Type: application/json" \
  -d '{
    "updates": {"owner": "platform-team"},
    "removals": ["old_property"]
  }'
```

---

## Delete Namespace

Delete an empty namespace.

### Request

```http
DELETE /v1/namespaces/{namespace}
```

### Response

- **204 No Content**: Namespace deleted successfully

### Errors

| Code | Error | Description |
|------|-------|-------------|
| 404 | `NoSuchNamespaceException` | Namespace does not exist |
| 409 | `NamespaceNotEmptyException` | Namespace contains tables or views |

### Example

```bash
curl -X DELETE http://localhost:8181/v1/namespaces/analytics
```

> [!WARNING]
> You must delete all tables and views in a namespace before deleting the namespace itself.
