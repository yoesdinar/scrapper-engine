#!/bin/bash

# This script simulates concurrent load on all workers
# Usage: ./scripts/test-concurrent-load.sh [requests_per_worker]

REQUESTS_PER_WORKER=${1:-10}

WORKER1_URL="http://localhost:8082"
WORKER2_URL="http://localhost:8083"
WORKER3_URL="http://localhost:8084"

echo "=== Concurrent Load Test ==="
echo "Sending ${REQUESTS_PER_WORKER} requests to each worker simultaneously..."
echo ""

hit_worker() {
    local worker_url=$1
    local worker_name=$2
    local request_num=$3
    
    start_time=$(date +%s%3N)
    response=$(curl -s -w "\n%{http_code}" "${worker_url}/hit" 2>/dev/null)
    end_time=$(date +%s%3N)
    
    http_code=$(echo "$response" | tail -n1)
    duration=$((end_time - start_time))
    
    if [ "$http_code" = "200" ]; then
        echo "[${worker_name}] Request #${request_num}: ✓ ${duration}ms"
    else
        echo "[${worker_name}] Request #${request_num}: ✗ HTTP ${http_code}"
    fi
}

export -f hit_worker

declare -a pids

for i in $(seq 1 $REQUESTS_PER_WORKER); do
    hit_worker "$WORKER1_URL" "Worker1" "$i" &
    pids+=($!)
    hit_worker "$WORKER2_URL" "Worker2" "$i" &
    pids+=($!)
    hit_worker "$WORKER3_URL" "Worker3" "$i" &
    pids+=($!)
done

for pid in "${pids[@]}"; do
    wait "$pid"
done

echo ""
echo "=== Load Test Complete ==="
echo "Total requests sent: $((REQUESTS_PER_WORKER * 3))"
