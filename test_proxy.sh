#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PROXY_PORT=18080
POLICY_FILE="./policy.jsonplaceholder.json"
ALLOWLIST_FILE="./allowlist.jsonplaceholder.json"

echo "Building proxy..."
bazel build //cmd/proxy > /dev/null 2>&1

echo "Starting proxy on port $PROXY_PORT..."
bazel-bin/cmd/proxy/proxy_/proxy \
  --port $PROXY_PORT \
  --allowlist $ALLOWLIST_FILE \
  --policy $POLICY_FILE > /tmp/proxy_test.log 2>&1 &

PROXY_PID=$!
echo "Proxy PID: $PROXY_PID"

# Wait for proxy to start
sleep 2

# Check if proxy is running
if ! ps -p $PROXY_PID > /dev/null; then
  echo -e "${RED}✗ Proxy failed to start${NC}"
  cat /tmp/proxy_test.log
  exit 1
fi

echo -e "${GREEN}✓ Proxy started${NC}"
echo ""

# Cleanup function
cleanup() {
  echo ""
  echo "Stopping proxy..."
  kill $PROXY_PID 2>/dev/null || true
  wait $PROXY_PID 2>/dev/null || true
  echo "Cleanup complete"
}

trap cleanup EXIT

# Test 1: Valid token, allowed route with filter (/users)
echo "Test 1: Valid token + allowed route /users (should filter fields)"
RESPONSE=$(curl -s -H "Authorization: Bearer token_123" \
  -H "Host: jsonplaceholder.typicode.com" \
  "http://localhost:$PROXY_PORT/users" || echo "FAILED")

if echo "$RESPONSE" | jq -e '.[0] | has("id") and has("name") and has("address")' > /dev/null 2>&1; then
  if echo "$RESPONSE" | jq -e '.[0] | has("email")' > /dev/null 2>&1; then
    echo -e "${RED}✗ FAILED: Response should NOT contain email field${NC}"
    echo "$RESPONSE" | jq '.[0]' | head -20
  else
    echo -e "${GREEN}✓ PASSED: Response filtered correctly${NC}"
    echo "Sample filtered response:"
    echo "$RESPONSE" | jq '.[0]' | head -10
  fi
else
  echo -e "${RED}✗ FAILED: Invalid response format${NC}"
  echo "$RESPONSE" | head -20
fi
echo ""

# Test 2: Valid token, specific user (/users/1)
echo "Test 2: Valid token + allowed route /users/1 (should filter fields)"
RESPONSE=$(curl -s -H "Authorization: Bearer token_123" \
  -H "Host: jsonplaceholder.typicode.com" \
  "http://localhost:$PROXY_PORT/users/1" || echo "FAILED")

if echo "$RESPONSE" | jq -e 'has("id") and has("name") and has("address")' > /dev/null 2>&1; then
  if echo "$RESPONSE" | jq -e 'has("email")' > /dev/null 2>&1; then
    echo -e "${RED}✗ FAILED: Response should NOT contain email field${NC}"
    echo "$RESPONSE" | jq '.' | head -20
  else
    echo -e "${GREEN}✓ PASSED: Response filtered correctly${NC}"
    echo "Sample filtered response:"
    echo "$RESPONSE" | jq '.'
  fi
else
  echo -e "${RED}✗ FAILED: Invalid response format${NC}"
  echo "$RESPONSE" | head -20
fi
echo ""

# Test 3: Invalid token
echo "Test 3: Invalid token (should return 403)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer invalid_token" \
  -H "Host: jsonplaceholder.typicode.com" \
  "http://localhost:$PROXY_PORT/users")

if [ "$HTTP_CODE" = "403" ]; then
  echo -e "${GREEN}✓ PASSED: Got 403 for invalid token${NC}"
else
  echo -e "${RED}✗ FAILED: Expected 403, got $HTTP_CODE${NC}"
fi
echo ""

# Test 4: No token
echo "Test 4: No token (should return 403)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Host: jsonplaceholder.typicode.com" \
  "http://localhost:$PROXY_PORT/users")

if [ "$HTTP_CODE" = "403" ]; then
  echo -e "${GREEN}✓ PASSED: Got 403 for missing token${NC}"
else
  echo -e "${RED}✗ FAILED: Expected 403, got $HTTP_CODE${NC}"
fi
echo ""

# Test 5: Valid token, denied route (default deny)
echo "Test 5: Valid token + denied route /posts (should return 403)"
RESPONSE=$(curl -s -H "Authorization: Bearer token_123" \
  -H "Host: jsonplaceholder.typicode.com" \
  "http://localhost:$PROXY_PORT/posts")

if echo "$RESPONSE" | jq -e 'has("error")' > /dev/null 2>&1; then
  ERROR_MSG=$(echo "$RESPONSE" | jq -r '.error')
  echo -e "${GREEN}✓ PASSED: Got error response: $ERROR_MSG${NC}"
else
  echo -e "${RED}✗ FAILED: Expected error response${NC}"
  echo "$RESPONSE" | head -20
fi
echo ""

# Test 6: Host not in allowlist
echo "Test 6: Host not in allowlist (should return 403)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  -H "Authorization: Bearer token_123" \
  -H "Host: evil.com" \
  "http://localhost:$PROXY_PORT/users")

if [ "$HTTP_CODE" = "403" ]; then
  echo -e "${GREEN}✓ PASSED: Got 403 for host not in allowlist${NC}"
else
  echo -e "${RED}✗ FAILED: Expected 403, got $HTTP_CODE${NC}"
fi
echo ""

echo "All tests completed!"
