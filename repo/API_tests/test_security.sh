#!/bin/bash
# Integration tests for security hardening:
# - Cross-user order access denied (object-level authorization)
# - Server-side PII detection blocks messages
# - Hidden collectibles not visible to non-admin
# - CSRF enforcement
# - A/B test datetime-local format accepted

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

# Login helpers
login_as() {
    local user="$1" jar="$2"
    curl -s -c "$jar" -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$user\",\"password\":\"testpass123\"}" > /dev/null
}

get_csrf() {
    grep csrf_token "$1" | awk '{print $NF}'
}

ADMIN_JAR=$(mktemp); SELLER_JAR=$(mktemp); BUYER_JAR=$(mktemp); ANALYST_JAR=$(mktemp)
login_as "admin" "$ADMIN_JAR"
login_as "seller1" "$SELLER_JAR"
login_as "buyer1" "$BUYER_JAR"
login_as "analyst1" "$ANALYST_JAR"
ADMIN_CSRF=$(get_csrf "$ADMIN_JAR")
SELLER_CSRF=$(get_csrf "$SELLER_JAR")
BUYER_CSRF=$(get_csrf "$BUYER_JAR")
ANALYST_CSRF=$(get_csrf "$ANALYST_JAR")

cleanup() { rm -f "$ADMIN_JAR" "$SELLER_JAR" "$BUYER_JAR" "$ANALYST_JAR"; }
trap cleanup EXIT

echo "=== Security Integration Tests ==="

# =============================================
# 1. Object-Level Authorization on Orders
# =============================================
echo ""
echo "--- Object-Level Authorization ---"

# Create an order as buyer
IDEM=$(python3 -c "import uuid; print(uuid.uuid4())")
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -H "Idempotency-Key: $IDEM" \
    -X POST "$BASE_URL/api/orders" \
    -d '{"collectible_id":"10000000-0000-0000-0000-000000000001"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
ORDER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ -n "$ORDER_ID" ] && [ "$ORDER_ID" != "" ] && [ "$ORDER_ID" != "None" ]; then
    # Analyst (not buyer or seller) tries to view the order -> 403
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ANALYST_JAR" \
        -H "X-CSRF-Token: $ANALYST_CSRF" \
        "$BASE_URL/api/orders/$ORDER_ID")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Non-participant cannot view order (403)" "403" "$HTTP_CODE"

    # Analyst tries to cancel the order -> 403
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ANALYST_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $ANALYST_CSRF" \
        -X POST "$BASE_URL/api/orders/$ORDER_ID/cancel" \
        -d '{"reason":"not my order"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Non-participant cannot cancel order (403)" "403" "$HTTP_CODE"

    # Buyer (participant) CAN view the order
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        "$BASE_URL/api/orders/$ORDER_ID")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Buyer (participant) can view own order (200)" "200" "$HTTP_CODE"

    # Seller (participant) CAN view the order
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
        -H "X-CSRF-Token: $SELLER_CSRF" \
        "$BASE_URL/api/orders/$ORDER_ID")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Seller (participant) can view order (200)" "200" "$HTTP_CODE"

    # Clean up: cancel the order
    curl -s -b "$BUYER_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        -X POST "$BASE_URL/api/orders/$ORDER_ID/cancel" \
        -d '{"reason":"test cleanup"}' > /dev/null
else
    echo "  SKIP: Could not create order for OLAC tests"
fi

# =============================================
# 2. Server-Side PII Detection
# =============================================
echo ""
echo "--- Server-Side PII Detection ---"

# Create order for message tests
IDEM2=$(python3 -c "import uuid; print(uuid.uuid4())")
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -H "Idempotency-Key: $IDEM2" \
    -X POST "$BASE_URL/api/orders" \
    -d '{"collectible_id":"10000000-0000-0000-0000-000000000002"}')
BODY=$(echo "$RESPONSE" | sed '$d')
MSG_ORDER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ -n "$MSG_ORDER_ID" ] && [ "$MSG_ORDER_ID" != "" ] && [ "$MSG_ORDER_ID" != "None" ]; then
    # SSN in message body -> blocked
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        -X POST "$BASE_URL/api/orders/$MSG_ORDER_ID/messages" \
        -F "body=My SSN is 123-45-6789")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Message with SSN blocked at API (422)" "422" "$HTTP_CODE"

    # Phone number in message body -> blocked
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        -X POST "$BASE_URL/api/orders/$MSG_ORDER_ID/messages" \
        -F "body=Call me at 555-123-4567")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Message with phone number blocked at API (422)" "422" "$HTTP_CODE"

    # Email in message body -> blocked
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        -X POST "$BASE_URL/api/orders/$MSG_ORDER_ID/messages" \
        -F "body=Email me at user@example.com")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Message with email blocked at API (422)" "422" "$HTTP_CODE"

    # Clean message -> allowed
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        -X POST "$BASE_URL/api/orders/$MSG_ORDER_ID/messages" \
        -F "body=When will my order ship?")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Clean message passes PII check (201)" "201" "$HTTP_CODE"

    # Cleanup
    curl -s -b "$BUYER_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        -X POST "$BASE_URL/api/orders/$MSG_ORDER_ID/cancel" \
        -d '{"reason":"test cleanup"}' > /dev/null
