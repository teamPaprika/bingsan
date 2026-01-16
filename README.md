<p align="center">
  <img src="assets/bingsan-logo.png" alt="Bingsan" width="150">
</p>

<h1 align="center">Bingsan</h1>

<p align="center">
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/teamPaprika/bingsan"><img src="https://goreportcard.com/badge/github.com/teamPaprika/bingsan" alt="Go Report Card"></a>
  <a href="https://teampaprika.github.io/bingsan/"><img src="https://img.shields.io/badge/docs-latest-brightgreen.svg" alt="Documentation"></a>
</p>

<p align="center">
  <strong>High-performance Apache Iceberg REST Catalog in Go — 2-3x faster than alternatives</strong>
</p>

[한국어](#한국어) | [English](#english)

---

## Performance

Bingsan significantly outperforms other REST Catalog implementations. Benchmark results using [Apache Polaris Tools](https://github.com/apache/polaris-tools) with 10 concurrent connections:

| Operation | Bingsan (Go) | Lakekeeper (Rust) | Speedup |
|-----------|--------------|-------------------|---------|
| Health Check | 11,933 req/s | 4,253 req/s | **2.8x** |
| GET Namespace | 6,991 req/s | 2,481 req/s | **2.8x** |
| LIST Namespaces | 9,018 req/s | 3,073 req/s | **2.9x** |
| CREATE Namespace | 457 req/s | 378 req/s | **1.2x** |

<details>
<summary>Run benchmarks yourself</summary>

```bash
# Setup benchmark framework
make bench-setup

# Start Bingsan with OAuth2
make bench-start

# Run benchmarks
make bench-run

# View results
make bench-report
```

</details>

---

## 한국어

### 소개

**Bingsan**은 Go로 구현된 고성능 Apache Iceberg REST Catalog입니다. PostgreSQL을 메타데이터 저장소로 사용하며, Go의 동시성 모델을 활용하여 높은 처리량과 낮은 지연시간을 제공합니다.

### 주요 특징

- **완전한 Iceberg REST API 구현**: Apache Iceberg REST Catalog OpenAPI 스펙 준수
- **PostgreSQL 백엔드**: ACID 트랜잭션을 보장하는 안정적인 메타데이터 저장소
- **Go 동시성**: 고루틴 기반의 효율적인 요청 처리
- **실시간 이벤트 스트리밍**: WebSocket을 통한 카탈로그 변경 이벤트 수신
- **Prometheus 메트릭**: 운영 모니터링을 위한 메트릭 엔드포인트
- **Kubernetes 친화적**: 단일 노드 또는 멀티 노드 클러스터 배포 지원

### 아키텍처

```
┌─────────────────────────────────────────────────────────────┐
│                      Clients                                │
│         (Spark, Trino, Flink, PyIceberg, etc.)             │
└─────────────────────────┬───────────────────────────────────┘
                          │ REST API
┌─────────────────────────▼───────────────────────────────────┐
│                    Bingsan Cluster                          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐                     │
│  │ Node 1  │  │ Node 2  │  │ Node N  │  (Kubernetes Pods)  │
│  │  :8181  │  │  :8181  │  │  :8181  │                     │
│  └────┬────┘  └────┬────┘  └────┬────┘                     │
│       └────────────┼────────────┘                           │
└────────────────────┼────────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
┌───────▼───────┐       ┌────────▼────────┐
│  PostgreSQL   │       │   S3 / GCS      │
│  (Metadata)   │       │   (Data Lake)   │
└───────────────┘       └─────────────────┘
```

### 빠른 시작

#### 요구사항

- Go 1.25+
- PostgreSQL 15+
- Docker & Docker Compose (개발용)

#### 설치 및 실행

```bash
# 저장소 클론
git clone https://github.com/teamPaprika/bingsan.git
cd bingsan

# 설정 파일 복사
cp config.example.yaml config.yaml

# Docker Compose로 실행
docker compose -f deployments/docker/docker-compose.yml up -d

# 테스트 실행
docker compose -f deployments/docker/test.yaml up --abort-on-container-exit
```

#### 로컬 개발

```bash
# 의존성 설치
go mod download

# 빌드
go build -o bin/iceberg-catalog ./cmd/iceberg-catalog

# 실행
./bin/iceberg-catalog
```

### API 엔드포인트

#### 네임스페이스 관리
| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/v1/namespaces` | 네임스페이스 목록 조회 |
| POST | `/v1/namespaces` | 네임스페이스 생성 |
| GET | `/v1/namespaces/{namespace}` | 네임스페이스 조회 |
| DELETE | `/v1/namespaces/{namespace}` | 네임스페이스 삭제 |
| HEAD | `/v1/namespaces/{namespace}` | 네임스페이스 존재 확인 |
| POST | `/v1/namespaces/{namespace}/properties` | 속성 업데이트 |

#### 테이블 관리
| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/v1/namespaces/{namespace}/tables` | 테이블 목록 조회 |
| POST | `/v1/namespaces/{namespace}/tables` | 테이블 생성/커밋 |
| GET | `/v1/namespaces/{namespace}/tables/{table}` | 테이블 메타데이터 조회 |
| DELETE | `/v1/namespaces/{namespace}/tables/{table}` | 테이블 삭제 |
| HEAD | `/v1/namespaces/{namespace}/tables/{table}` | 테이블 존재 확인 |
| POST | `/v1/namespaces/{namespace}/register` | 기존 테이블 등록 |
| POST | `/v1/tables/rename` | 테이블 이름 변경 |

#### 뷰 관리
| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/v1/namespaces/{namespace}/views` | 뷰 목록 조회 |
| POST | `/v1/namespaces/{namespace}/views` | 뷰 생성 |
| GET | `/v1/namespaces/{namespace}/views/{view}` | 뷰 메타데이터 조회 |
| DELETE | `/v1/namespaces/{namespace}/views/{view}` | 뷰 삭제 |
| POST | `/v1/views/rename` | 뷰 이름 변경 |

#### 스캔 계획
| 메서드 | 경로 | 설명 |
|--------|------|------|
| POST | `/v1/namespaces/{namespace}/tables/{table}/plan` | 스캔 계획 제출 |
| GET | `/v1/namespaces/{namespace}/tables/{table}/plan/{plan-id}` | 스캔 계획 상태 조회 |
| DELETE | `/v1/namespaces/{namespace}/tables/{table}/plan/{plan-id}` | 스캔 계획 취소 |
| POST | `/v1/namespaces/{namespace}/tables/{table}/tasks` | 스캔 태스크 조회 |

#### 모니터링
| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/metrics` | Prometheus 메트릭 |
| GET | `/health/live` | Liveness 체크 |
| GET | `/health/ready` | Readiness 체크 |
| WebSocket | `/v1/events/stream` | 실시간 이벤트 스트림 |

### 설정

`config.yaml` 예시:

```yaml
server:
  host: "0.0.0.0"
  port: 8181
  read_timeout: 30s
  write_timeout: 30s

database:
  host: "localhost"
  port: 5432
  user: "iceberg"
  password: "iceberg"
  database: "iceberg_catalog"
  max_open_conns: 25
  max_idle_conns: 5

storage:
  type: "s3"
  warehouse: "s3://my-bucket/warehouse"
  region: "us-east-1"

auth:
  enabled: false
```

### 호환성

다음 시스템과 호환됩니다:

- Apache Spark (Iceberg Spark runtime)
- Trino / Presto
- Apache Flink
- PyIceberg
- Dremio
- StarRocks

### 라이선스

Apache License 2.0

---

## English

### Introduction

**Bingsan** is a high-performance Apache Iceberg REST Catalog implemented in Go. It uses PostgreSQL as the metadata store and leverages Go's concurrency model to provide high throughput and low latency.

### Key Features

- **Complete Iceberg REST API**: Compliant with Apache Iceberg REST Catalog OpenAPI spec
- **PostgreSQL Backend**: Reliable metadata storage with ACID transactions
- **Go Concurrency**: Efficient request handling using goroutines
- **Real-time Event Streaming**: Receive catalog change events via WebSocket
- **Prometheus Metrics**: Metrics endpoint for operational monitoring
- **Kubernetes-Friendly**: Supports single-node or multi-node cluster deployments

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Clients                                │
│         (Spark, Trino, Flink, PyIceberg, etc.)             │
└─────────────────────────┬───────────────────────────────────┘
                          │ REST API
┌─────────────────────────▼───────────────────────────────────┐
│                    Bingsan Cluster                          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐                     │
│  │ Node 1  │  │ Node 2  │  │ Node N  │  (Kubernetes Pods)  │
│  │  :8181  │  │  :8181  │  │  :8181  │                     │
│  └────┬────┘  └────┬────┘  └────┬────┘                     │
│       └────────────┼────────────┘                           │
└────────────────────┼────────────────────────────────────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
┌───────▼───────┐       ┌────────▼────────┐
│  PostgreSQL   │       │   S3 / GCS      │
│  (Metadata)   │       │   (Data Lake)   │
└───────────────┘       └─────────────────┘
```

### Quick Start

#### Requirements

- Go 1.25+
- PostgreSQL 15+
- Docker & Docker Compose (for development)

#### Installation & Running

```bash
# Clone the repository
git clone https://github.com/teamPaprika/bingsan.git
cd bingsan

# Copy configuration file
cp config.example.yaml config.yaml

# Run with Docker Compose
docker compose -f deployments/docker/docker-compose.yml up -d

# Run tests
docker compose -f deployments/docker/test.yaml up --abort-on-container-exit
```

#### Local Development

```bash
# Install dependencies
go mod download

# Build
go build -o bin/iceberg-catalog ./cmd/iceberg-catalog

# Run
./bin/iceberg-catalog
```

### API Endpoints

#### Namespace Management
| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/namespaces` | List namespaces |
| POST | `/v1/namespaces` | Create namespace |
| GET | `/v1/namespaces/{namespace}` | Get namespace |
| DELETE | `/v1/namespaces/{namespace}` | Delete namespace |
| HEAD | `/v1/namespaces/{namespace}` | Check namespace exists |
| POST | `/v1/namespaces/{namespace}/properties` | Update properties |

#### Table Management
| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/namespaces/{namespace}/tables` | List tables |
| POST | `/v1/namespaces/{namespace}/tables` | Create/commit table |
| GET | `/v1/namespaces/{namespace}/tables/{table}` | Get table metadata |
| DELETE | `/v1/namespaces/{namespace}/tables/{table}` | Drop table |
| HEAD | `/v1/namespaces/{namespace}/tables/{table}` | Check table exists |
| POST | `/v1/namespaces/{namespace}/register` | Register existing table |
| POST | `/v1/tables/rename` | Rename table |

#### View Management
| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/namespaces/{namespace}/views` | List views |
| POST | `/v1/namespaces/{namespace}/views` | Create view |
| GET | `/v1/namespaces/{namespace}/views/{view}` | Get view metadata |
| DELETE | `/v1/namespaces/{namespace}/views/{view}` | Drop view |
| POST | `/v1/views/rename` | Rename view |

#### Scan Planning
| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/namespaces/{namespace}/tables/{table}/plan` | Submit scan plan |
| GET | `/v1/namespaces/{namespace}/tables/{table}/plan/{plan-id}` | Get scan plan status |
| DELETE | `/v1/namespaces/{namespace}/tables/{table}/plan/{plan-id}` | Cancel scan plan |
| POST | `/v1/namespaces/{namespace}/tables/{table}/tasks` | Fetch scan tasks |

#### Monitoring
| Method | Path | Description |
|--------|------|-------------|
| GET | `/metrics` | Prometheus metrics |
| GET | `/health/live` | Liveness check |
| GET | `/health/ready` | Readiness check |
| WebSocket | `/v1/events/stream` | Real-time event stream |

### Configuration

Example `config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8181
  read_timeout: 30s
  write_timeout: 30s

database:
  host: "localhost"
  port: 5432
  user: "iceberg"
  password: "iceberg"
  database: "iceberg_catalog"
  max_open_conns: 25
  max_idle_conns: 5

storage:
  type: "s3"
  warehouse: "s3://my-bucket/warehouse"
  region: "us-east-1"

auth:
  enabled: false
```

### Compatibility

Compatible with:

- Apache Spark (Iceberg Spark runtime)
- Trino / Presto
- Apache Flink
- PyIceberg
- Dremio
- StarRocks

### License

Apache License 2.0
