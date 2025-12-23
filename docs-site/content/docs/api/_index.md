---
title: "API Reference"
weight: 2
bookCollapseSection: true
---

# API Reference

Complete REST API documentation for Bingsan, compliant with the Apache Iceberg REST Catalog specification.

## Base URL

```
http://localhost:8181/v1
```

## Authentication

When authentication is enabled, include the bearer token in requests:

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8181/v1/namespaces
```

See [Authentication Configuration]({{< relref "/docs/configuration/auth" >}}) for details.

## Content Type

All requests with a body should use:

```
Content-Type: application/json
```

## API Sections

### Core Operations

- [Configuration]({{< relref "/docs/api/configuration" >}}) - Catalog configuration endpoint
- [Namespaces]({{< relref "/docs/api/namespaces" >}}) - Namespace CRUD operations
- [Tables]({{< relref "/docs/api/tables" >}}) - Table management and commits
- [Views]({{< relref "/docs/api/views" >}}) - View management

### Advanced Operations

- [Scan Planning]({{< relref "/docs/api/scan-planning" >}}) - Server-side scan planning
- [Transactions]({{< relref "/docs/api/transactions" >}}) - Multi-table atomic commits
- [Events]({{< relref "/docs/api/events" >}}) - Real-time WebSocket event streaming

### Operational

- [Health & Metrics]({{< relref "/docs/api/health-metrics" >}}) - Health checks and Prometheus metrics
- [OAuth]({{< relref "/docs/api/oauth" >}}) - Token exchange endpoints

## Error Responses

All error responses follow this format:

```json
{
  "error": {
    "message": "Error description",
    "type": "ErrorType",
    "code": 400
  }
}
```

### Common Error Types

| Code | Type | Description |
|------|------|-------------|
| 400 | `BadRequestException` | Invalid request parameters |
| 401 | `UnauthorizedException` | Missing or invalid authentication |
| 403 | `ForbiddenException` | Permission denied |
| 404 | `NoSuchNamespaceException` | Namespace not found |
| 404 | `NoSuchTableException` | Table not found |
| 404 | `NoSuchViewException` | View not found |
| 409 | `AlreadyExistsException` | Resource already exists |
| 409 | `CommitFailedException` | Optimistic locking failure |
| 500 | `ServerError` | Internal server error |

## Catalog Prefix

Bingsan supports prefixed API paths for multi-catalog deployments:

```
/v1/{prefix}/namespaces
/v1/{prefix}/namespaces/{namespace}/tables
```

Example:
```bash
curl http://localhost:8181/v1/my-catalog/namespaces
```
