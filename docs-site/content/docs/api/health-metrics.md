---
title: "Health & Metrics"
weight: 8
---

# Health & Metrics API

Bingsan provides health check and Prometheus metrics endpoints for operational monitoring.

## Health Check

Simple health check endpoint for load balancers.

### Request

```http
GET /health
```

### Response

```json
{"status": "ok"}
```

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Server is healthy |
| 503 | Server is unhealthy |

### Example

```bash
curl http://localhost:8181/health
```

---

## Readiness Check

Readiness check including database connectivity.

### Request

```http
GET /ready
```

### Response (Ready)

```json
{
  "status": "ready",
  "database": "connected"
}
```

### Response (Not Ready)

```json
{
  "status": "not_ready",
  "database": "disconnected",
  "error": "connection refused"
}
```

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Server is ready to accept requests |
| 503 | Server is not ready (database unavailable) |

### Example

```bash
curl http://localhost:8181/ready
```

### Kubernetes Usage

Configure liveness and readiness probes:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8181
  initialDelaySeconds: 10
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8181
  initialDelaySeconds: 5
  periodSeconds: 5
```

---

## Prometheus Metrics

Expose metrics in Prometheus format.

### Request

```http
GET /metrics
```

### Response

```
# HELP iceberg_catalog_http_requests_total Total number of HTTP requests
# TYPE iceberg_catalog_http_requests_total counter
iceberg_catalog_http_requests_total{method="GET",path="/v1/namespaces",status="200"} 1234

# HELP iceberg_namespaces_total Total number of namespaces
# TYPE iceberg_namespaces_total gauge
iceberg_namespaces_total 15

# HELP iceberg_tables_total Total number of tables per namespace
# TYPE iceberg_tables_total gauge
iceberg_tables_total{namespace="analytics"} 42
iceberg_tables_total{namespace="raw"} 18

# HELP iceberg_db_connections_open Current open database connections
# TYPE iceberg_db_connections_open gauge
iceberg_db_connections_open 5

# HELP iceberg_db_connections_in_use Current in-use database connections
# TYPE iceberg_db_connections_in_use gauge
iceberg_db_connections_in_use 2
```

### Example

```bash
curl http://localhost:8181/metrics
```

---

## Available Metrics

### HTTP Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `iceberg_catalog_http_requests_total` | counter | method, path, status | Total HTTP requests |
| `iceberg_catalog_http_request_duration_seconds` | histogram | method, path | Request duration |
| `iceberg_catalog_http_requests_in_flight` | gauge | - | Current in-flight requests |
| `iceberg_catalog_http_request_size_bytes` | histogram | method, path | Request body size |
| `iceberg_catalog_http_response_size_bytes` | histogram | method, path | Response body size |

### Catalog Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `iceberg_namespaces_total` | gauge | - | Total namespaces |
| `iceberg_tables_total` | gauge | namespace | Tables per namespace |
| `iceberg_views_total` | gauge | namespace | Views per namespace |
| `iceberg_commits_total` | counter | namespace, table | Table commits |
| `iceberg_scan_plans_total` | counter | status | Scan plans by status |
| `iceberg_transactions_total` | counter | status | Multi-table transactions |

### Database Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `iceberg_db_connections_open` | gauge | - | Open connections |
| `iceberg_db_connections_in_use` | gauge | - | In-use connections |
| `iceberg_db_connections_idle` | gauge | - | Idle connections |
| `iceberg_db_connections_max` | gauge | - | Max connections |
| `iceberg_db_wait_count_total` | counter | - | Connection wait count |
| `iceberg_db_wait_duration_seconds_total` | counter | - | Connection wait time |

### Event Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `iceberg_events_published_total` | counter | type | Events published |
| `iceberg_websocket_connections` | gauge | - | Active WebSocket connections |

---

## Prometheus Configuration

Add Bingsan to your Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'bingsan'
    static_configs:
      - targets: ['localhost:8181']
    metrics_path: /metrics
    scrape_interval: 15s
```

### Service Discovery (Kubernetes)

```yaml
scrape_configs:
  - job_name: 'bingsan'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: bingsan
      - source_labels: [__meta_kubernetes_pod_container_port_number]
        action: keep
        regex: "8181"
```

---

## Grafana Dashboard

### Key Panels

#### Request Rate

```promql
sum(rate(iceberg_catalog_http_requests_total[5m])) by (method, status)
```

#### Request Latency (p99)

```promql
histogram_quantile(0.99, sum(rate(iceberg_catalog_http_request_duration_seconds_bucket[5m])) by (le, path))
```

#### Error Rate

```promql
sum(rate(iceberg_catalog_http_requests_total{status=~"5.."}[5m]))
/
sum(rate(iceberg_catalog_http_requests_total[5m]))
```

#### Database Connection Pool

```promql
iceberg_db_connections_in_use / iceberg_db_connections_max
```

#### Namespaces and Tables

```promql
iceberg_namespaces_total
sum(iceberg_tables_total)
```

### Example Dashboard JSON

Import this into Grafana:

```json
{
  "dashboard": {
    "title": "Bingsan Overview",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "sum(rate(iceberg_catalog_http_requests_total[5m])) by (status)",
            "legendFormat": "{{status}}"
          }
        ]
      },
      {
        "title": "Latency p99",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, sum(rate(iceberg_catalog_http_request_duration_seconds_bucket[5m])) by (le))",
            "legendFormat": "p99"
          }
        ]
      }
    ]
  }
}
```

---

## Alerting Rules

Example Prometheus alerting rules:

```yaml
groups:
  - name: bingsan
    rules:
      - alert: BingsanHighErrorRate
        expr: |
          sum(rate(iceberg_catalog_http_requests_total{status=~"5.."}[5m]))
          / sum(rate(iceberg_catalog_http_requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate in Bingsan"
          description: "Error rate is {{ $value | humanizePercentage }}"

      - alert: BingsanDatabaseConnectionPoolExhausted
        expr: iceberg_db_connections_in_use / iceberg_db_connections_max > 0.9
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Database connection pool nearly exhausted"

      - alert: BingsanHighLatency
        expr: |
          histogram_quantile(0.99, sum(rate(iceberg_catalog_http_request_duration_seconds_bucket[5m])) by (le)) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High request latency in Bingsan"
          description: "p99 latency is {{ $value }}s"
```
