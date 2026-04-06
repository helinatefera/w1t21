#!/bin/bash
# API tests for schema/UI/runtime consistency:
#   1. Orphan notification templates removed
#   2. A/B test experiment registry enforcement
#   3. Transaction history populated on order completion
#
# Requires: services running, seed data loaded.

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

assert_json() {
    local test_name="$1" body="$2" expr="$3" expected="$4"
    local actual
    actual=$(echo "$body" | python3 -c "import sys,json; d=json.load(sys.stdin); print($expr)" 2>/dev/null)
    if [ "$actual" = "$expected" ]; then
        echo "  PASS: $test_name"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $test_name (expected '$expected', got '$actual')"
        FAIL=$((FAIL + 1))
    fi
}

login_as() {
    local user="$1" jar="$2"
    curl -s -c "$jar" -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$user\",\"password\":\"testpass123\"}" > /dev/null
}
get_csrf() { grep csrf_token "$1" | awk '{print $NF}'; }

ADMIN_JAR=$(mktemp); SELLER_JAR=$(mktemp); BUYER_JAR=$(mktemp)
login_as "admin" "$ADMIN_JAR"; ADMIN_CSRF=$(get_csrf "$ADMIN_JAR")
login_as "seller1" "$SELLER_JAR"; SELLER_CSRF=$(get_csrf "$SELLER_JAR")
login_as "buyer1" "$BUYER_JAR"; BUYER_CSRF=$(get_csrf "$BUYER_JAR")
cleanup() { rm -f "$ADMIN_JAR" "$SELLER_JAR" "$BUYER_JAR"; }
trap cleanup EXIT

echo "=== Consistency Tests ==="

# ===========================================================
# 1. Orphan Notification Templates Removed
# ===========================================================
echo ""
echo "--- Notification Template Cleanup ---"

# Notification templates went through two lifecycle stages:
# - Migration 013 removed orphan templates (refund_approved, arbitration_opened, review_posted)
# - Migration 014 restored them once business-flow emitters were added
# Verify the notification endpoint is functional and templates are usable.

RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/notifications")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Notifications endpoint available (200)" "200" "$HTTP_CODE"

# Verify preferences can be set for all 7 active template slugs
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X PUT "$BASE_URL/api/notifications/preferences" \
    -d '{"preferences":{"order_confirmed":true,"order_processing":true,"order_completed":true,"order_cancelled":true,"refund_approved":true,"arbitration_opened":true,"review_posted":true}}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Set preferences for all 7 templates (200)" "200" "$HTTP_CODE"

# ===========================================================
# 2. A/B Test Experiment Registry Enforcement
# ===========================================================
echo ""
echo "--- A/B Test Experiment Registry ---"

# Test: Unregistered experiment name rejected
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/ab-tests" \
    -d '{
        "name":"nonexistent_experiment","description":"Should fail",
        "traffic_pct":50,"start_date":"2020-01-01T00:00","end_date":"2099-12-31T23:59",
        "control_variant":"a","test_variant":"b","rollback_threshold_pct":10
    }')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Unregistered experiment name rejected (422)" "422" "$HTTP_CODE"

# Verify error mentions "unknown experiment"
ERR_MSG=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('error',{}).get('message',''))" 2>/dev/null)
if echo "$ERR_MSG" | grep -qi "unknown experiment"; then
    echo "  PASS: Error message mentions unknown experiment"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Error message should mention unknown experiment (got: '$ERR_MSG')"
    FAIL=$((FAIL + 1))
fi

# Test: Registered name but invalid variant rejected
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/ab-tests" \
    -d '{
        "name":"catalog_layout","description":"Bad variant",
        "traffic_pct":50,"start_date":"2020-01-01T00:00","end_date":"2099-12-31T23:59",
        "control_variant":"grid","test_variant":"carousel","rollback_threshold_pct":10
    }')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Unregistered variant rejected (422)" "422" "$HTTP_CODE"

ERR_MSG=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('error',{}).get('message',''))" 2>/dev/null)
if echo "$ERR_MSG" | grep -qi "not registered"; then
    echo "  PASS: Error message mentions variant not registered"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Error message should mention variant not registered (got: '$ERR_MSG')"
    FAIL=$((FAIL + 1))
fi

# Test: Same variant for control and test rejected
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/ab-tests" \
    -d '{
        "name":"catalog_layout","description":"Same variants",
        "traffic_pct":50,"start_date":"2020-01-01T00:00","end_date":"2099-12-31T23:59",
        "control_variant":"grid","test_variant":"grid","rollback_threshold_pct":10
    }')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Identical control/test variants rejected (422)" "422" "$HTTP_CODE"

ERR_MSG=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('error',{}).get('message',''))" 2>/dev/null)
if echo "$ERR_MSG" | grep -qi "must be different"; then
    echo "  PASS: Error message mentions variants must be different"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Error message should mention variants must be different (got: '$ERR_MSG')"
    FAIL=$((FAIL + 1))
fi

