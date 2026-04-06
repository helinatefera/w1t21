#!/bin/bash
# API tests for authentication endpoints
# Requires: services running via docker compose up, seed data loaded

BASE_URL="${API_BASE_URL:-http://localhost:8080}"
PASS=0
FAIL=0
COOKIE_JAR=$(mktemp)

cleanup() { rm -f "$COOKIE_JAR"; }
trap cleanup EXIT

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

echo "=== Auth API Tests ==="

# Test 1: Login with valid credentials
echo ""
echo "--- Login Tests ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -c "$COOKIE_JAR" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"testpass123"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Login with valid admin credentials" "200" "$HTTP_CODE"
assert_json_field "Login returns username" "$BODY" "['user']['username']" "admin"
assert_json_field "Login returns admin role" "$BODY" "['roles'][0]" "administrator"

# Test 2: Login returns cookies
COOKIES=$(cat "$COOKIE_JAR")
if echo "$COOKIES" | grep -q "access_token"; then
    echo "  PASS: Login sets access_token cookie"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Login should set access_token cookie"
    FAIL=$((FAIL + 1))
fi

if echo "$COOKIES" | grep -q "csrf_token"; then
    echo "  PASS: Login sets csrf_token cookie"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Login should set csrf_token cookie"
    FAIL=$((FAIL + 1))
fi

# Test 3: Login with invalid password
RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"wrongpassword"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Login with invalid password returns 401" "401" "$HTTP_CODE"
assert_json_field "Invalid login returns error code" "$BODY" "['error']['code']" "ERR_INVALID_CREDENTIALS"

# Test 4: Login with nonexistent user
RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"nonexistent","password":"testpass123"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Login with nonexistent user returns 401" "401" "$HTTP_CODE"

# Test 5: Login with missing fields
RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Login with missing password returns 422" "422" "$HTTP_CODE"

# Test 6: Login with short username
RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"ab","password":"testpass123"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Login with short username returns 422" "422" "$HTTP_CODE"

# Test 7: Login with short password
RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"short"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Login with short password returns 422" "422" "$HTTP_CODE"

# Test 8: Access protected endpoint without auth
echo ""
echo "--- Authorization Tests ---"
RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X GET "$BASE_URL/api/dashboard")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Accessing protected endpoint without auth returns 401" "401" "$HTTP_CODE"

# Test 9: Access protected endpoint with auth
CSRF_TOKEN=$(grep csrf_token "$COOKIE_JAR" | awk '{print $NF}')
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$COOKIE_JAR" \
    -H "X-CSRF-Token: $CSRF_TOKEN" \
    -X GET "$BASE_URL/api/dashboard")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Accessing dashboard with valid auth returns 200" "200" "$HTTP_CODE"

# Test 10: Refresh token
echo ""
echo "--- Token Refresh Tests ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
    -H "X-CSRF-Token: $CSRF_TOKEN" \
    -X POST "$BASE_URL/api/auth/refresh")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Token refresh returns 200" "200" "$HTTP_CODE"

# Test 11: Refresh without cookie
RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST "$BASE_URL/api/auth/refresh")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Refresh without cookie returns 401" "401" "$HTTP_CODE"

# Test 12: Logout
echo ""
echo "--- Logout Tests ---"
CSRF_TOKEN=$(grep csrf_token "$COOKIE_JAR" | awk '{print $NF}')
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$COOKIE_JAR" \
    -H "X-CSRF-Token: $CSRF_TOKEN" \
    -X POST "$BASE_URL/api/auth/logout")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Logout returns 200" "200" "$HTTP_CODE"

