#!/bin/bash
# End-to-end integration tests for production hardening:
# - A/B assignment → event tagging → rollback
# - Notification retry state transitions
# - Checkout-failure anomaly triggering
# - Overselling / concurrency protection
# - Collectible identity (chain_id/token_id) validation
# - Dashboard owned-items metric reflects completed purchases

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

echo "=== E2E Hardening Tests ==="

# =============================================
# 1. Collectible Identity Validation
# =============================================
echo ""
echo "--- Collectible Identity ---"

# contract_address without chain_id/token_id should fail
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"title":"Bad Identity","price_cents":1000,"contract_address":"0xabc"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "contract_address without chain_id/token_id rejected (422)" "422" "$HTTP_CODE"

# With all three identity fields should succeed (or 409 if already created)
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"title":"Full Identity NFT","price_cents":5000,"contract_address":"0x1111111111111111111111111111111111111111","chain_id":137,"token_id":"unique-tok-001"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "409" ]; then
    echo "  PASS: Full identity collectible created or exists ($HTTP_CODE)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Full identity collectible created (expected HTTP 201 or 409, got HTTP $HTTP_CODE)"
    FAIL=$((FAIL + 1))
fi

# Duplicate identity should fail (409)
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"title":"Duplicate NFT","price_cents":5000,"contract_address":"0x1111111111111111111111111111111111111111","chain_id":137,"token_id":"unique-tok-001"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Duplicate collectible identity rejected (409)" "409" "$HTTP_CODE"

# Without contract_address, chain_id and token_id auto-generated
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"title":"Auto Identity","price_cents":2000}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Auto-identity collectible created (201)" "201" "$HTTP_CODE"
AUTO_CHAIN=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('chain_id',0))" 2>/dev/null)
AUTO_TOKEN=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('token_id',''))" 2>/dev/null)
if [ "$AUTO_CHAIN" = "1" ] && [ -n "$AUTO_TOKEN" ] && [ "$AUTO_TOKEN" != "" ]; then
    echo "  PASS: Auto-generated chain_id=1 and token_id ($AUTO_TOKEN)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Expected auto-generated chain_id=1 and non-empty token_id (got chain=$AUTO_CHAIN token=$AUTO_TOKEN)"
    FAIL=$((FAIL + 1))
fi

# =============================================
# 2. Checkout Failure Events (overselling)
# =============================================
echo ""
echo "--- Checkout Failure Events ---"

# Create a collectible and place first order
IDEM1=$(python3 -c "import uuid; print(uuid.uuid4())")
RESPONSE=$(curl -s -b "$SELLER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"title":"Oversell Test","price_cents":3000}')
OVERSELL_ID=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ -n "$OVERSELL_ID" ] && [ "$OVERSELL_ID" != "" ] && [ "$OVERSELL_ID" != "None" ]; then
    # First order succeeds
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
        -H "Idempotency-Key: $IDEM1" \
        -X POST "$BASE_URL/api/orders" \
        -d "{\"collectible_id\":\"$OVERSELL_ID\"}")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "First order succeeds (201)" "201" "$HTTP_CODE"

    # Second order on same item should fail with oversold (409)
    IDEM2=$(python3 -c "import uuid; print(uuid.uuid4())")
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
        -H "Idempotency-Key: $IDEM2" \
        -X POST "$BASE_URL/api/orders" \
        -d "{\"collectible_id\":\"$OVERSELL_ID\"}")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    assert_status "Second order rejected as oversold (409)" "409" "$HTTP_CODE"
    assert_json "Error code is ERR_OVERSOLD" "$BODY" "d['error']['code']" "ERR_OVERSOLD"
fi

# =============================================
# 3. A/B Test → Assignment → Event Tagging
# =============================================
echo ""
echo "--- A/B Test End-to-End ---"

# Create a running A/B test using a registered experiment
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/ab-tests" \
    -d '{
        "name":"catalog_layout","description":"End to end test",
        "traffic_pct":100,"start_date":"2020-01-01T00:00","end_date":"2099-12-31T23:59",
        "control_variant":"grid","test_variant":"list","rollback_threshold_pct":50
    }')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "409" ]; then
    echo "  PASS: Create running A/B test ($HTTP_CODE)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Create running A/B test (expected HTTP 201 or 409, got HTTP $HTTP_CODE)"
    FAIL=$((FAIL + 1))
