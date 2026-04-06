#!/bin/bash
# API tests for order endpoints
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

# Login as buyer
BUYER_JAR=$(mktemp)
curl -s -c "$BUYER_JAR" -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"buyer1","password":"testpass123"}' > /dev/null
BUYER_CSRF=$(grep csrf_token "$BUYER_JAR" | awk '{print $NF}')

# Login as seller
SELLER_JAR=$(mktemp)
curl -s -c "$SELLER_JAR" -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"seller1","password":"testpass123"}' > /dev/null
SELLER_CSRF=$(grep csrf_token "$SELLER_JAR" | awk '{print $NF}')

# Login as admin
ADMIN_JAR=$(mktemp)
curl -s -c "$ADMIN_JAR" -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"testpass123"}' > /dev/null
ADMIN_CSRF=$(grep csrf_token "$ADMIN_JAR" | awk '{print $NF}')

cleanup() { rm -f "$BUYER_JAR" "$SELLER_JAR" "$ADMIN_JAR"; }
trap cleanup EXIT

COLLECTIBLE_ID="10000000-0000-0000-0000-000000000001"
IDEM_KEY=$(python3 -c "import uuid; print(uuid.uuid4())")

echo "=== Orders API Tests ==="

# Test 1: Create order as buyer
echo ""
echo "--- Create Order ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -H "Idempotency-Key: $IDEM_KEY" \
    -X POST "$BASE_URL/api/orders" \
    -d "{\"collectible_id\":\"$COLLECTIBLE_ID\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Create order as buyer returns 201" "201" "$HTTP_CODE"

ORDER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null)
assert_json_field "Order status is pending" "$BODY" "['status']" "pending"

if [ -n "$ORDER_ID" ] && [ "$ORDER_ID" != "None" ]; then
    echo "  PASS: Order created with ID ($ORDER_ID)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Order should have an ID"
    FAIL=$((FAIL + 1))
fi

# Test 2: Idempotency - same key returns same order
echo ""
echo "--- Idempotency ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -H "Idempotency-Key: $IDEM_KEY" \
    -X POST "$BASE_URL/api/orders" \
    -d "{\"collectible_id\":\"$COLLECTIBLE_ID\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
IDEM_ORDER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null)
if [ "$IDEM_ORDER_ID" = "$ORDER_ID" ]; then
    echo "  PASS: Idempotent request returns same order"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Idempotent request should return same order (got $IDEM_ORDER_ID vs $ORDER_ID)"
    FAIL=$((FAIL + 1))
fi

# Test 3: Create order without idempotency key
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/orders" \
    -d "{\"collectible_id\":\"$COLLECTIBLE_ID\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Create order without Idempotency-Key returns 400" "400" "$HTTP_CODE"

# Test 4: Seller cannot create order
echo ""
echo "--- Permission Tests ---"
NEW_IDEM=$(python3 -c "import uuid; print(uuid.uuid4())")
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $SELLER_CSRF" \
    -H "Idempotency-Key: $NEW_IDEM" \
    -X POST "$BASE_URL/api/orders" \
    -d "{\"collectible_id\":\"$COLLECTIBLE_ID\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Seller cannot create order (403)" "403" "$HTTP_CODE"

# Test 5: Get order
echo ""
echo "--- Get Order ---"
if [ -n "$ORDER_ID" ] && [ "$ORDER_ID" != "None" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        "$BASE_URL/api/orders/$ORDER_ID")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    assert_status "Get order by ID returns 200" "200" "$HTTP_CODE"
    assert_json_field "Order has correct status" "$BODY" "['status']" "pending"
fi

# Test 6: List orders as buyer
echo ""
echo "--- List Orders ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/orders?role=buyer")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "List orders as buyer returns 200" "200" "$HTTP_CODE"

# Test 7: List orders as seller
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
    -H "X-CSRF-Token: $SELLER_CSRF" \
    "$BASE_URL/api/orders?role=seller")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "List orders as seller returns 200" "200" "$HTTP_CODE"

