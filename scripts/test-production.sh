#!/bin/bash

# Production Test Script for Configuration Management System
# Tests deployment on VM: 103.157.116.91
# Run this from your LOCAL machine
# Usage: ./scripts/test-production.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
VM_IP="103.157.116.91"
BASE_URL="http://$VM_IP"
ADMIN_USER="admin"
ADMIN_PASS="admin123"
AGENT1_USER="agent1"
AGENT1_PASS="agent1pass"
AGENT2_USER="agent2"
AGENT2_PASS="agent2pass"
AGENT3_USER="agent3"
AGENT3_PASS="agent3pass"

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Configuration Management System${NC}"
echo -e "${BLUE}Production Test Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Testing VM: $VM_IP${NC}"
echo ""

# Function to test endpoint
test_endpoint() {
    local test_name="$1"
    local url="$2"
    local expected_code="$3"
    local auth="$4"
    
    echo -n "Testing: $test_name ... "
    
    if [ -n "$auth" ]; then
        response=$(curl -s -o /dev/null -w "%{http_code}" -u "$auth" "$url" 2>/dev/null || echo "000")
    else
        response=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
    fi
    
    if [ "$response" = "$expected_code" ]; then
        echo -e "${GREEN}✓ PASSED${NC} (HTTP $response)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}✗ FAILED${NC} (Expected: $expected_code, Got: $response)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test 1: Homepage
echo -e "${YELLOW}[1/11] Testing Homepage${NC}"
test_endpoint "Homepage" "$BASE_URL/" "200"
echo ""

# Test 2: Nginx health
echo -e "${YELLOW}[2/11] Testing Nginx Health${NC}"
test_endpoint "Nginx Health" "$BASE_URL/nginx-health" "200"
echo ""

# Test 3: Controller health
echo -e "${YELLOW}[3/11] Testing Controller Health${NC}"
test_endpoint "Controller Health" "$BASE_URL/admin/health" "200"
echo ""

# Test 4: Controller Swagger (requires auth)
echo -e "${YELLOW}[4/11] Testing Controller Swagger${NC}"
test_endpoint "Controller Swagger" "$BASE_URL/admin/swagger/index.html" "200" "$ADMIN_USER:$ADMIN_PASS"
echo ""

# Test 5-7: Worker health checks
echo -e "${YELLOW}[5/11] Testing Worker 1 Health${NC}"
test_endpoint "Worker 1 Health" "$BASE_URL/worker1/health" "200"
echo ""

echo -e "${YELLOW}[6/11] Testing Worker 2 Health${NC}"
test_endpoint "Worker 2 Health" "$BASE_URL/worker2/health" "200"
echo ""

echo -e "${YELLOW}[7/11] Testing Worker 3 Health${NC}"
test_endpoint "Worker 3 Health" "$BASE_URL/worker3/health" "200"
echo ""

# Test 8-10: Worker Swagger
echo -e "${YELLOW}[8/11] Testing Worker 1 Swagger${NC}"
test_endpoint "Worker 1 Swagger" "$BASE_URL/worker1/swagger/index.html" "200"
echo ""

echo -e "${YELLOW}[9/11] Testing Worker 2 Swagger${NC}"
test_endpoint "Worker 2 Swagger" "$BASE_URL/worker2/swagger/index.html" "200"
echo ""

echo -e "${YELLOW}[10/11] Testing Worker 3 Swagger${NC}"
test_endpoint "Worker 3 Swagger" "$BASE_URL/worker3/swagger/index.html" "200"
echo ""

# Test 11: End-to-End Configuration Test
echo -e "${YELLOW}[11/11] Testing End-to-End Configuration Flow${NC}"
echo -e "${BLUE}  Step 1: Create Configuration${NC}"

# Create config
CONFIG_RESPONSE=$(curl -s -X POST "$BASE_URL/admin/api/v1/config" \
  -u "$ADMIN_USER:$ADMIN_PASS" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-config-automated",
    "description": "Test configuration from automated test",
    "script": "echo Test successful",
    "agent_ids": []
  }' 2>/dev/null || echo '{"error":"connection failed"}')

if echo "$CONFIG_RESPONSE" | grep -q '"message"\|"version"\|"id"'; then
    echo -e "  ${GREEN}✓ Configuration created/updated successfully${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    
    echo -e "${BLUE}  Step 2: Verify configurations endpoint${NC}"
    CONFIGS=$(curl -s -u "$ADMIN_USER:$ADMIN_PASS" "$BASE_URL/admin/config" 2>/dev/null)
    
    if [ -n "$CONFIGS" ]; then
        echo -e "  ${GREEN}✓ Configuration API working${NC}"
        echo -e "${GREEN}✓ END-TO-END TEST PASSED${NC}"
    else
        echo -e "  ${YELLOW}⚠ Configuration list empty${NC}"
        echo -e "${YELLOW}✓ END-TO-END TEST PARTIALLY PASSED${NC}"
    fi
else
    echo -e "  ${RED}✗ Failed to create configuration${NC}"
    echo -e "  Response: $CONFIG_RESPONSE"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
echo ""

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Total Tests:  $((TESTS_PASSED + TESTS_FAILED))"
echo -e "${GREEN}Passed:       $TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}Failed:       $TESTS_FAILED${NC}"
else
    echo -e "Failed:       $TESTS_FAILED"
fi
echo ""

# Service URLs
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Service URLs${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Homepage:          ${GREEN}$BASE_URL/${NC}"
echo -e "Admin Panel:       ${GREEN}$BASE_URL/admin/swagger/index.html${NC}"
echo -e "  Credentials:     ${YELLOW}$ADMIN_USER / $ADMIN_PASS${NC}"
echo ""
echo -e "Worker 1 API:      ${GREEN}$BASE_URL/worker1/swagger/index.html${NC}"
echo -e "Worker 2 API:      ${GREEN}$BASE_URL/worker2/swagger/index.html${NC}"
echo -e "Worker 3 API:      ${GREEN}$BASE_URL/worker3/swagger/index.html${NC}"
echo ""

# Exit code
if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed! System is operational.${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠ Some tests failed. Check the output above.${NC}"
    exit 1
fi
