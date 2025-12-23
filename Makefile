.PHONY: all build test lint clean run dev docker-build docker-up docker-down migrate help

# Variables
BINARY_NAME=iceberg-catalog
BUILD_DIR=./bin
MAIN_PATH=./cmd/iceberg-catalog
GO=go
GOFLAGS=-ldflags="-s -w"
DOCKER_COMPOSE=docker compose

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Default target
all: lint test build

## help: Show this help message
help:
	@echo "Iceberg REST Catalog - Build Targets"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

## build-linux: Build for Linux (useful for Docker)
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

## run: Run the application locally
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

## dev: Run with hot reload (requires air)
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

## test: Run all tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...

## test-short: Run short tests only
test-short:
	@echo "Running short tests..."
	$(GO) test -v -short ./...

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GO) test -v -tags=integration ./tests/integration/...

## coverage: Show test coverage
coverage: test
	@echo "Coverage report:"
	$(GO) tool cover -func=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "HTML report: coverage.html"

## lint: Run linters
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@echo "Running linters..."
	golangci-lint run ./...

## lint-fix: Run linters and fix issues
lint-fix:
	@echo "Running linters with auto-fix..."
	golangci-lint run --fix ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	gofumpt -l -w .

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

## tidy: Tidy go modules
tidy:
	@echo "Tidying modules..."
	$(GO) mod tidy

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t iceberg-catalog:$(VERSION) -f deployments/docker/Dockerfile .

## docker-up: Start services with Docker Compose
docker-up:
	@echo "Starting services..."
	$(DOCKER_COMPOSE) -f deployments/docker/docker-compose.yml up -d

## docker-down: Stop services
docker-down:
	@echo "Stopping services..."
	$(DOCKER_COMPOSE) -f deployments/docker/docker-compose.yml down

## docker-logs: Show logs
docker-logs:
	$(DOCKER_COMPOSE) -f deployments/docker/docker-compose.yml logs -f

## migrate-up: Run database migrations
migrate-up:
	@echo "Running migrations..."
	migrate -path internal/db/migrations -database "postgres://iceberg:iceberg@localhost:5432/iceberg_catalog?sslmode=disable" up

## migrate-down: Rollback database migrations
migrate-down:
	@echo "Rolling back migrations..."
	migrate -path internal/db/migrations -database "postgres://iceberg:iceberg@localhost:5432/iceberg_catalog?sslmode=disable" down 1

## migrate-create: Create new migration (usage: make migrate-create name=migration_name)
migrate-create:
	@echo "Creating migration: $(name)"
	migrate create -ext sql -dir internal/db/migrations -seq $(name)

## sqlc: Generate SQL code
sqlc:
	@which sqlc > /dev/null || (echo "Installing sqlc..." && go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest)
	@echo "Generating SQL code..."
	sqlc generate

## proto: Generate protobuf code (if using protobuf)
proto:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/*.proto

## openapi: Validate OpenAPI spec
openapi:
	@which swagger > /dev/null || (echo "Installing swagger..." && go install github.com/go-swagger/go-swagger/cmd/swagger@latest)
	swagger validate api/openapi.yaml

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

## security: Run security checks
security:
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	@echo "Running security checks..."
	gosec -quiet ./...

## check: Run all checks (lint, test, vet)
check: lint vet test

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install mvdan.cc/gofumpt@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Tools installed!"
