#!/bin/bash
# API tests for rolling-window login lockout policy.
# Covers: threshold enforcement, lockout release via admin unlock,
#         window expiry (via direct DB manipulation), and post-unlock login.
# Requires: services running, seed data loaded, admin account available.

BASE_URL="${API_BASE_URL:-http://localhost:8080}"
DB_URL="${DATABASE_URL:-postgresql://ledgermint:ledgermint@localhost:5432/ledgermint?sslmode=disable}"
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

echo "=== Rolling-Window Login Lockout Tests ==="

# Log in as admin
ADMIN_JAR=$(mktemp)
curl -s -c "$ADMIN_JAR" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"testpass123"}' > /dev/null
ADMIN_CSRF=$(grep csrf_token "$ADMIN_JAR" | awk '{print $NF}')

cleanup() { rm -f "$ADMIN_JAR"; }
trap cleanup EXIT

# ------------------------------------------------------------------
# Test group 1: Five failures in window → lockout on 6th attempt
# ------------------------------------------------------------------
echo ""
echo "--- Threshold Enforcement ---"

LOCKOUT_USER="lockwin_$(date +%s)"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/users" \
    -d "{\"username\":\"$LOCKOUT_USER\",\"password\":\"testpass123\",\"display_name\":\"Lockout Window Test\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
LOCKOUT_USER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ "$HTTP_CODE" != "201" ] || [ -z "$LOCKOUT_USER_ID" ] || [ "$LOCKOUT_USER_ID" = "" ]; then
    echo "  SKIP: Could not create test user (HTTP $HTTP_CODE)"
    echo "=== Rolling-Window Lockout Tests Summary: 0 passed, 0 failed (skipped) ==="
    exit 0
fi

echo "  Created test user: $LOCKOUT_USER ($LOCKOUT_USER_ID)"

# Send 5 failed logins (should all return 401)
for i in 1 2 3 4 5; do
    RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$LOCKOUT_USER\",\"password\":\"wrongpassword\"}")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Failed login attempt $i returns 401" "401" "$HTTP_CODE"
done

# 6th attempt — should be locked
RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$LOCKOUT_USER\",\"password\":\"wrongpassword\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "6th attempt returns 423 (locked)" "423" "$HTTP_CODE"
assert_json_field "Locked response error code" "$BODY" "['error']['code']" "ERR_ACCOUNT_LOCKED"

# Correct password also rejected while locked
RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$LOCKOUT_USER\",\"password\":\"testpass123\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Correct password rejected while locked (423)" "423" "$HTTP_CODE"

# ------------------------------------------------------------------
# Test group 2: Admin unlock → login succeeds
# ------------------------------------------------------------------
echo ""
echo "--- Lockout Release via Admin Unlock ---"

RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/users/$LOCKOUT_USER_ID/unlock")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Admin unlock succeeds (200)" "200" "$HTTP_CODE"

RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$LOCKOUT_USER\",\"password\":\"testpass123\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Login succeeds after admin unlock (200)" "200" "$HTTP_CODE"

# ------------------------------------------------------------------
# Test group 3: Window expiry — old attempts don't count
# Moves existing login_attempts rows back 20 minutes (outside the
# 15-minute window), then verifies login is allowed.
# ------------------------------------------------------------------
echo ""
echo "--- Rolling Window Expiry ---"

EXPIRY_USER="lockexp_$(date +%s)"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/users" \
    -d "{\"username\":\"$EXPIRY_USER\",\"password\":\"testpass123\",\"display_name\":\"Expiry Test\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
EXPIRY_USER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ "$HTTP_CODE" != "201" ] || [ -z "$EXPIRY_USER_ID" ] || [ "$EXPIRY_USER_ID" = "" ]; then
    echo "  SKIP: Could not create expiry test user (HTTP $HTTP_CODE)"
else
    echo "  Created expiry test user: $EXPIRY_USER ($EXPIRY_USER_ID)"

    # Send 4 failed logins (just under threshold)
    for i in 1 2 3 4; do
        curl -s -X POST "$BASE_URL/api/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"username\":\"$EXPIRY_USER\",\"password\":\"wrongpassword\"}" > /dev/null
    done

    # Move those 4 attempts back 20 minutes (outside the 15-min window)
    psql "$DB_URL" -q -c \
        "UPDATE login_attempts SET attempted_at = attempted_at - INTERVAL '20 minutes' WHERE user_id = '$EXPIRY_USER_ID';" 2>/dev/null

    if [ $? -ne 0 ]; then
        echo "  SKIP: Cannot manipulate DB directly (psql not available)"
    else
        # Now send 4 more failures — only these are inside the window,
        # so we should NOT be locked (threshold is 5).
        for i in 1 2 3 4; do
            curl -s -X POST "$BASE_URL/api/auth/login" \
                -H "Content-Type: application/json" \
                -d "{\"username\":\"$EXPIRY_USER\",\"password\":\"wrongpassword\"}" > /dev/null
        done

        # Login with correct password should still succeed (only 4 in-window failures)
        RESPONSE=$(curl -s -w "\n%{http_code}" \
            -X POST "$BASE_URL/api/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"username\":\"$EXPIRY_USER\",\"password\":\"testpass123\"}")
        HTTP_CODE=$(echo "$RESPONSE" | tail -1)
        assert_status "Login succeeds when old attempts outside window (200)" "200" "$HTTP_CODE"
    fi
fi

# ------------------------------------------------------------------
# Test group 4: Successful login clears the window
# After a successful login the failure count resets, so the user
# can sustain another round of failures before locking.
# ------------------------------------------------------------------
echo ""
echo "--- Successful Login Clears Window ---"

CLEAR_USER="lockclear_$(date +%s)"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/users" \
    -d "{\"username\":\"$CLEAR_USER\",\"password\":\"testpass123\",\"display_name\":\"Clear Test\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
CLEAR_USER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ "$HTTP_CODE" != "201" ] || [ -z "$CLEAR_USER_ID" ] || [ "$CLEAR_USER_ID" = "" ]; then
    echo "  SKIP: Could not create clear test user (HTTP $HTTP_CODE)"
else
    echo "  Created clear test user: $CLEAR_USER ($CLEAR_USER_ID)"

    # 4 failed attempts (just under threshold)
    for i in 1 2 3 4; do
        curl -s -X POST "$BASE_URL/api/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"username\":\"$CLEAR_USER\",\"password\":\"wrongpassword\"}" > /dev/null
    done

    # Successful login should clear the window
    RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$CLEAR_USER\",\"password\":\"testpass123\"}")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Login succeeds with 4 failures (under threshold)" "200" "$HTTP_CODE"

    # Now another 4 failures — should NOT trigger lock because window was cleared
    for i in 1 2 3 4; do
        curl -s -X POST "$BASE_URL/api/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"username\":\"$CLEAR_USER\",\"password\":\"wrongpassword\"}" > /dev/null
    done

    RESPONSE=$(curl -s -w "\n%{http_code}" \
        -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$CLEAR_USER\",\"password\":\"testpass123\"}")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Login succeeds after window cleared by prior success (200)" "200" "$HTTP_CODE"
fi

echo ""
echo "=== Rolling-Window Lockout Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
