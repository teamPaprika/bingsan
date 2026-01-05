# Bingsan Polaris Benchmarks Makefile
# Uses apache/polaris-tools Gatling framework

SHELL := /bin/bash
BENCHMARKS_DIR := $(shell pwd)
POLARIS_TOOLS_DIR := $(BENCHMARKS_DIR)/polaris-tools
GATLING_DIR := $(POLARIS_TOOLS_DIR)/benchmarks
REPORTS_DIR := $(GATLING_DIR)/build/reports/gatling
CONFIG_FILE := $(BENCHMARKS_DIR)/config/bingsan.conf

# Use Java 22 if available (Java 25+ has Gradle compatibility issues)
JAVA_HOME_22 := $(shell /usr/libexec/java_home -v 22 2>/dev/null || echo "")
JAVA_HOME_17 := $(shell /usr/libexec/java_home -v 17 2>/dev/null || echo "")
ifneq ($(JAVA_HOME_22),)
    export JAVA_HOME := $(JAVA_HOME_22)
else ifneq ($(JAVA_HOME_17),)
    export JAVA_HOME := $(JAVA_HOME_17)
endif

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m

.PHONY: help setup start-bingsan stop-bingsan wait-healthy \
        create-dataset read-benchmark read-update-benchmark \
        create-commits-benchmark weighted-benchmark full-benchmark \
        report clean clean-data

help: ## Show this help
	@echo "Bingsan Polaris Benchmark Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-22s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "Typical workflow:"
	@echo "  1. make setup           # First-time setup"
	@echo "  2. make start-bingsan   # Start the catalog"
	@echo "  3. make create-dataset  # Populate test data"
	@echo "  4. make read-benchmark  # Run benchmarks"
	@echo "  5. make report          # View results"
	@echo "  6. make stop-bingsan    # Cleanup"

# =============================================================================
# Setup
# =============================================================================

setup: ## Clone polaris-tools and build Gatling simulations
	@echo "$(GREEN)Setting up benchmark environment...$(NC)"
	@./scripts/setup.sh

check-setup: ## Verify polaris-tools is set up
	@if [ ! -d "$(POLARIS_TOOLS_DIR)" ]; then \
		echo "$(RED)Error: polaris-tools not found. Run 'make setup' first.$(NC)"; \
		exit 1; \
	fi

# =============================================================================
# Bingsan Lifecycle
# =============================================================================

start-bingsan: ## Start bingsan with OAuth2 enabled for benchmarking
	@echo "$(GREEN)Starting bingsan benchmark stack...$(NC)"
	@docker compose -f docker-compose.benchmark.yaml up -d --build
	@$(MAKE) wait-healthy

stop-bingsan: ## Stop bingsan benchmark stack
	@echo "$(YELLOW)Stopping bingsan benchmark stack...$(NC)"
	@docker compose -f docker-compose.benchmark.yaml down

wait-healthy: ## Wait for bingsan to be healthy
	@echo "Waiting for bingsan to be healthy..."
	@for i in $$(seq 1 60); do \
		if curl -sf http://localhost:8181/health > /dev/null 2>&1; then \
			echo "$(GREEN)Bingsan is ready!$(NC)"; \
			exit 0; \
		fi; \
		echo "  Waiting... ($$i/60)"; \
		sleep 2; \
	done; \
	echo "$(RED)Timeout waiting for bingsan$(NC)"; \
	exit 1

logs: ## Show bingsan logs
	@docker compose -f docker-compose.benchmark.yaml logs -f catalog

# =============================================================================
# Benchmark Simulations
# =============================================================================

create-dataset: check-setup ## Create test dataset (run first before other benchmarks)
	@echo "$(GREEN)Creating benchmark dataset...$(NC)"
	@cd $(GATLING_DIR) && \
		./gradlew gatlingRun --simulation org.apache.polaris.benchmarks.simulations.CreateTreeDataset \
			-Dconfig.file=$(CONFIG_FILE)

read-benchmark: check-setup ## Run read-only benchmark
	@echo "$(GREEN)Running read benchmark...$(NC)"
	@cd $(GATLING_DIR) && \
		./gradlew gatlingRun --simulation org.apache.polaris.benchmarks.simulations.ReadTreeDataset \
			-Dconfig.file=$(CONFIG_FILE)

