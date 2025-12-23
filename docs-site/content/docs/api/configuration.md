---
title: "Configuration Endpoint"
weight: 1
---

# Configuration Endpoint

Retrieve catalog configuration and defaults.

## Get Configuration

Returns the catalog's configuration properties, including storage defaults and feature flags.

### Request

```http
GET /v1/config
```

### Response

```json
{
  "defaults": {
    "warehouse": "s3://my-bucket/warehouse"
  },
  "overrides": {}
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `defaults` | object | Default configuration properties |
| `defaults.warehouse` | string | Default warehouse location |
| `overrides` | object | Properties that override client settings |

### Example

```bash
curl http://localhost:8181/v1/config
```

### Usage

Clients typically call this endpoint during initialization to discover:

- Default warehouse location for new tables
- Required configuration overrides
- Catalog capabilities

### Notes

- This endpoint does not require authentication (even when auth is enabled)
- The response varies based on server configuration
- Clients should merge `defaults` with their local config, then apply `overrides`