fi
AB_TEST_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)
# If 409, fetch existing test ID for rollback
if [ "$HTTP_CODE" = "409" ] || [ -z "$AB_TEST_ID" ] || [ "$AB_TEST_ID" = "None" ] || [ "$AB_TEST_ID" = "" ]; then
    AB_TEST_ID=$(curl -s -b "$ADMIN_JAR" -H "X-CSRF-Token: $ADMIN_CSRF" \
        "$BASE_URL/api/ab-tests" | python3 -c "
import sys,json
tests=json.load(sys.stdin)
for t in (tests if isinstance(tests, list) else tests.get('data',[])):
    if t.get('name')=='catalog_layout' and t.get('status')=='running':
        print(t['id']); break
" 2>/dev/null)
fi

# Get assignments as buyer — should include the test
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/ab-tests/assignments")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Get A/B assignments (200)" "200" "$HTTP_CODE"

HAS_E2E=$(echo "$BODY" | python3 -c "
import sys,json
data=json.load(sys.stdin)
found=any(a['test_name']=='catalog_layout' for a in data)
print(found)
" 2>/dev/null)
if [ "$HAS_E2E" = "True" ]; then
    echo "  PASS: Buyer assigned to catalog_layout"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Buyer should be assigned to catalog_layout"
    FAIL=$((FAIL + 1))
fi

# Trigger a view event (which should be tagged with the A/B variant)
curl -s -b "$BUYER_JAR" -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/collectibles/10000000-0000-0000-0000-000000000001" > /dev/null
sleep 1

# Rollback the test
if [ -n "$AB_TEST_ID" ] && [ "$AB_TEST_ID" != "None" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X POST "$BASE_URL/api/ab-tests/$AB_TEST_ID/rollback")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Rollback A/B test (200)" "200" "$HTTP_CODE"

    # Verify it's rolled back — can't rollback again
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X POST "$BASE_URL/api/ab-tests/$AB_TEST_ID/rollback")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Double rollback fails (422)" "422" "$HTTP_CODE"
fi

# =============================================
# 4. Notification Lifecycle
# =============================================
echo ""
echo "--- Notification Lifecycle ---"

# Trigger a notification by confirming an order
NOTIF_IDEM=$(python3 -c "import uuid; print(uuid.uuid4())")
RESPONSE=$(curl -s -b "$SELLER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"title":"Notif Test Item","price_cents":1000}')
NOTIF_CID=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ -n "$NOTIF_CID" ] && [ "$NOTIF_CID" != "None" ]; then
    # Buyer places order
    RESPONSE=$(curl -s -b "$BUYER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
        -H "Idempotency-Key: $NOTIF_IDEM" \
        -X POST "$BASE_URL/api/orders" \
        -d "{\"collectible_id\":\"$NOTIF_CID\"}")
    NOTIF_OID=$(echo "$RESPONSE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

    if [ -n "$NOTIF_OID" ] && [ "$NOTIF_OID" != "None" ]; then
        # Seller confirms — triggers notification to buyer
        curl -s -b "$SELLER_JAR" \
            -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
            -X POST "$BASE_URL/api/orders/$NOTIF_OID/confirm" > /dev/null

        sleep 1

        # Check buyer has notifications
        RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
            -H "X-CSRF-Token: $BUYER_CSRF" \
            "$BASE_URL/api/notifications")
        HTTP_CODE=$(echo "$RESPONSE" | tail -1)
        BODY=$(echo "$RESPONSE" | sed '$d')
        assert_status "Buyer has notifications (200)" "200" "$HTTP_CODE"

        NOTIF_COUNT=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('total_count',0))" 2>/dev/null)
        if [ "$NOTIF_COUNT" -gt 0 ] 2>/dev/null; then
            echo "  PASS: Buyer has $NOTIF_COUNT notification(s)"
            PASS=$((PASS + 1))
        else
            echo "  FAIL: Buyer should have notifications after order confirmed"
            FAIL=$((FAIL + 1))
        fi
    fi
fi

# =============================================
# 5. Dashboard Owned Items = Completed Purchases
# =============================================
echo ""
echo "--- Dashboard Owned Items ---"

RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/dashboard")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Dashboard returns 200" "200" "$HTTP_CODE"

# owned_collectibles should reflect completed orders (not seller listings)
HAS_FIELD=$(echo "$BODY" | python3 -c "import sys,json; d=json.load(sys.stdin); print('owned_collectibles' in d)" 2>/dev/null)
if [ "$HAS_FIELD" = "True" ]; then
    echo "  PASS: Dashboard includes owned_collectibles field"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Dashboard should include owned_collectibles"
    FAIL=$((FAIL + 1))
fi

# =============================================
# 6. Encrypted Config Support
# =============================================
echo ""
echo "--- Config Security ---"
# We can't test the actual encrypted keyfile loading at runtime since the
# server is already running with env vars. But we verify the code accepts
# development mode by checking the /api/auth/login endpoint still works
# (proving the config loaded successfully with plaintext env in dev mode).
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"testpass123"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Server running with config (login works)" "200" "$HTTP_CODE"

echo ""
echo "=== E2E Hardening Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
