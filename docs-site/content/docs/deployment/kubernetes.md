---
title: "Kubernetes"
weight: 2
---

# Kubernetes Deployment

Deploy Bingsan on Kubernetes for production workloads.

## Prerequisites

- Kubernetes 1.24+
- kubectl configured
- PostgreSQL database (managed or self-hosted)
- Object storage (S3/GCS)

## Quick Start

### 1. Create Namespace

```bash
kubectl create namespace bingsan
```

### 2. Create Secrets

```bash
# Database credentials
kubectl create secret generic bingsan-db \
  --namespace bingsan \
  --from-literal=host=postgres.example.com \
  --from-literal=port=5432 \
  --from-literal=user=iceberg \
  --from-literal=password=your-password \
  --from-literal=database=iceberg_catalog

# S3 credentials (if using static credentials)
kubectl create secret generic bingsan-s3 \
  --namespace bingsan \
  --from-literal=access-key-id=AKIA... \
  --from-literal=secret-access-key=...
```

### 3. Create ConfigMap

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: bingsan-config
  namespace: bingsan
data:
  config.yaml: |
    server:
      host: 0.0.0.0
      port: 8181
      debug: false

    storage:
      type: s3
      warehouse: s3://your-bucket/warehouse
      s3:
        region: us-east-1
        bucket: your-bucket

    auth:
      enabled: true
      token_expiry: 1h

    catalog:
      lock_timeout: 30s
EOF
```

### 4. Deploy

```bash
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bingsan
  namespace: bingsan
spec:
  replicas: 3
  selector:
    matchLabels:
      app: bingsan
  template:
    metadata:
      labels:
        app: bingsan
    spec:
      containers:
      - name: bingsan
        image: ghcr.io/kimuyb/bingsan:latest
        ports:
        - containerPort: 8181
          name: http
        env:
        - name: ICEBERG_DATABASE_HOST
          valueFrom:
            secretKeyRef:
              name: bingsan-db
              key: host
        - name: ICEBERG_DATABASE_PORT
          valueFrom:
            secretKeyRef:
              name: bingsan-db
              key: port
        - name: ICEBERG_DATABASE_USER
          valueFrom:
            secretKeyRef:
              name: bingsan-db
              key: user
        - name: ICEBERG_DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: bingsan-db
              key: password
        - name: ICEBERG_DATABASE_DATABASE
          valueFrom:
            secretKeyRef:
              name: bingsan-db
              key: database
        - name: ICEBERG_STORAGE_S3_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: bingsan-s3
              key: access-key-id
        - name: ICEBERG_STORAGE_S3_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: bingsan-s3
              key: secret-access-key
        - name: ICEBERG_AUTH_SIGNING_KEY
          valueFrom:
            secretKeyRef:
              name: bingsan-auth
              key: signing-key
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: bingsan-config
---
apiVersion: v1
kind: Service
metadata:
  name: bingsan
  namespace: bingsan
spec:
  selector:
    app: bingsan
  ports:
  - port: 8181
    targetPort: http
    name: http
EOF
```

---

## Complete Manifests

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bingsan
  namespace: bingsan
  labels:
    app: bingsan
spec:
  replicas: 3
  selector:
    matchLabels:
      app: bingsan
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  template:
    metadata:
      labels:
        app: bingsan
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8181"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: bingsan
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
      - name: bingsan
        image: ghcr.io/kimuyb/bingsan:v1.0.0
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8181
          name: http
          protocol: TCP
        env:
        - name: ICEBERG_DATABASE_HOST
          valueFrom:
            secretKeyRef:
              name: bingsan-db
              key: host
        - name: ICEBERG_DATABASE_PORT
          valueFrom:
            secretKeyRef:
              name: bingsan-db
              key: port
        - name: ICEBERG_DATABASE_USER
          valueFrom:
            secretKeyRef:
              name: bingsan-db
              key: user
        - name: ICEBERG_DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: bingsan-db
              key: password
        - name: ICEBERG_DATABASE_DATABASE
          valueFrom:
            secretKeyRef:
              name: bingsan-db
              key: database
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
          readOnly: true
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: http
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: http
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
      volumes:
      - name: config
        configMap:
          name: bingsan-config
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: bingsan
              topologyKey: kubernetes.io/hostname
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: bingsan
  namespace: bingsan
  labels:
    app: bingsan
spec:
  type: ClusterIP
  selector:
    app: bingsan
  ports:
  - port: 8181
    targetPort: http
    protocol: TCP
    name: http
```

### Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: bingsan
  namespace: bingsan
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "50m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "60"
spec:
  ingressClassName: nginx
  rules:
  - host: catalog.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: bingsan
            port:
              number: 8181
  tls:
  - hosts:
    - catalog.example.com
    secretName: bingsan-tls
```

### ServiceAccount and RBAC

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bingsan
  namespace: bingsan
---
# If using AWS IRSA or GCP Workload Identity,
# add annotations to the ServiceAccount
```

### HorizontalPodAutoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: bingsan
  namespace: bingsan
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: bingsan
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
      - type: Percent
        value: 100
        periodSeconds: 15
```

### PodDisruptionBudget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: bingsan
  namespace: bingsan
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: bingsan
```

---

## AWS IAM Roles for Service Accounts (IRSA)

For AWS S3 access without static credentials:

### 1. Create IAM Role

```bash
eksctl create iamserviceaccount \
  --name bingsan \
  --namespace bingsan \
  --cluster your-cluster \
  --attach-policy-arn arn:aws:iam::ACCOUNT:policy/BingsanS3Access \
  --approve
```

### 2. IAM Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::your-bucket",
        "arn:aws:s3:::your-bucket/*"
      ]
    }
  ]
}
```

### 3. Update Deployment

```yaml
spec:
  serviceAccountName: bingsan
  containers:
  - name: bingsan
    env:
    # Remove S3 credentials - IRSA handles auth
    - name: ICEBERG_STORAGE_S3_REGION
      value: us-east-1
```

---

## GCP Workload Identity

For GCS access:

### 1. Create Service Account Binding

```bash
gcloud iam service-accounts add-iam-policy-binding \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:PROJECT.svc.id.goog[bingsan/bingsan]" \
  bingsan-sa@PROJECT.iam.gserviceaccount.com
```

### 2. Annotate Kubernetes ServiceAccount

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: bingsan
  namespace: bingsan
  annotations:
    iam.gke.io/gcp-service-account: bingsan-sa@PROJECT.iam.gserviceaccount.com
```

---

## Monitoring

### ServiceMonitor (Prometheus Operator)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: bingsan
  namespace: bingsan
spec:
  selector:
    matchLabels:
      app: bingsan
  endpoints:
  - port: http
    path: /metrics
    interval: 15s
```

### PrometheusRule

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: bingsan
  namespace: bingsan
spec:
  groups:
  - name: bingsan
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
```

---

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n bingsan
kubectl describe pod -n bingsan bingsan-xxx
```

### View Logs

```bash
kubectl logs -n bingsan -l app=bingsan --tail=100 -f
```

### Test Connectivity

```bash
# Port forward for local testing
kubectl port-forward -n bingsan svc/bingsan 8181:8181

# Test health
curl http://localhost:8181/health
```

### Debug Container

```bash
kubectl exec -it -n bingsan deployment/bingsan -- /bin/sh
```
