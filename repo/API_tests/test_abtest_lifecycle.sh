#!/bin/bash
# API tests for A/B test lifecycle endpoints:
#   GET   /api/ab-tests/:id
#   PATCH /api/ab-tests/:id
#   POST  /api/ab-tests/:id/complete
#   GET   /api/ab-tests/registry

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

ADMIN_JAR=$(mktemp); BUYER_JAR=$(mktemp); ANALYST_JAR=$(mktemp)
login_as "admin" "$ADMIN_JAR"; ADMIN_CSRF=$(get_csrf "$ADMIN_JAR")
login_as "buyer1" "$BUYER_JAR"; BUYER_CSRF=$(get_csrf "$BUYER_JAR")
login_as "analyst1" "$ANALYST_JAR"; ANALYST_CSRF=$(get_csrf "$ANALYST_JAR")
cleanup() { rm -f "$ADMIN_JAR" "$BUYER_JAR" "$ANALYST_JAR"; }
trap cleanup EXIT

echo "=== A/B Test Lifecycle Tests ==="

# ===========================================================
# 1. GET /api/ab-tests/registry
# ===========================================================
echo ""
echo "--- A/B Test Registry ---"

RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/ab-tests/registry")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Get registry returns 200" "200" "$HTTP_CODE"

# Registry should contain catalog_layout
HAS_CATALOG=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
print('catalog_layout' in d)
" 2>/dev/null)
if [ "$HAS_CATALOG" = "True" ]; then
    echo "  PASS: Registry contains catalog_layout experiment"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Registry should contain catalog_layout"
    FAIL=$((FAIL + 1))
fi

# Registry entries should have variants
HAS_VARIANTS=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
entry=d.get('catalog_layout',{})
variants=entry.get('variants',[])
print(len(variants) >= 2)
" 2>/dev/null)
if [ "$HAS_VARIANTS" = "True" ]; then
    echo "  PASS: catalog_layout has at least 2 variants"
    PASS=$((PASS + 1))
else
    echo "  FAIL: catalog_layout should have at least 2 variants"
    FAIL=$((FAIL + 1))
fi

# ===========================================================
# 2. Create a test, then GET /api/ab-tests/:id
# ===========================================================
echo ""
echo "--- Get A/B Test by ID ---"

# Create a test for the lifecycle
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/ab-tests" \
    -d '{
        "name":"checkout_flow","description":"Lifecycle test",
        "traffic_pct":50,"start_date":"2020-01-01T00:00","end_date":"2099-12-31T23:59",
        "control_variant":"standard","test_variant":"express","rollback_threshold_pct":25
    }')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
TEST_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

# If already exists (409), fetch the ID
if [ "$HTTP_CODE" = "409" ] || [ -z "$TEST_ID" ] || [ "$TEST_ID" = "None" ] || [ "$TEST_ID" = "" ]; then
    TEST_ID=$(curl -s -b "$ADMIN_JAR" -H "X-CSRF-Token: $ADMIN_CSRF" \
        "$BASE_URL/api/ab-tests" | python3 -c "
import sys,json
data=json.load(sys.stdin)
tests = data if isinstance(data, list) else data.get('data',[])
for t in tests:
    if t.get('name')=='checkout_flow' and t.get('status') in ('draft','running'):
        print(t['id']); break
" 2>/dev/null)
fi

if [ -n "$TEST_ID" ] && [ "$TEST_ID" != "None" ] && [ "$TEST_ID" != "" ]; then
    # Get by ID as admin
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "X-CSRF-Token: $ADMIN_CSRF" \
        "$BASE_URL/api/ab-tests/$TEST_ID")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    assert_status "Get A/B test by ID returns 200" "200" "$HTTP_CODE"
    assert_json_field "Test has correct name" "$BODY" "['test']['name']" "checkout_flow"

    # Compliance analyst can also view
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ANALYST_JAR" \
        -H "X-CSRF-Token: $ANALYST_CSRF" \
        "$BASE_URL/api/ab-tests/$TEST_ID")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Analyst can view A/B test (200)" "200" "$HTTP_CODE"

    # Buyer cannot view individual test (403)
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        "$BASE_URL/api/ab-tests/$TEST_ID")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Buyer cannot view A/B test (403)" "403" "$HTTP_CODE"

    # ===========================================================
    # 3. PATCH /api/ab-tests/:id
    # ===========================================================
    echo ""
    echo "--- Update A/B Test ---"

    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X PATCH "$BASE_URL/api/ab-tests/$TEST_ID" \
        -d '{"description":"Updated lifecycle description","traffic_pct":75}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    assert_status "Update A/B test returns 200" "200" "$HTTP_CODE"
    assert_json_field "Description updated" "$BODY" "['description']" "Updated lifecycle description"
    assert_json_field "Traffic pct updated" "$BODY" "['traffic_pct']" "75"

    # ===========================================================
    # 4. POST /api/ab-tests/:id/complete
    # ===========================================================
    echo ""
    echo "--- Complete A/B Test ---"

    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X POST "$BASE_URL/api/ab-tests/$TEST_ID/complete")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Complete A/B test returns 200" "200" "$HTTP_CODE"

    # Cannot complete again (422 - not running)
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X POST "$BASE_URL/api/ab-tests/$TEST_ID/complete")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Double complete fails (422)" "422" "$HTTP_CODE"

    # Verify status is completed
    RESPONSE=$(curl -s -b "$ADMIN_JAR" -H "X-CSRF-Token: $ADMIN_CSRF" \
        "$BASE_URL/api/ab-tests/$TEST_ID")
    BODY=$(echo "$RESPONSE")
    assert_json_field "Test status is completed" "$BODY" "['test']['status']" "completed"
else
    echo "  SKIP: Could not create A/B test for lifecycle tests"
fi

# Invalid test ID returns 400
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/ab-tests/not-a-uuid")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Invalid test ID returns 400" "400" "$HTTP_CODE"

echo ""
echo "=== A/B Test Lifecycle Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