read-update-benchmark: check-setup ## Run mixed read/update benchmark
	@echo "$(GREEN)Running read-update benchmark...$(NC)"
	@cd $(GATLING_DIR) && \
		./gradlew gatlingRun --simulation org.apache.polaris.benchmarks.simulations.ReadUpdateTreeDataset \
			-Dconfig.file=$(CONFIG_FILE)

create-commits-benchmark: check-setup ## Run commit throughput benchmark
	@echo "$(GREEN)Running create-commits benchmark...$(NC)"
	@cd $(GATLING_DIR) && \
		./gradlew gatlingRun --simulation org.apache.polaris.benchmarks.simulations.CreateCommits \
			-Dconfig.file=$(CONFIG_FILE)

weighted-benchmark: check-setup ## Run weighted workload benchmark
	@echo "$(GREEN)Running weighted workload benchmark...$(NC)"
	@cd $(GATLING_DIR) && \
		./gradlew gatlingRun --simulation org.apache.polaris.benchmarks.simulations.WeightedWorkloadOnTreeDataset \
			-Dconfig.file=$(CONFIG_FILE)

full-benchmark: check-setup ## Run all benchmarks sequentially
	@echo "$(GREEN)Running full benchmark suite...$(NC)"
	@$(MAKE) create-dataset
	@$(MAKE) read-benchmark
	@$(MAKE) read-update-benchmark
	@$(MAKE) create-commits-benchmark
	@echo "$(GREEN)Full benchmark suite complete!$(NC)"

# =============================================================================
# Results
# =============================================================================

report: ## Open the latest benchmark report in browser
	@echo "$(GREEN)Opening benchmark reports...$(NC)"
	@if [ -d "$(REPORTS_DIR)" ]; then \
		LATEST=$$(ls -td $(REPORTS_DIR)/*/ 2>/dev/null | head -1); \
		if [ -n "$$LATEST" ]; then \
			echo "Opening: $$LATEST/index.html"; \
			open "$$LATEST/index.html" 2>/dev/null || \
			xdg-open "$$LATEST/index.html" 2>/dev/null || \
			echo "Report at: $$LATEST/index.html"; \
		else \
			echo "$(YELLOW)No reports found. Run a benchmark first.$(NC)"; \
		fi; \
	else \
		echo "$(YELLOW)Reports directory not found. Run 'make setup' and a benchmark first.$(NC)"; \
	fi

list-reports: ## List all benchmark reports
	@echo "Available benchmark reports:"
	@if [ -d "$(REPORTS_DIR)" ]; then \
		ls -lt $(REPORTS_DIR) | tail -n +2 | head -20; \
	else \
		echo "  No reports found."; \
	fi

# =============================================================================
# Cleanup
# =============================================================================

clean: ## Remove polaris-tools and reports
	@echo "$(YELLOW)Cleaning benchmark environment...$(NC)"
	@rm -rf $(POLARIS_TOOLS_DIR)
	@echo "Done."

clean-data: ## Remove benchmark docker volumes (keeps polaris-tools)
	@echo "$(YELLOW)Removing benchmark data volumes...$(NC)"
	@docker compose -f docker-compose.benchmark.yaml down -v
	@echo "Done."

# =============================================================================
# Quick Test
# =============================================================================

quick-test: ## Quick connectivity test (no Gatling)
	@echo "Testing bingsan connectivity..."
	@echo "  GET /health:"
	@curl -s http://localhost:8181/health | head -c 200
	@echo ""
	@echo "  GET /v1/config:"
	@curl -s http://localhost:8181/v1/config | head -c 500
	@echo ""
	@echo ""
	@echo "  POST /v1/oauth/tokens (get bearer token):"
	@curl -s -X POST http://localhost:8181/v1/oauth/tokens \
		-H "Content-Type: application/x-www-form-urlencoded" \
		-d "grant_type=client_credentials&client_id=benchmark-client&client_secret=benchmark-secret" | head -c 500
	@echo ""
