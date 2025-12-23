---
title: "Monitoring"
weight: 6
---

# Monitoring Configuration

Configure observability features including logging, metrics, and event streaming.

## Logging

Bingsan uses Go's structured logging (`slog`) with JSON output by default.

### Log Levels

Control log verbosity via the `debug` flag:

```yaml
server:
  debug: false  # INFO level
  # or
  debug: true   # DEBUG level
```

### Log Output

Logs are written to stdout in JSON format:

```json
{
  "time": "2024-01-15T10:30:00.000Z",
  "level": "INFO",
  "msg": "server listening",
  "address": "0.0.0.0:8181"
}
```

### Docker/Kubernetes

Logs are automatically collected from stdout:

```bash
# Docker
docker logs bingsan

# Kubernetes
kubectl logs deployment/bingsan
```

---

## Prometheus Metrics

Metrics are exposed at `/metrics` in Prometheus format.

### Endpoint

```bash
curl http://localhost:8181/metrics
```

### Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'bingsan'
    static_configs:
      - targets: ['bingsan:8181']
    metrics_path: /metrics
    scrape_interval: 15s
```

### Kubernetes ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: bingsan
spec:
  selector:
    matchLabels:
      app: bingsan
  endpoints:
    - port: http
      path: /metrics
      interval: 15s
```

### Key Metrics

| Metric | Description |
|--------|-------------|
| `iceberg_catalog_http_requests_total` | Total HTTP requests |
| `iceberg_catalog_http_request_duration_seconds` | Request latency histogram |
| `iceberg_namespaces_total` | Total namespaces |
| `iceberg_tables_total` | Tables per namespace |
| `iceberg_db_connections_open` | Open database connections |
| `iceberg_db_connections_in_use` | In-use database connections |

See [Health & Metrics API]({{< relref "/docs/api/health-metrics" >}}) for the complete list.

---

## Health Checks

### Liveness Probe

Check if the server is running:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8181
  initialDelaySeconds: 10
  periodSeconds: 10
  failureThreshold: 3
```

### Readiness Probe

Check if the server can accept requests (includes database connectivity):

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8181
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 3
```

---

## Event Streaming

Real-time catalog events are available via WebSocket.

### Endpoint

```
ws://localhost:8181/v1/events/stream
```

### Connection

```bash
# With wscat
wscat -c "ws://localhost:8181/v1/events/stream"

# With authentication
wscat -c "ws://localhost:8181/v1/events/stream?token=YOUR_TOKEN"
```

### Event Types

- `namespace_created`, `namespace_updated`, `namespace_deleted`
- `table_created`, `table_updated`, `table_dropped`, `table_renamed`
- `view_created`, `view_updated`, `view_dropped`, `view_renamed`
- `transaction_committed`

See [Events API]({{< relref "/docs/api/events" >}}) for details.

---

## Grafana Dashboards

### Request Rate

```promql
sum(rate(iceberg_catalog_http_requests_total[5m])) by (status)
```

### Request Latency (p99)

```promql
histogram_quantile(0.99,
  sum(rate(iceberg_catalog_http_request_duration_seconds_bucket[5m])) by (le)
)
```

### Error Rate

```promql
sum(rate(iceberg_catalog_http_requests_total{status=~"5.."}[5m]))
/
sum(rate(iceberg_catalog_http_requests_total[5m]))
```

### Database Pool Usage

```promql
iceberg_db_connections_in_use / iceberg_db_connections_max
```

### Catalog Size

```promql
iceberg_namespaces_total
sum(iceberg_tables_total)
```

---

## Alerting

### Prometheus Alerting Rules

```yaml
groups:
  - name: bingsan-alerts
    rules:
      - alert: BingsanDown
        expr: up{job="bingsan"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Bingsan is down"

      - alert: BingsanHighErrorRate
        expr: |
          sum(rate(iceberg_catalog_http_requests_total{status=~"5.."}[5m]))
          / sum(rate(iceberg_catalog_http_requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate in Bingsan"

      - alert: BingsanHighLatency
        expr: |
          histogram_quantile(0.99,
            sum(rate(iceberg_catalog_http_request_duration_seconds_bucket[5m])) by (le)
          ) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency in Bingsan"

      - alert: BingsanDatabasePoolExhausted
        expr: iceberg_db_connections_in_use / iceberg_db_connections_max > 0.9
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Database connection pool nearly exhausted"
```

---

## Distributed Tracing

> [!NOTE]
> OpenTelemetry tracing support is planned for a future release.

---

## Log Aggregation

### Fluentd Configuration

```xml
<source>
  @type forward
  port 24224
</source>

<filter bingsan.**>
  @type parser
  key_name log
  <parse>
    @type json
  </parse>
</filter>

<match bingsan.**>
  @type elasticsearch
  host elasticsearch
  port 9200
  logstash_format true
  logstash_prefix bingsan
</match>
```

### Docker Logging Driver

```yaml
services:
  bingsan:
    logging:
      driver: json-file
      options:
        max-size: "100m"
        max-file: "5"
```

### Kubernetes

Logs are automatically collected by most Kubernetes logging solutions (Loki, Elasticsearch, CloudWatch).

---

## Request Tracing

Each request includes a unique ID in the response headers and logs:

```bash
curl -v http://localhost:8181/v1/namespaces
# Response headers include: X-Request-ID: abc123
```

Use this ID to correlate requests across logs:

```json
{"level":"INFO","msg":"request completed","request_id":"abc123","status":200}
```

---

## Best Practices

1. **Set up basic monitoring first**: Health checks, error rate, latency
2. **Alert on symptoms, not causes**: High error rate, not specific errors
3. **Use dashboards for investigation**: Drill down from alerts to detailed metrics
4. **Aggregate logs centrally**: Easier debugging and audit trails
5. **Monitor database connections**: Often the bottleneck in catalog operations
