#!/bin/bash
# API tests for notification and admin endpoints
# Requires: services running, seed data loaded

BASE_URL="${API_BASE_URL:-http://localhost:8080}"
PASS=0
FAIL=0

assert_status() {
    local test_name="$1" expected="$2" actual="$3"
    if [ "$expected" = "$actual" ]; then
        echo "  PASS: $test_name (HTTP $actual)"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $test_name (expected HTTP $expected, got HTTP $actual)"
        FAIL=$((FAIL + 1))
    fi
}

assert_json_field() {
    local test_name="$1" body="$2" field="$3" expected="$4"
    local actual
    actual=$(echo "$body" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d${field})" 2>/dev/null)
    if [ "$actual" = "$expected" ]; then
        echo "  PASS: $test_name"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $test_name (expected '$expected', got '$actual')"
        FAIL=$((FAIL + 1))
    fi
}

# Login as admin
ADMIN_JAR=$(mktemp)
curl -s -c "$ADMIN_JAR" -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"testpass123"}' > /dev/null
ADMIN_CSRF=$(grep csrf_token "$ADMIN_JAR" | awk '{print $NF}')

# Login as buyer
BUYER_JAR=$(mktemp)
curl -s -c "$BUYER_JAR" -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"buyer1","password":"testpass123"}' > /dev/null
BUYER_CSRF=$(grep csrf_token "$BUYER_JAR" | awk '{print $NF}')

# Login as analyst
ANALYST_JAR=$(mktemp)
curl -s -c "$ANALYST_JAR" -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"analyst1","password":"testpass123"}' > /dev/null
ANALYST_CSRF=$(grep csrf_token "$ANALYST_JAR" | awk '{print $NF}')

cleanup() { rm -f "$ADMIN_JAR" "$BUYER_JAR" "$ANALYST_JAR"; }
trap cleanup EXIT

echo "=== Notifications & Admin API Tests ==="

# --- Notifications ---
echo ""
echo "--- Notification Tests ---"

# Test 1: List notifications
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/notifications")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "List notifications returns 200" "200" "$HTTP_CODE"

# Test 2: Mark all read
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/notifications/read-all")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Mark all read returns 200" "200" "$HTTP_CODE"

# Test 3: Get notification preferences
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/notifications/preferences")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Get notification preferences returns 200" "200" "$HTTP_CODE"

# Test 4: Update notification preferences
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X PUT "$BASE_URL/api/notifications/preferences" \
    -d '{"preferences":{"order_confirmed":true,"order_cancelled":false}}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Update notification preferences returns 200" "200" "$HTTP_CODE"

# Test 5: Update preferences with subscription_mode
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X PUT "$BASE_URL/api/notifications/preferences" \
    -d '{"preferences":{"order_confirmed":true},"subscription_mode":"status_only"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Update preferences with status_only mode (200)" "200" "$HTTP_CODE"

# Test 6: Verify subscription_mode is persisted
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/notifications/preferences")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Get preferences after mode change (200)" "200" "$HTTP_CODE"
assert_json_field "subscription_mode is status_only" "$BODY" "['subscription_mode']" "status_only"

# Test 7: Invalid subscription_mode rejected
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X PUT "$BASE_URL/api/notifications/preferences" \
    -d '{"preferences":{"order_confirmed":true},"subscription_mode":"weekly_digest"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Invalid subscription_mode rejected (422)" "422" "$HTTP_CODE"

# Reset to all_events for subsequent tests
curl -s -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X PUT "$BASE_URL/api/notifications/preferences" \
    -d '{"preferences":{"order_confirmed":true},"subscription_mode":"all_events"}' > /dev/null

# Test 8: Notifications require auth
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/notifications")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Notifications require authentication (401)" "401" "$HTTP_CODE"

# --- Dashboard ---
echo ""
echo "--- Dashboard Tests ---"