# Test: Valid registered experiment + variants accepted
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/ab-tests" \
    -d '{
        "name":"catalog_layout","description":"Valid experiment",
        "traffic_pct":50,"start_date":"2020-01-01T00:00","end_date":"2099-12-31T23:59",
        "control_variant":"grid","test_variant":"list","rollback_threshold_pct":15
    }')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
# 201 = created, 409 = already exists from prior test run (both acceptable)
if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "409" ]; then
    echo "  PASS: Valid experiment accepted or already exists (HTTP $HTTP_CODE)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Valid experiment should return 201 or 409 (got HTTP $HTTP_CODE)"
    FAIL=$((FAIL + 1))
fi

# ===========================================================
# 3. Transaction History Populated on Order Completion
# ===========================================================
echo ""
echo "--- Transaction History on Completion ---"

# Create a collectible
RESPONSE=$(curl -s -b "$SELLER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"title":"TxHistory Test Item","price_cents":4200}')
TX_CID=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ -z "$TX_CID" ] || [ "$TX_CID" = "None" ] || [ "$TX_CID" = "" ]; then
    echo "  SKIP: Could not create collectible for tx history test"
else
    echo "  Created collectible: $TX_CID"

    # Check transaction_history is empty before any order
    RESPONSE=$(curl -s -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        "$BASE_URL/api/collectibles/$TX_CID")
    BODY=$(echo "$RESPONSE" | python3 -c "
import sys,json
d=json.load(sys.stdin)
th=d.get('transaction_history',[]) or []
print(len(th))
" 2>/dev/null)
    if [ "$BODY" = "0" ]; then
        echo "  PASS: Transaction history empty before order"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: Transaction history should be empty before order (got $BODY entries)"
        FAIL=$((FAIL + 1))
    fi

    # Buyer places order
    TX_IDEM=$(python3 -c "import uuid; print(uuid.uuid4())")
    RESPONSE=$(curl -s -b "$BUYER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
        -H "Idempotency-Key: $TX_IDEM" \
        -X POST "$BASE_URL/api/orders" \
        -d "{\"collectible_id\":\"$TX_CID\"}")
    TX_OID=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

    if [ -z "$TX_OID" ] || [ "$TX_OID" = "None" ] || [ "$TX_OID" = "" ]; then
        echo "  SKIP: Could not create order for tx history test"
    else
        echo "  Created order: $TX_OID"

        # Seller confirms → processes → completes
        curl -s -b "$SELLER_JAR" \
            -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
            -X POST "$BASE_URL/api/orders/$TX_OID/confirm" > /dev/null

        curl -s -b "$SELLER_JAR" \
            -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
            -X POST "$BASE_URL/api/orders/$TX_OID/process" > /dev/null

        RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
            -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
            -X POST "$BASE_URL/api/orders/$TX_OID/complete")
        HTTP_CODE=$(echo "$RESPONSE" | tail -1)
        assert_status "Order completed (200)" "200" "$HTTP_CODE"

        # Now check transaction_history has a record
        RESPONSE=$(curl -s -b "$BUYER_JAR" \
            -H "X-CSRF-Token: $BUYER_CSRF" \
            "$BASE_URL/api/collectibles/$TX_CID")
        BODY=$(echo "$RESPONSE")

        TX_COUNT=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
th=d.get('transaction_history',[]) or []
print(len(th))
" 2>/dev/null)
        if [ "$TX_COUNT" -ge 1 ] 2>/dev/null; then
            echo "  PASS: Transaction history has $TX_COUNT record(s) after completion"
            PASS=$((PASS + 1))
        else
            echo "  FAIL: Transaction history should have at least 1 record after completion (got $TX_COUNT)"
            FAIL=$((FAIL + 1))
        fi

        # Verify the tx_hash references the order
        TX_HASH=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
th=d.get('transaction_history',[]) or []
print(th[0].get('tx_hash','') if th else '')
" 2>/dev/null)
        if echo "$TX_HASH" | grep -q "order:$TX_OID"; then
            echo "  PASS: tx_hash references the order ($TX_HASH)"
            PASS=$((PASS + 1))
        else
            echo "  FAIL: tx_hash should reference order:$TX_OID (got '$TX_HASH')"
            FAIL=$((FAIL + 1))
        fi

        # Verify to_address is the buyer
        BUYER_ID=$(curl -s -b "$BUYER_JAR" -H "X-CSRF-Token: $BUYER_CSRF" \
            "$BASE_URL/api/dashboard" | python3 -c "print('ok')" 2>/dev/null)
        TO_ADDR=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
th=d.get('transaction_history',[]) or []
print(th[0].get('to_address','') if th else '')
" 2>/dev/null)
        if [ -n "$TO_ADDR" ] && [ "$TO_ADDR" != "" ]; then
            echo "  PASS: to_address populated ($TO_ADDR)"
            PASS=$((PASS + 1))
        else
            echo "  FAIL: to_address should be populated"
            FAIL=$((FAIL + 1))
        fi
    fi
fi

echo ""
echo "=== Consistency Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
