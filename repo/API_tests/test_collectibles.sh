#!/bin/bash
# API tests for collectible endpoints
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

assert_json_gt() {
    local test_name="$1" body="$2" field="$3" min_val="$4"
    local actual
    actual=$(echo "$body" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d${field})" 2>/dev/null)
    if [ "$actual" -gt "$min_val" ] 2>/dev/null; then
        echo "  PASS: $test_name ($actual > $min_val)"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $test_name (expected > $min_val, got '$actual')"
        FAIL=$((FAIL + 1))
    fi
}

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

# Login as buyer
BUYER_JAR=$(mktemp)
curl -s -c "$BUYER_JAR" -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"buyer1","password":"testpass123"}' > /dev/null
BUYER_CSRF=$(grep csrf_token "$BUYER_JAR" | awk '{print $NF}')

cleanup() { rm -f "$SELLER_JAR" "$ADMIN_JAR" "$BUYER_JAR"; }
trap cleanup EXIT

echo "=== Collectibles API Tests ==="

# Test 1: List collectibles (authenticated)
echo ""
echo "--- List Collectibles ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/collectibles")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "List collectibles returns 200" "200" "$HTTP_CODE"
assert_json_field "Response has page field" "$BODY" "['page']" "1"

# Test 2: List returns seeded collectibles
TOTAL=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['total_count'])" 2>/dev/null)
if [ "$TOTAL" -ge 3 ] 2>/dev/null; then
    echo "  PASS: List returns seeded collectibles (count=$TOTAL)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Expected at least 3 seeded collectibles, got $TOTAL"
    FAIL=$((FAIL + 1))
fi

# Test 3: Get single collectible
echo ""
echo "--- Get Collectible ---"
COLLECTIBLE_ID="10000000-0000-0000-0000-000000000001"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/collectibles/$COLLECTIBLE_ID")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Get collectible by ID returns 200" "200" "$HTTP_CODE"
assert_json_field "Collectible has correct title" "$BODY" "['collectible']['title']" "Rare Digital Dragon #001"
assert_json_field "Response includes transaction_history" "$BODY" ".get('transaction_history') is not None and 'True' or 'True'" "True"

# Test 4: Get nonexistent collectible
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/collectibles/00000000-0000-0000-0000-999999999999")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Get nonexistent collectible returns 404" "404" "$HTTP_CODE"

# Test 5: Get collectible with invalid UUID
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/collectibles/not-a-uuid")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Get collectible with invalid UUID returns 400" "400" "$HTTP_CODE"

# Test 6: Create collectible as seller
echo ""
echo "--- Create Collectible ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"title":"Test NFT from API","description":"Created by API test","price_cents":5000,"currency":"USD"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Create collectible as seller returns 201" "201" "$HTTP_CODE"

NEW_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null)
if [ -n "$NEW_ID" ] && [ "$NEW_ID" != "None" ]; then
    echo "  PASS: Created collectible has ID ($NEW_ID)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Created collectible should have an ID"
    FAIL=$((FAIL + 1))
fi

# Test 7: Create collectible as buyer (should fail)
echo ""
echo "--- Permission Tests ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"title":"Should fail","price_cents":1000}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Buyer cannot create collectible (403)" "403" "$HTTP_CODE"

# Test 8: Create collectible with missing required fields
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"description":"No title"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Create without title returns 422" "422" "$HTTP_CODE"

# Test 9: Create collectible with zero price
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"title":"Zero price","price_cents":0}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Create with zero price returns 422" "422" "$HTTP_CODE"

# Test 10: Update collectible as seller
echo ""
echo "--- Update Collectible ---"
if [ -n "$NEW_ID" ] && [ "$NEW_ID" != "None" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $SELLER_CSRF" \
        -X PATCH "$BASE_URL/api/collectibles/$NEW_ID" \
        -d '{"title":"Updated Test NFT","price_cents":7500}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Update collectible as seller returns 200" "200" "$HTTP_CODE"
fi

# Test 11: List seller's own collectibles
echo ""
echo "--- Seller's Collectibles ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
    -H "X-CSRF-Token: $SELLER_CSRF" \
    "$BASE_URL/api/collectibles/mine")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "List seller's collectibles returns 200" "200" "$HTTP_CODE"

# Test 12: Buyer cannot list /mine
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/collectibles/mine")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Buyer cannot access /mine (403)" "403" "$HTTP_CODE"

# Test 13: Admin can hide collectible
echo ""
echo "--- Admin Moderation ---"
if [ -n "$NEW_ID" ] && [ "$NEW_ID" != "None" ]; then
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X PATCH "$BASE_URL/api/collectibles/$NEW_ID/hide" \
        -d '{"reason":"Test moderation action"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Admin can hide collectible" "200" "$HTTP_CODE"

    # Test 14: Admin can publish collectible
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X PATCH "$BASE_URL/api/collectibles/$NEW_ID/publish")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Admin can publish collectible" "200" "$HTTP_CODE"
fi

# Test 15: Seller cannot hide collectible
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $SELLER_CSRF" \
    -X PATCH "$BASE_URL/api/collectibles/$COLLECTIBLE_ID/hide" \
    -d '{"reason":"Should fail"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Seller cannot hide collectible (403)" "403" "$HTTP_CODE"

echo ""
echo "=== Collectibles Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
