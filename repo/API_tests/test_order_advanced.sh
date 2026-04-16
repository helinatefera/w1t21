#!/bin/bash
# API tests for advanced order endpoints:
#   POST  /api/orders/:id/refund
#   POST  /api/orders/:id/arbitration
#   PATCH /api/orders/:id/fulfillment

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

login_as() {
    local user="$1" jar="$2"
    curl -s -c "$jar" -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$user\",\"password\":\"testpass123\"}" > /dev/null
}
get_csrf() { grep csrf_token "$1" | awk '{print $NF}'; }

SELLER_JAR=$(mktemp); BUYER_JAR=$(mktemp); ANALYST_JAR=$(mktemp)
login_as "seller1" "$SELLER_JAR"; SELLER_CSRF=$(get_csrf "$SELLER_JAR")
login_as "buyer1" "$BUYER_JAR"; BUYER_CSRF=$(get_csrf "$BUYER_JAR")
login_as "analyst1" "$ANALYST_JAR"; ANALYST_CSRF=$(get_csrf "$ANALYST_JAR")
cleanup() { rm -f "$SELLER_JAR" "$BUYER_JAR" "$ANALYST_JAR"; }
trap cleanup EXIT

echo "=== Advanced Order Tests ==="

# Helper: create a collectible and place+complete an order
create_completed_order() {
    local cid oid

    # Create collectible
    cid=$(curl -s -b "$SELLER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
        -X POST "$BASE_URL/api/collectibles" \
        -d '{"title":"Adv Order Test Item","price_cents":2000}' \
        | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

    if [ -z "$cid" ] || [ "$cid" = "None" ]; then echo ""; return; fi

    # Buyer places order
    local idem=$(python3 -c "import uuid; print(uuid.uuid4())")
    oid=$(curl -s -b "$BUYER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
        -H "Idempotency-Key: $idem" \
        -X POST "$BASE_URL/api/orders" \
        -d "{\"collectible_id\":\"$cid\"}" \
        | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

    if [ -z "$oid" ] || [ "$oid" = "None" ]; then echo ""; return; fi

    # Seller: confirm -> process -> complete
    curl -s -b "$SELLER_JAR" -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
        -X POST "$BASE_URL/api/orders/$oid/confirm" > /dev/null
    curl -s -b "$SELLER_JAR" -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
        -X POST "$BASE_URL/api/orders/$oid/process" > /dev/null
    curl -s -b "$SELLER_JAR" -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
        -X POST "$BASE_URL/api/orders/$oid/complete" > /dev/null

    echo "$oid"
}

create_confirmed_order() {
    local cid oid

    cid=$(curl -s -b "$SELLER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
        -X POST "$BASE_URL/api/collectibles" \
        -d '{"title":"Fulfillment Test Item","price_cents":3000}' \
        | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

    if [ -z "$cid" ] || [ "$cid" = "None" ]; then echo ""; return; fi

    local idem=$(python3 -c "import uuid; print(uuid.uuid4())")
    oid=$(curl -s -b "$BUYER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
        -H "Idempotency-Key: $idem" \
        -X POST "$BASE_URL/api/orders" \
        -d "{\"collectible_id\":\"$cid\"}" \
        | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

    if [ -z "$oid" ] || [ "$oid" = "None" ]; then echo ""; return; fi

    curl -s -b "$SELLER_JAR" -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
        -X POST "$BASE_URL/api/orders/$oid/confirm" > /dev/null

    echo "$oid"
}

# ===========================================================
# 1. PATCH /api/orders/:id/fulfillment
# ===========================================================
echo ""
echo "--- Fulfillment Updates ---"

FULFILL_OID=$(create_confirmed_order)

if [ -n "$FULFILL_OID" ] && [ "$FULFILL_OID" != "" ]; then
    # Seller updates fulfillment tracking
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
        -X PATCH "$BASE_URL/api/orders/$FULFILL_OID/fulfillment" \
        -d '{"carrier":"FedEx","tracking_number":"FX123456789"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Seller updates fulfillment (200)" "200" "$HTTP_CODE"

    # Buyer cannot update fulfillment (403)
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
        -X PATCH "$BASE_URL/api/orders/$FULFILL_OID/fulfillment" \
        -d '{"carrier":"UPS","tracking_number":"1Z999"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Buyer cannot update fulfillment (403)" "403" "$HTTP_CODE"

    # Analyst cannot update fulfillment (403)
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ANALYST_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $ANALYST_CSRF" \
        -X PATCH "$BASE_URL/api/orders/$FULFILL_OID/fulfillment" \
        -d '{"carrier":"DHL","tracking_number":"DHL999"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Analyst cannot update fulfillment (403)" "403" "$HTTP_CODE"

    # Verify fulfillment data persisted via GET
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
        -H "X-CSRF-Token: $SELLER_CSRF" \
        "$BASE_URL/api/orders/$FULFILL_OID")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    assert_status "Get order with fulfillment returns 200" "200" "$HTTP_CODE"

    HAS_TRACKING=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
ft=d.get('fulfillment_tracking',{})
print(ft.get('carrier','') == 'FedEx' and ft.get('tracking_number','') == 'FX123456789')
" 2>/dev/null)
    if [ "$HAS_TRACKING" = "True" ]; then
        echo "  PASS: Fulfillment data persisted correctly"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: Fulfillment data not persisted"
        FAIL=$((FAIL + 1))
    fi
else
    echo "  SKIP: Could not create order for fulfillment test"
fi

# ===========================================================
# 2. POST /api/orders/:id/refund
# ===========================================================
echo ""
echo "--- Refund Approval ---"

REFUND_OID=$(create_completed_order)

if [ -n "$REFUND_OID" ] && [ "$REFUND_OID" != "" ]; then
    # Seller approves refund on completed order
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
        -X POST "$BASE_URL/api/orders/$REFUND_OID/refund" \
        -d '{"reason":"Customer requested full refund"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Seller approves refund (200)" "200" "$HTTP_CODE"

    # Buyer cannot approve refund (403)
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
        -X POST "$BASE_URL/api/orders/$REFUND_OID/refund" \
        -d '{"reason":"I want a refund"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Buyer cannot approve refund (403)" "403" "$HTTP_CODE"

    # Refund without reason fails (422)
    REFUND_OID2=$(create_completed_order)
    if [ -n "$REFUND_OID2" ] && [ "$REFUND_OID2" != "" ]; then
        RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
            -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
            -X POST "$BASE_URL/api/orders/$REFUND_OID2/refund" \
            -d '{}')
        HTTP_CODE=$(echo "$RESPONSE" | tail -1)
        assert_status "Refund without reason rejected (422)" "422" "$HTTP_CODE"
    fi
else
    echo "  SKIP: Could not create order for refund test"
fi

# ===========================================================
# 3. POST /api/orders/:id/arbitration
# ===========================================================
echo ""
echo "--- Arbitration ---"

ARB_OID=$(create_completed_order)

if [ -n "$ARB_OID" ] && [ "$ARB_OID" != "" ]; then
    # Buyer opens arbitration
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
        -X POST "$BASE_URL/api/orders/$ARB_OID/arbitration" \
        -d '{"reason":"Item received in damaged condition"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Buyer opens arbitration (200)" "200" "$HTTP_CODE"

    # Seller can also open arbitration
    ARB_OID2=$(create_completed_order)
    if [ -n "$ARB_OID2" ] && [ "$ARB_OID2" != "" ]; then
        RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
            -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
            -X POST "$BASE_URL/api/orders/$ARB_OID2/arbitration" \
            -d '{"reason":"Buyer claims damage but item was shipped correctly"}')
        HTTP_CODE=$(echo "$RESPONSE" | tail -1)
        assert_status "Seller opens arbitration (200)" "200" "$HTTP_CODE"
    fi

    # Analyst (non-participant) cannot open arbitration (403)
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ANALYST_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $ANALYST_CSRF" \
        -X POST "$BASE_URL/api/orders/$ARB_OID/arbitration" \
        -d '{"reason":"I want to intervene"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Non-participant cannot open arbitration (403)" "403" "$HTTP_CODE"

    # Arbitration without reason fails (422)
    ARB_OID3=$(create_completed_order)
    if [ -n "$ARB_OID3" ] && [ "$ARB_OID3" != "" ]; then
        RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
            -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
            -X POST "$BASE_URL/api/orders/$ARB_OID3/arbitration" \
            -d '{}')
        HTTP_CODE=$(echo "$RESPONSE" | tail -1)
        assert_status "Arbitration without reason rejected (422)" "422" "$HTTP_CODE"
    fi
else
    echo "  SKIP: Could not create order for arbitration test"
fi

echo ""
echo "=== Advanced Order Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