# Test 13: Login with seller1
echo ""
echo "--- Role-specific Login Tests ---"
SELLER_JAR=$(mktemp)
RESPONSE=$(curl -s -w "\n%{http_code}" -c "$SELLER_JAR" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"seller1","password":"testpass123"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Login as seller1" "200" "$HTTP_CODE"
assert_json_field "Seller1 username" "$BODY" "['user']['username']" "seller1"
rm -f "$SELLER_JAR"

# Test 14: Login with buyer1
BUYER_JAR=$(mktemp)
RESPONSE=$(curl -s -w "\n%{http_code}" -c "$BUYER_JAR" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"buyer1","password":"testpass123"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Login as buyer1" "200" "$HTTP_CODE"
assert_json_field "Buyer1 username" "$BODY" "['user']['username']" "buyer1"
rm -f "$BUYER_JAR"

# ===========================================================
# Account lockout integration test
# Creates a disposable user, hammers it with 5 bad passwords,
# verifies the 6th attempt returns 423, then unlocks via admin.
# ===========================================================
echo ""
echo "--- Account Lockout Integration Test ---"

# Log in as admin to create a throwaway user
LOCKOUT_ADMIN_JAR=$(mktemp)
curl -s -c "$LOCKOUT_ADMIN_JAR" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"testpass123"}' > /dev/null
LOCKOUT_ADMIN_CSRF=$(grep csrf_token "$LOCKOUT_ADMIN_JAR" | awk '{print $NF}')

LOCKOUT_USER="lockout_test_$(date +%s)"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$LOCKOUT_ADMIN_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $LOCKOUT_ADMIN_CSRF" \
    -X POST "$BASE_URL/api/users" \
    -d "{\"username\":\"$LOCKOUT_USER\",\"password\":\"testpass123\",\"display_name\":\"Lockout Test\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
LOCKOUT_USER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ "$HTTP_CODE" != "201" ] || [ -z "$LOCKOUT_USER_ID" ] || [ "$LOCKOUT_USER_ID" = "" ]; then
    echo "  SKIP: Could not create throwaway user for lockout test (HTTP $HTTP_CODE)"
else
    # Send 5 failed logins
    for i in 1 2 3 4 5; do
        curl -s -X POST "$BASE_URL/api/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"username\":\"$LOCKOUT_USER\",\"password\":\"wrongwrongwrong\"}" > /dev/null
    done

    # 6th attempt — account should be locked (HTTP 423)
    RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$LOCKOUT_USER\",\"password\":\"wrongwrongwrong\"}")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    assert_status "6th failed login returns 423 (account locked)" "423" "$HTTP_CODE"
    assert_json_field "Locked response error code" "$BODY" "['error']['code']" "ERR_ACCOUNT_LOCKED"

    # Even correct password should be rejected while locked
    RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$LOCKOUT_USER\",\"password\":\"testpass123\"}")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Correct password rejected while locked (423)" "423" "$HTTP_CODE"

    # Admin unlocks the account
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$LOCKOUT_ADMIN_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $LOCKOUT_ADMIN_CSRF" \
        -X POST "$BASE_URL/api/users/$LOCKOUT_USER_ID/unlock")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Admin unlocks account (200)" "200" "$HTTP_CODE"

    # Now the correct password should work again
    RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$LOCKOUT_USER\",\"password\":\"testpass123\"}")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Login succeeds after admin unlock (200)" "200" "$HTTP_CODE"
fi
rm -f "$LOCKOUT_ADMIN_JAR"

# ===========================================================
# CSRF negative test
# Sends a protected POST with auth cookies but WITHOUT the
# X-CSRF-Token header and expects 403.
# ===========================================================
echo ""
echo "--- CSRF Negative Tests ---"

# Get fresh cookies
CSRF_JAR=$(mktemp)
curl -s -c "$CSRF_JAR" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"testpass123"}' > /dev/null

# POST without X-CSRF-Token header — should be rejected
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$CSRF_JAR" \
    -X POST "$BASE_URL/api/auth/logout")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "POST /auth/logout without X-CSRF-Token returns 403" "403" "$HTTP_CODE"

CSRF_ERR_MSG=$(echo "$BODY" | python3 -c "
import sys,json
try:
    d=json.load(sys.stdin)
    # Echo wraps HTTPError; try both shapes
    if 'error' in d:
        print(d['error'].get('message',''))
    elif 'message' in d:
        m = d['message']
        if isinstance(m, dict):
            print(m.get('error',{}).get('message',''))
        else:
            print(m)
    else:
        print('')
except: print('')
" 2>/dev/null)
if echo "$CSRF_ERR_MSG" | grep -qi "csrf"; then
    echo "  PASS: Error message mentions CSRF"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Error message should mention CSRF (got: '$CSRF_ERR_MSG')"
    FAIL=$((FAIL + 1))
fi

# PATCH without X-CSRF-Token — should also be rejected
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$CSRF_JAR" \
    -H "Content-Type: application/json" \
    -X POST "$BASE_URL/api/notifications/read-all")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "POST /notifications/read-all without CSRF returns 403" "403" "$HTTP_CODE"

# GET should pass without X-CSRF-Token (safe method, CSRF not checked)
CSRF_TOKEN_VAL=$(grep csrf_token "$CSRF_JAR" | awk '{print $NF}')
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$CSRF_JAR" \
    "$BASE_URL/api/dashboard")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "GET /dashboard without X-CSRF-Token returns 200 (safe method)" "200" "$HTTP_CODE"

# POST with wrong CSRF value — should be rejected
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$CSRF_JAR" \
    -H "X-CSRF-Token: totally-wrong-token" \
    -X POST "$BASE_URL/api/auth/logout")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "POST with wrong X-CSRF-Token returns 403" "403" "$HTTP_CODE"

rm -f "$CSRF_JAR"

echo ""
echo "=== Auth Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