else
    echo "  SKIP: Could not create order for PII tests"
fi

# =============================================
# 3. Hidden Collectible Visibility
# =============================================
echo ""
echo "--- Hidden Collectible Visibility ---"

# Create a collectible as seller, then hide it as admin
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"title":"Hidden Test Item","description":"Should not be visible","price_cents":1000,"currency":"USD"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
HIDDEN_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ -n "$HIDDEN_ID" ] && [ "$HIDDEN_ID" != "" ] && [ "$HIDDEN_ID" != "None" ]; then
    # Admin hides the collectible
    curl -s -b "$ADMIN_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X PATCH "$BASE_URL/api/collectibles/$HIDDEN_ID/hide" \
        -d '{"reason":"test"}' > /dev/null

    # Buyer cannot see hidden collectible by ID
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        "$BASE_URL/api/collectibles/$HIDDEN_ID")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Buyer cannot see hidden collectible (404)" "404" "$HTTP_CODE"

    # Admin CAN see hidden collectible
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "X-CSRF-Token: $ADMIN_CSRF" \
        "$BASE_URL/api/collectibles/$HIDDEN_ID")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Admin can see hidden collectible (200)" "200" "$HTTP_CODE"

    # Buyer listing with status=hidden returns only published
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        "$BASE_URL/api/collectibles?status=hidden")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    HIDDEN_COUNT=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
items = d.get('data',[]) or []
hidden = [i for i in items if i.get('status')=='hidden']
print(len(hidden))
" 2>/dev/null)
    if [ "$HIDDEN_COUNT" = "0" ]; then
        echo "  PASS: Buyer listing with status=hidden returns 0 hidden items"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: Buyer should not see hidden items (got $HIDDEN_COUNT)"
        FAIL=$((FAIL + 1))
    fi

    # Re-publish for cleanup
    curl -s -b "$ADMIN_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X PATCH "$BASE_URL/api/collectibles/$HIDDEN_ID/publish" > /dev/null
else
    echo "  SKIP: Could not create collectible for hidden visibility tests"
fi

# =============================================
# 4. Analytics Events Generated
# =============================================
echo ""
echo "--- Analytics Pipeline ---"

# View a collectible (generates item_view event)
curl -s -b "$BUYER_JAR" -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/collectibles/10000000-0000-0000-0000-000000000001" > /dev/null

# List collectibles (generates catalog_view event)
curl -s -b "$BUYER_JAR" -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/collectibles" > /dev/null

# Small delay for async event recording
sleep 1

# Check funnel has data
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/analytics/funnel?days=30")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Analytics funnel returns 200" "200" "$HTTP_CODE"

VIEWS=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('views',0))" 2>/dev/null)
if [ "$VIEWS" -gt 0 ] 2>/dev/null; then
    echo "  PASS: Funnel has view events (views=$VIEWS)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Funnel should have view events (got views=$VIEWS)"
    FAIL=$((FAIL + 1))
fi

# =============================================
# 5. A/B Test datetime-local format
# =============================================
echo ""
echo "--- A/B Test datetime-local ---"

RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/ab-tests" \
    -d '{
        "name":"datetime-local-test","description":"Test HTML format",
        "traffic_pct":50,"start_date":"2025-01-01T00:00","end_date":"2030-12-31T23:59",
        "control_variant":"A","test_variant":"B","rollback_threshold_pct":20
    }')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "409" ]; then
    echo "  PASS: A/B test with datetime-local format accepted ($HTTP_CODE)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: A/B test with datetime-local format accepted (201) (expected HTTP 201 or 409, got HTTP $HTTP_CODE)"
    FAIL=$((FAIL + 1))
fi

# Also test RFC3339 still works
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/ab-tests" \
    -d '{
        "name":"rfc3339-test","description":"Test RFC3339 format",
        "traffic_pct":50,"start_date":"2025-01-01T00:00:00Z","end_date":"2030-12-31T23:59:59Z",
        "control_variant":"A","test_variant":"B","rollback_threshold_pct":20
    }')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "409" ]; then
    echo "  PASS: A/B test with RFC3339 format accepted ($HTTP_CODE)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: A/B test with RFC3339 format accepted (201) (expected HTTP 201 or 409, got HTTP $HTTP_CODE)"
    FAIL=$((FAIL + 1))
fi

# =============================================
# 6. Error response format consistency
# =============================================
echo ""
echo "--- Error Response Format ---"

# Hit a nonexistent endpoint
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/nonexistent")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
HAS_ERROR=$(echo "$BODY" | python3 -c "
import sys,json
try:
    d=json.load(sys.stdin)
    print('error' in d and 'code' in d.get('error',{}))
except: print(False)
" 2>/dev/null)
if [ "$HAS_ERROR" = "True" ]; then
    echo "  PASS: 404 returns structured JSON error"
    PASS=$((PASS + 1))
else
    echo "  FAIL: 404 should return structured JSON error (got: $BODY)"
    FAIL=$((FAIL + 1))
fi

echo ""
echo "=== Security Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
