#!/bin/bash
# Benchmark comparison script for Bingsan vs Lakekeeper
# Uses hey (https://github.com/rakyll/hey) for HTTP load testing

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINGSAN_URL="http://localhost:8181"
LAKEKEEPER_URL="http://localhost:8182"
LAKEKEEPER_PROJECT="00000000-0000-0000-0000-000000000000"
LAKEKEEPER_WAREHOUSE="benchmark"

REQUESTS=100
CONCURRENCY=10

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Bingsan vs Lakekeeper Benchmark${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Check if hey is installed
if ! command -v hey &> /dev/null; then
    echo -e "${YELLOW}Installing hey...${NC}"
    go install github.com/rakyll/hey@latest
fi

# Get OAuth token for Bingsan
echo -e "${YELLOW}Getting Bingsan OAuth token...${NC}"
BINGSAN_TOKEN=$(curl -s -X POST "$BINGSAN_URL/v1/oauth/tokens" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "grant_type=client_credentials&client_id=benchmark-client&client_secret=benchmark-secret" \
    | jq -r '.access_token')

if [ "$BINGSAN_TOKEN" == "null" ] || [ -z "$BINGSAN_TOKEN" ]; then
    echo -e "${RED}Failed to get Bingsan token${NC}"
    exit 1
fi
echo -e "${GREEN}Got Bingsan token${NC}"

# Setup: Create test namespace in both catalogs
echo ""
echo -e "${YELLOW}Setting up test namespaces...${NC}"

# Bingsan namespace
curl -s -X POST "$BINGSAN_URL/v1/namespaces" \
    -H "Authorization: Bearer $BINGSAN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"namespace": ["benchmark_ns"]}' > /dev/null 2>&1 || true

# Lakekeeper namespace (uses different path structure)
curl -s -X POST "$LAKEKEEPER_URL/catalog/v1/$LAKEKEEPER_PROJECT/$LAKEKEEPER_WAREHOUSE/namespaces" \
    -H "Content-Type: application/json" \
    -d '{"namespace": ["benchmark_ns"]}' > /dev/null 2>&1 || true

echo -e "${GREEN}Setup complete${NC}"

# Function to run benchmark and extract metrics
run_benchmark() {
    local name=$1
    local url=$2
    local method=$3
    local auth_header=$4
    local body=$5

    echo -e "\n${BLUE}Benchmarking: $name${NC}"

    if [ -n "$body" ]; then
        hey -n $REQUESTS -c $CONCURRENCY -m $method \
            -H "Content-Type: application/json" \
            $auth_header \
            -d "$body" \
            "$url" 2>&1 | grep -E "(Requests/sec|Average|Fastest|Slowest|requests in)"
    else
        hey -n $REQUESTS -c $CONCURRENCY -m $method \
            -H "Content-Type: application/json" \
            $auth_header \
            "$url" 2>&1 | grep -E "(Requests/sec|Average|Fastest|Slowest|requests in)"
    fi
}

echo ""
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  1. GET /health (No Auth)${NC}"
echo -e "${BLUE}============================================${NC}"

echo -e "\n${GREEN}>>> BINGSAN${NC}"
hey -n $REQUESTS -c $CONCURRENCY "$BINGSAN_URL/health" 2>&1 | grep -E "(Requests/sec|Average|Fastest|Slowest|requests in)"

echo -e "\n${GREEN}>>> LAKEKEEPER${NC}"
hey -n $REQUESTS -c $CONCURRENCY "$LAKEKEEPER_URL/health" 2>&1 | grep -E "(Requests/sec|Average|Fastest|Slowest|requests in)"

echo ""
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  2. GET Namespace (Read Operation)${NC}"
echo -e "${BLUE}============================================${NC}"

echo -e "\n${GREEN}>>> BINGSAN${NC}"
hey -n $REQUESTS -c $CONCURRENCY \
    -H "Authorization: Bearer $BINGSAN_TOKEN" \
    "$BINGSAN_URL/v1/namespaces/benchmark_ns" 2>&1 | grep -E "(Requests/sec|Average|Fastest|Slowest|requests in)"

echo -e "\n${GREEN}>>> LAKEKEEPER${NC}"
hey -n $REQUESTS -c $CONCURRENCY \
    "$LAKEKEEPER_URL/catalog/v1/$LAKEKEEPER_PROJECT/$LAKEKEEPER_WAREHOUSE/namespaces/benchmark_ns" 2>&1 | grep -E "(Requests/sec|Average|Fastest|Slowest|requests in)"

echo ""
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  3. LIST Namespaces${NC}"
echo -e "${BLUE}============================================${NC}"

echo -e "\n${GREEN}>>> BINGSAN${NC}"
hey -n $REQUESTS -c $CONCURRENCY \
    -H "Authorization: Bearer $BINGSAN_TOKEN" \
    "$BINGSAN_URL/v1/namespaces" 2>&1 | grep -E "(Requests/sec|Average|Fastest|Slowest|requests in)"

echo -e "\n${GREEN}>>> LAKEKEEPER${NC}"
hey -n $REQUESTS -c $CONCURRENCY \
    "$LAKEKEEPER_URL/catalog/v1/$LAKEKEEPER_PROJECT/$LAKEKEEPER_WAREHOUSE/namespaces" 2>&1 | grep -E "(Requests/sec|Average|Fastest|Slowest|requests in)"

echo ""
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  4. CREATE Namespace (Write Operation)${NC}"
echo -e "${BLUE}============================================${NC}"

# Use unique namespace names for each request
echo -e "\n${GREEN}>>> BINGSAN (creating 100 namespaces)${NC}"
for i in $(seq 1 $REQUESTS); do
    curl -s -X POST "$BINGSAN_URL/v1/namespaces" \
        -H "Authorization: Bearer $BINGSAN_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"namespace\": [\"bench_$i\"]}" > /dev/null &
    if [ $((i % CONCURRENCY)) -eq 0 ]; then
        wait
    fi
done
wait
echo "  Completed $REQUESTS namespace creations"

echo -e "\n${GREEN}>>> LAKEKEEPER (creating 100 namespaces)${NC}"
for i in $(seq 1 $REQUESTS); do
    curl -s -X POST "$LAKEKEEPER_URL/catalog/v1/$LAKEKEEPER_PROJECT/$LAKEKEEPER_WAREHOUSE/namespaces" \
        -H "Content-Type: application/json" \
        -d "{\"namespace\": [\"bench_$i\"]}" > /dev/null &
    if [ $((i % CONCURRENCY)) -eq 0 ]; then
        wait
    fi
done
wait
echo "  Completed $REQUESTS namespace creations"

echo ""
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  SUMMARY${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""
echo "Run complete. For detailed comparison, check the metrics above."
echo ""
echo "Key metrics to compare:"
echo "  - Requests/sec: Higher is better"
echo "  - Average latency: Lower is better"
echo "  - Slowest request: Lower is better (p99 proxy)"
