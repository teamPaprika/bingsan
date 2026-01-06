---
title: "Metrics"
weight: 4
---

# Performance Metrics

Bingsan exposes performance metrics via Prometheus at the `/metrics` endpoint. This page documents all performance-related metrics for monitoring and alerting.

## Pool Metrics

Object pool utilization metrics help monitor memory efficiency.

### bingsan_pool_gets_total

**Type**: Counter
**Labels**: `pool`

Total number of `Get()` operations on the pool.

```promql
# Get rate per pool
rate(bingsan_pool_gets_total[5m])

# Total gets by pool type
sum by (pool) (bingsan_pool_gets_total)
```

### bingsan_pool_returns_total

**Type**: Counter
**Labels**: `pool`

Total number of successful `Put()` operations returning items to the pool.

```promql
# Return rate
rate(bingsan_pool_returns_total[5m])

# Pool efficiency (returns/gets)
rate(bingsan_pool_returns_total{pool="buffer"}[5m])
/ rate(bingsan_pool_gets_total{pool="buffer"}[5m])
```

### bingsan_pool_discards_total

**Type**: Counter
**Labels**: `pool`

Total number of discarded items (oversized or invalid).

```promql
# Discard rate (should be low)
rate(bingsan_pool_discards_total[5m])

# Discard percentage
rate(bingsan_pool_discards_total{pool="buffer"}[5m])
/ rate(bingsan_pool_gets_total{pool="buffer"}[5m]) * 100
```

### bingsan_pool_misses_total

**Type**: Counter
**Labels**: `pool`

Total number of pool misses requiring new allocations.

```promql
# Miss rate
rate(bingsan_pool_misses_total[5m])

# Hit rate (estimated)
1 - (rate(bingsan_pool_misses_total[5m]) / rate(bingsan_pool_gets_total[5m]))
```

---

## Pool Labels

| Label | Values | Description |
|-------|--------|-------------|
| `pool` | `buffer`, `bytes` | Pool type identifier |

---

## Grafana Dashboard

### Pool Utilization Panel

```json
{
  "title": "Pool Utilization",
  "type": "stat",
  "targets": [
    {
      "expr": "100 * rate(bingsan_pool_returns_total{pool=\"buffer\"}[5m]) / rate(bingsan_pool_gets_total{pool=\"buffer\"}[5m])",
      "legendFormat": "Buffer Pool"
    }
  ],
  "options": {
    "reduceOptions": {
      "values": false,
      "calcs": ["lastNotNull"]
    }
  },
  "fieldConfig": {
    "defaults": {
      "unit": "percent",
      "thresholds": {
        "mode": "absolute",
        "steps": [
          {"color": "red", "value": 0},
          {"color": "yellow", "value": 70},
          {"color": "green", "value": 90}
        ]
      }
    }
  }
}
```

### Pool Operations Rate

```json
{
  "title": "Pool Operations",
  "type": "graph",
  "targets": [
    {
      "expr": "rate(bingsan_pool_gets_total[5m])",
      "legendFormat": "Gets - {{pool}}"
    },
    {
      "expr": "rate(bingsan_pool_returns_total[5m])",
      "legendFormat": "Returns - {{pool}}"
    },
    {
      "expr": "rate(bingsan_pool_discards_total[5m])",
      "legendFormat": "Discards - {{pool}}"
    }
  ]
}
```

### Pool Health Dashboard

Complete dashboard JSON:

```json
{
  "title": "Bingsan Pool Health",
  "panels": [
    {
      "title": "Buffer Pool Utilization",
      "gridPos": {"x": 0, "y": 0, "w": 8, "h": 6},
      "type": "gauge",
      "targets": [{
        "expr": "100 * rate(bingsan_pool_returns_total{pool=\"buffer\"}[5m]) / rate(bingsan_pool_gets_total{pool=\"buffer\"}[5m])"
      }],
      "fieldConfig": {
        "defaults": {
          "unit": "percent",
          "min": 0,
          "max": 100
        }
      }
    },
    {
      "title": "Discard Rate",
      "gridPos": {"x": 8, "y": 0, "w": 8, "h": 6},
      "type": "stat",
      "targets": [{
        "expr": "rate(bingsan_pool_discards_total{pool=\"buffer\"}[5m])"
      }],
      "fieldConfig": {
        "defaults": {
          "unit": "ops",
          "thresholds": {
            "steps": [
              {"color": "green", "value": 0},
              {"color": "yellow", "value": 5},
              {"color": "red", "value": 20}
            ]
          }
        }
      }
    },
    {
      "title": "Pool Operations Over Time",
      "gridPos": {"x": 0, "y": 6, "w": 24, "h": 10},
      "type": "timeseries",
      "targets": [
        {"expr": "rate(bingsan_pool_gets_total[5m])", "legendFormat": "Gets {{pool}}"},
        {"expr": "rate(bingsan_pool_returns_total[5m])", "legendFormat": "Returns {{pool}}"},
        {"expr": "rate(bingsan_pool_discards_total[5m])", "legendFormat": "Discards {{pool}}"}
      ]
    }
  ]
}
```

