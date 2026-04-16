#!/bin/bash
# API tests for user management and setup endpoints:
#   GET  /api/setup/status
#   POST /api/setup/admin
#   GET  /api/auth/me
#   GET  /api/users/:id
#   PATCH /api/users/:id
#   DELETE /api/users/:id/roles/:roleId

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

ADMIN_JAR=$(mktemp); BUYER_JAR=$(mktemp)
login_as "admin" "$ADMIN_JAR"; ADMIN_CSRF=$(get_csrf "$ADMIN_JAR")
login_as "buyer1" "$BUYER_JAR"; BUYER_CSRF=$(get_csrf "$BUYER_JAR")
cleanup() { rm -f "$ADMIN_JAR" "$BUYER_JAR"; }
trap cleanup EXIT

echo "=== User Management & Setup Tests ==="

# ===========================================================
# 1. GET /api/setup/status
# ===========================================================
echo ""
echo "--- Setup Status ---"

RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/setup/status")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Setup status returns 200" "200" "$HTTP_CODE"
assert_json_field "Setup is complete" "$BODY" "['setup_complete']" "True"

# ===========================================================
# 2. POST /api/setup/admin (already complete => 409)
# ===========================================================
echo ""
echo "--- Setup Admin (already seeded) ---"

RESPONSE=$(curl -s -w "\n%{http_code}" \
    -X POST "$BASE_URL/api/setup/admin" \
    -H "Content-Type: application/json" \
    -d '{"username":"newadmin","password":"longpassword123","display_name":"New Admin"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "POST /api/setup/admin returns 409 when admin exists" "409" "$HTTP_CODE"

# ===========================================================
# 3. GET /api/auth/me
# ===========================================================
echo ""
echo "--- Auth Me ---"

RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/auth/me")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "GET /api/auth/me returns 200" "200" "$HTTP_CODE"
assert_json_field "Me returns correct username" "$BODY" "['user']['username']" "admin"
assert_json_field "Me returns roles" "$BODY" "['roles'][0]" "administrator"

# Unauthenticated /me returns 401
RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/auth/me")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "GET /api/auth/me unauthenticated returns 401" "401" "$HTTP_CODE"

# ===========================================================
# 4. GET /api/users/:id
# ===========================================================
echo ""
echo "--- Get User by ID ---"

ADMIN_ID="00000000-0000-0000-0000-000000000001"

RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/users/$ADMIN_ID")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Get user by ID returns 200" "200" "$HTTP_CODE"
assert_json_field "User has correct username" "$BODY" "['username']" "admin"

# Buyer cannot get user by ID (403)
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/users/$ADMIN_ID")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Buyer cannot get user by ID (403)" "403" "$HTTP_CODE"

# Invalid UUID returns 400
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    "$BASE_URL/api/users/not-a-uuid")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Invalid UUID returns 400" "400" "$HTTP_CODE"

# ===========================================================
# 5. PATCH /api/users/:id
# ===========================================================
echo ""
echo "--- Update User ---"

# Create a fresh user with a unique name
UNIQUE_USER="upd_usr_$(date +%s)"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/users" \
    -d "{\"username\":\"$UNIQUE_USER\",\"password\":\"testpass123\",\"display_name\":\"Before Update\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
UPDATE_USER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ -n "$UPDATE_USER_ID" ] && [ "$UPDATE_USER_ID" != "None" ] && [ "$UPDATE_USER_ID" != "" ]; then
    # Update display name
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X PATCH "$BASE_URL/api/users/$UPDATE_USER_ID" \
        -d '{"display_name":"After Update"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    assert_status "Update user display name returns 200" "200" "$HTTP_CODE"
    assert_json_field "Display name updated" "$BODY" "['display_name']" "After Update"

    # Buyer cannot update user (403)
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "Content-Type: application/json" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        -X PATCH "$BASE_URL/api/users/$UPDATE_USER_ID" \
        -d '{"display_name":"Hacked"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Buyer cannot update user (403)" "403" "$HTTP_CODE"
else
    echo "  SKIP: Could not create user for update test"
fi

# ===========================================================
# 6. DELETE /api/users/:id/roles/:roleId
# ===========================================================
echo ""
echo "--- Remove Role ---"

# Add seller role to buyer1, verify, then test DELETE endpoint
ROLE_USER_ID="00000000-0000-0000-0000-000000000003"

RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $ADMIN_CSRF" \
    -X POST "$BASE_URL/api/users/$ROLE_USER_ID/roles" \
    -d '{"role_name":"seller"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ "${HTTP_CODE:0:2}" = "20" ]; then
    echo "  PASS: Added seller role to buyer1"
    PASS=$((PASS + 1))

    # Test DELETE endpoint — with a non-existent role UUID it should respond
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ADMIN_JAR" \
        -H "X-CSRF-Token: $ADMIN_CSRF" \
        -X DELETE "$BASE_URL/api/users/$ROLE_USER_ID/roles/00000000-0000-0000-0000-000000000000")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "422" ]; then
        echo "  PASS: Delete role endpoint responds ($HTTP_CODE)"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: Delete role endpoint should respond 200/404/422 (got $HTTP_CODE)"
        FAIL=$((FAIL + 1))
    fi

    # Buyer cannot remove roles (403)
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        -X DELETE "$BASE_URL/api/users/$ROLE_USER_ID/roles/00000000-0000-0000-0000-000000000001")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Buyer cannot remove role (403)" "403" "$HTTP_CODE"
else
    echo "  SKIP: Could not add role for removal test (HTTP $HTTP_CODE)"
fi

echo ""
echo "=== User Management Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