# Test 6: Dashboard returns expected fields
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/dashboard")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Dashboard returns 200" "200" "$HTTP_CODE"

HAS_FIELDS=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
required = ['open_orders', 'owned_collectibles', 'unread_notifications', 'seller_open_orders', 'listed_items']
print(all(k in d for k in required))
" 2>/dev/null)
if [ "$HAS_FIELDS" = "True" ]; then
    echo "  PASS: Dashboard has expected fields (including seller counters)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Dashboard missing expected fields (need open_orders, owned_collectibles, unread_notifications, seller_open_orders, listed_items)"
    FAIL=$((FAIL + 1))
fi

# --- Admin: Users ---
echo ""
echo "--- Admin User Management ---"

# Test 7: List users as admin
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/users")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "List users as admin returns 200" "200" "$HTTP_CODE"

# Test 8: Create user as admin
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/users" \
    -d '{"username":"testuser_api","password":"testpass123","display_name":"API Test User"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "409" ]; then
    echo "  PASS: Create user as admin ($HTTP_CODE)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Create user as admin returns 201 (expected HTTP 201 or 409, got HTTP $HTTP_CODE)"
    FAIL=$((FAIL + 1))
fi

TEST_USER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

# Test 9: Add role to user
if [ -n "$TEST_USER_ID" ] && [ "$TEST_USER_ID" != "None" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X POST "$BASE_URL/api/users/$TEST_USER_ID/roles" \
        -d '{"role_name":"buyer"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Add buyer role to user returns 200 or 201" "20" "${HTTP_CODE:0:2}"
fi

# Test 10: Buyer cannot list users
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/users")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Buyer cannot list users (403)" "403" "$HTTP_CODE"

# Test 11: Buyer cannot create user
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/users" \
    -d '{"username":"shouldfail","password":"testpass123","display_name":"Fail"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Buyer cannot create user (403)" "403" "$HTTP_CODE"

# --- Admin: Analytics ---
echo ""
echo "--- Analytics Access ---"

# Test 12: Admin can access analytics funnel
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/analytics/funnel?days=7")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Admin can access analytics funnel (200)" "200" "$HTTP_CODE"

# Test 13: Admin can access retention
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/analytics/retention?days=30")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Admin can access retention (200)" "200" "$HTTP_CODE"

# Test 14: Admin can access content performance
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/analytics/content-performance")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Admin can access content performance (200)" "200" "$HTTP_CODE"

# Test 15: Buyer cannot access analytics
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/analytics/funnel")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Buyer cannot access analytics (403)" "403" "$HTTP_CODE"

# Test 16: Compliance analyst CAN access analytics
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ANALYST_JAR" \
    -H "X-CSRF-Token: $ANALYST_CSRF" \
    "$BASE_URL/api/analytics/funnel?days=7")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Compliance analyst can access analytics (200)" "200" "$HTTP_CODE"

# --- Admin: IP Rules ---
echo ""
echo "--- Admin IP Rules ---"

# Test 17: List IP rules
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/admin/ip-rules")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "List IP rules returns 200" "200" "$HTTP_CODE"

# Test 18: Admin can access anomalies
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/admin/anomalies")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "List anomalies returns 200" "200" "$HTTP_CODE"

# Test 19: Admin can access metrics
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/admin/metrics")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Admin metrics returns 200" "200" "$HTTP_CODE"

# --- A/B Tests ---
echo ""
echo "--- A/B Test Endpoints ---"

# Test 20: List AB tests
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/ab-tests")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "List A/B tests returns 200" "200" "$HTTP_CODE"

# Test 21: Get assignments (available to all authenticated users)
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/ab-tests/assignments")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Get AB test assignments returns 200" "200" "$HTTP_CODE"

# Test 22: Buyer cannot list AB tests
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/ab-tests")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Buyer cannot list A/B tests (403)" "403" "$HTTP_CODE"

echo ""
echo "=== Notifications & Admin Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