---

## Alerting Rules

### Low Pool Hit Rate

```yaml
groups:
  - name: bingsan_pool
    rules:
      - alert: LowPoolHitRate
        expr: |
          (rate(bingsan_pool_returns_total{pool="buffer"}[5m])
           / rate(bingsan_pool_gets_total{pool="buffer"}[5m])) < 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Pool hit rate below 80%"
          description: |
            Buffer pool hit rate is {{ $value | humanizePercentage }}.
            Consider checking for buffer leaks or memory pressure.
```

### High Discard Rate

```yaml
      - alert: HighPoolDiscardRate
        expr: rate(bingsan_pool_discards_total{pool="buffer"}[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High pool discard rate"
          description: |
            Discarding {{ $value | printf "%.1f" }} buffers/sec.
            Large responses may be impacting pool efficiency.
            Consider increasing MaxBufferSize.
```

### Pool Miss Spike

```yaml
      - alert: PoolMissSpike
        expr: |
          rate(bingsan_pool_misses_total[5m])
          / rate(bingsan_pool_gets_total[5m]) > 0.3
        for: 2m
        labels:
          severity: info
        annotations:
          summary: "Pool miss rate elevated"
          description: |
            {{ $value | humanizePercentage }} of pool gets are misses.
            This may indicate increased load or pool pressure.
```

---

## Recording Rules

Pre-compute expensive queries:

```yaml
groups:
  - name: bingsan_pool_recording
    interval: 30s
    rules:
      - record: bingsan:pool_hit_rate:5m
        expr: |
          rate(bingsan_pool_returns_total[5m])
          / rate(bingsan_pool_gets_total[5m])

      - record: bingsan:pool_discard_rate:5m
        expr: rate(bingsan_pool_discards_total[5m])

      - record: bingsan:pool_miss_rate:5m
        expr: |
          rate(bingsan_pool_misses_total[5m])
          / rate(bingsan_pool_gets_total[5m])
```

Use in alerts:

```yaml
- alert: LowPoolHitRate
  expr: bingsan:pool_hit_rate:5m{pool="buffer"} < 0.8
```

---

## Interpreting Metrics

### Healthy Pool

```
gets_total:    1,000,000
returns_total: 1,000,000
discards_total:        50
misses_total:         100

Utilization: 100% (returns = gets)
Discard rate: 0.005%
Miss rate: 0.01%
```

### Pool with Leaks

```
gets_total:    1,000,000
returns_total:   800,000  ← 200,000 missing!
discards_total:       100
misses_total:     200,100  ← High misses

Utilization: 80%
Miss rate: 20%
```

**Action**: Check for missing `defer pool.Put()` calls.

### Pool with Large Responses

```
gets_total:    1,000,000
returns_total:   700,000
discards_total:  300,000  ← High discards!
misses_total:         50

Utilization: 70%
Discard rate: 30%
```

**Action**: Consider increasing `MaxBufferSize` if schemas are large.

---

## Health Check Endpoints

### /health

Basic health check (returns 200 if healthy):

```bash
curl http://localhost:8181/health
```

### /metrics

Prometheus metrics endpoint:

```bash
curl http://localhost:8181/metrics | grep bingsan_pool
```

Example output:

```
# HELP bingsan_pool_gets_total Total number of pool Get() calls
# TYPE bingsan_pool_gets_total counter
bingsan_pool_gets_total{pool="buffer"} 15234
bingsan_pool_gets_total{pool="bytes"} 8421

# HELP bingsan_pool_returns_total Total number of pool Put() calls
# TYPE bingsan_pool_returns_total counter
bingsan_pool_returns_total{pool="buffer"} 15234
bingsan_pool_returns_total{pool="bytes"} 8421

# HELP bingsan_pool_discards_total Total number of discarded pool items
# TYPE bingsan_pool_discards_total counter
bingsan_pool_discards_total{pool="buffer"} 12
bingsan_pool_discards_total{pool="bytes"} 0

# HELP bingsan_pool_misses_total Total number of pool misses
# TYPE bingsan_pool_misses_total counter
bingsan_pool_misses_total{pool="buffer"} 5
bingsan_pool_misses_total{pool="bytes"} 3
```