# Test 8: Seller confirms order (pending -> confirmed)
echo ""
echo "--- Order State Transitions ---"
if [ -n "$ORDER_ID" ] && [ "$ORDER_ID" != "None" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $SELLER_CSRF" \
        -X POST "$BASE_URL/api/orders/$ORDER_ID/confirm")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    assert_status "Seller confirms order (200)" "200" "$HTTP_CODE"
    assert_json_field "Order is now confirmed" "$BODY" "['status']" "confirmed"

    # Test 9: Seller processes order (confirmed -> processing)
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $SELLER_CSRF" \
        -X POST "$BASE_URL/api/orders/$ORDER_ID/process")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    assert_status "Seller processes order (200)" "200" "$HTTP_CODE"
    assert_json_field "Order is now processing" "$BODY" "['status']" "processing"

    # Test 10: Seller completes order (processing -> completed)
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $SELLER_CSRF" \
        -X POST "$BASE_URL/api/orders/$ORDER_ID/complete")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    assert_status "Seller completes order (200)" "200" "$HTTP_CODE"
    assert_json_field "Order is now completed" "$BODY" "['status']" "completed"

    # Test 11: Cannot transition completed order
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $SELLER_CSRF" \
        -X POST "$BASE_URL/api/orders/$ORDER_ID/process")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Cannot transition completed order (422)" "422" "$HTTP_CODE"
fi

# Test 12: Create and cancel order
echo ""
echo "--- Order Cancellation ---"
CANCEL_IDEM=$(python3 -c "import uuid; print(uuid.uuid4())")
# Use a different collectible for the cancel test
COLLECTIBLE_ID_2="10000000-0000-0000-0000-000000000002"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -H "Idempotency-Key: $CANCEL_IDEM" \
    -X POST "$BASE_URL/api/orders" \
    -d "{\"collectible_id\":\"$COLLECTIBLE_ID_2\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
CANCEL_ORDER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null)

if [ -n "$CANCEL_ORDER_ID" ] && [ "$CANCEL_ORDER_ID" != "None" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        -X POST "$BASE_URL/api/orders/$CANCEL_ORDER_ID/cancel" \
        -d '{"reason":"Changed my mind"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    assert_status "Buyer cancels pending order (200)" "200" "$HTTP_CODE"
    assert_json_field "Order is now cancelled" "$BODY" "['status']" "cancelled"

    # Test 13: Cannot transition cancelled order
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $SELLER_CSRF" \
        -X POST "$BASE_URL/api/orders/$CANCEL_ORDER_ID/confirm")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Cannot confirm cancelled order (422)" "422" "$HTTP_CODE"
fi

# Test 14: Cancel without reason fails
echo ""
echo "--- Cancel Validation ---"
CANCEL_IDEM_3=$(python3 -c "import uuid; print(uuid.uuid4())")
COLLECTIBLE_ID_3="10000000-0000-0000-0000-000000000003"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -H "Idempotency-Key: $CANCEL_IDEM_3" \
    -X POST "$BASE_URL/api/orders" \
    -d "{\"collectible_id\":\"$COLLECTIBLE_ID_3\"}")
BODY=$(echo "$RESPONSE" | sed '$d')
CANCEL_ORDER_ID_3=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ -n "$CANCEL_ORDER_ID_3" ] && [ "$CANCEL_ORDER_ID_3" != "None" ] && [ "$CANCEL_ORDER_ID_3" != "" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        -X POST "$BASE_URL/api/orders/$CANCEL_ORDER_ID_3/cancel" \
        -d '{}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Cancel without reason returns 422" "422" "$HTTP_CODE"
fi

# Test 15: Buyer cannot confirm order (seller action)
echo ""
echo "--- Role Enforcement ---"
if [ -n "$CANCEL_ORDER_ID_3" ] && [ "$CANCEL_ORDER_ID_3" != "None" ] && [ "$CANCEL_ORDER_ID_3" != "" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        -X POST "$BASE_URL/api/orders/$CANCEL_ORDER_ID_3/confirm")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Buyer cannot confirm order (403)" "403" "$HTTP_CODE"
fi

echo ""
echo "=== Orders Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
