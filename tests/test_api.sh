#!/bin/sh
# API test script - run inside container

BASE_URL="http://127.0.0.1:8181"

post_json() {
    local url="$1"
    local data="$2"
    echo "$data" > /tmp/req.json
    wget --header="Content-Type: application/json" --post-file=/tmp/req.json -O- "$url" 2>&1
    echo ""
    rm -f /tmp/req.json
}

echo "=== Testing Register Table ==="
post_json "${BASE_URL}/v1/namespaces/test_phase2/register" '{"name": "registered_new", "metadata-location": "s3://warehouse/test_metadata.json"}'

echo ""
echo "=== Testing Rename Table ==="
post_json "${BASE_URL}/v1/tables/rename" '{"source": {"namespace": ["test_phase2"], "name": "commit_test"}, "destination": {"namespace": ["test_phase2"], "name": "renamed_table"}}'

echo ""
echo "=== Testing Table Metrics ==="
post_json "${BASE_URL}/v1/namespaces/test_phase2/tables/renamed_table/metrics" '{"table-uuid": "test-uuid", "metrics": {}}'

echo ""
echo "=== List Tables to Verify ==="
wget -O- "${BASE_URL}/v1/namespaces/test_phase2/tables" 2>&1
