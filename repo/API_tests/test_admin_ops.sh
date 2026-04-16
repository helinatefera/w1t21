#!/bin/bash
# API tests for admin operations:
#   POST   /api/admin/ip-rules
#   DELETE /api/admin/ip-rules/:id
#   PATCH  /api/admin/anomalies/:id/acknowledge

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

echo "=== Admin Operations Tests ==="

# ===========================================================
# 1. POST /api/admin/ip-rules — Create IP Rule
# ===========================================================
echo ""
echo "--- IP Rules: Create ---"

# Admin creates a deny rule (safe — only blocks 10.99.99.0/24 which is unused)
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/admin/ip-rules" \
    -d '{"cidr":"10.99.99.0/24","action":"deny"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Create deny IP rule returns 201" "201" "$HTTP_CODE"
DENY_RULE_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)
assert_json_field "Rule action is deny" "$BODY" "['action']" "deny"
assert_json_field "Rule CIDR matches" "$BODY" "['cidr']" "10.99.99.0/24"

# Invalid CIDR rejected (422)
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/admin/ip-rules" \
    -d '{"cidr":"not-a-cidr","action":"deny"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Invalid CIDR rejected (422)" "422" "$HTTP_CODE"

# Invalid action rejected (422)
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/admin/ip-rules" \
    -d '{"cidr":"10.0.0.0/8","action":"maybe"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Invalid action rejected (422)" "422" "$HTTP_CODE"

# Buyer cannot create IP rules (403)
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/admin/ip-rules" \
    -d '{"cidr":"10.0.0.0/8","action":"deny"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Buyer cannot create IP rules (403)" "403" "$HTTP_CODE"

# Verify rules appear in listing
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/admin/ip-rules")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
RULE_COUNT=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
rules = d if isinstance(d, list) else d.get('data',[])
print(len(rules))
" 2>/dev/null)
if [ "$RULE_COUNT" -ge 1 ] 2>/dev/null; then
    echo "  PASS: IP rules listing shows created rules (count=$RULE_COUNT)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: IP rules listing should have at least 1 rule (got $RULE_COUNT)"
    FAIL=$((FAIL + 1))
fi

# ===========================================================
# 2. DELETE /api/admin/ip-rules/:id
# ===========================================================
echo ""
echo "--- IP Rules: Delete ---"

if [ -n "$DENY_RULE_ID" ] && [ "$DENY_RULE_ID" != "None" ] && [ "$DENY_RULE_ID" != "" ]; then
    # Buyer cannot delete IP rules (403)
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        -X DELETE "$BASE_URL/api/admin/ip-rules/$DENY_RULE_ID")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Buyer cannot delete IP rules (403)" "403" "$HTTP_CODE"

    # Delete the deny rule
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X DELETE "$BASE_URL/api/admin/ip-rules/$DENY_RULE_ID")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Delete IP rule returns 200" "200" "$HTTP_CODE"
else
    echo "  SKIP: No rule ID for delete test"
fi

# ===========================================================
# 3. PATCH /api/admin/anomalies/:id/acknowledge
# ===========================================================
echo ""
echo "--- Anomaly Acknowledgment ---"

# List anomalies to see if any exist
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/admin/anomalies")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "List anomalies returns 200" "200" "$HTTP_CODE"

ANOMALY_ID=$(echo "$BODY" | python3 -c "
import sys,json
d=json.load(sys.stdin)
data=d.get('data',[]) or []
unacked=[a for a in data if not a.get('acknowledged')]
print(unacked[0]['id'] if unacked else '')
" 2>/dev/null)

if [ -n "$ANOMALY_ID" ] && [ "$ANOMALY_ID" != "" ]; then
    # Admin acknowledges anomaly
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X PATCH "$BASE_URL/api/admin/anomalies/$ANOMALY_ID/acknowledge")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Acknowledge anomaly returns 200" "200" "$HTTP_CODE"
else
    # No anomalies — test the endpoint responds correctly with a non-existent ID
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X PATCH "$BASE_URL/api/admin/anomalies/00000000-0000-0000-0000-000000000099/acknowledge")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ]; then
        echo "  PASS: Acknowledge anomaly endpoint responds ($HTTP_CODE)"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: Acknowledge anomaly endpoint should respond 200 or 404 (got $HTTP_CODE)"
        FAIL=$((FAIL + 1))
    fi
fi

# Buyer cannot acknowledge anomalies (403)
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X PATCH "$BASE_URL/api/admin/anomalies/00000000-0000-0000-0000-000000000001/acknowledge")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Buyer cannot acknowledge anomaly (403)" "403" "$HTTP_CODE"

echo ""
echo "=== Admin Operations Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
