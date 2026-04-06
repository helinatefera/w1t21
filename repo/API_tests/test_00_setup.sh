#!/bin/bash
# Pre-flight check: verifies API is reachable and seed data is present.
# This script runs first (sorted alphabetically) so the suite fails
# immediately with a clear message instead of producing confusing
# 401/403 noise when seed users or collectibles are missing.

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

echo "=== Pre-flight Setup Checks ==="

# ---- 1. API reachability ----
echo ""
echo "--- API Reachability ---"
RESPONSE=$(curl -s -w "\n%{http_code}" --max-time 5 \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"__probe__","password":"probeprobe"}' 2>/dev/null)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)

if [ -z "$HTTP_CODE" ] || [ "$HTTP_CODE" = "000" ]; then
    echo "  FAIL: API is not reachable at $BASE_URL"
    echo ""
    echo "  Ensure services are running:"
    echo "    docker compose up -d"
    echo ""
    FAIL=$((FAIL + 1))
    echo "=== Setup Checks Summary: $PASS passed, $FAIL failed ==="
    exit $FAIL
else
    echo "  PASS: API is reachable at $BASE_URL"
    PASS=$((PASS + 1))
fi

# ---- 2. Seed users exist ----
echo ""
echo "--- Seed Users ---"

check_user() {
    local username="$1" role="$2"
    local resp http_code body

    resp=$(curl -s -w "\n%{http_code}" \
        -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$username\",\"password\":\"testpass123\"}")
    http_code=$(echo "$resp" | tail -1)
    body=$(echo "$resp" | sed '$d')

    if [ "$http_code" = "200" ]; then
        echo "  PASS: User '$username' exists and password is correct"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: User '$username' missing or wrong password (HTTP $http_code)"
        echo "        Seed data is required. Run:"
        echo "          docker compose exec postgres psql -U ledgermint -d ledgermint -f /dev/stdin < scripts/seed.sql"
        FAIL=$((FAIL + 1))
    fi
}

check_user "admin"    "administrator"
check_user "seller1"  "seller"
check_user "buyer1"   "buyer"
check_user "analyst1" "compliance_analyst"

# ---- 3. Seed collectibles exist ----
echo ""
echo "--- Seed Collectibles ---"

# Log in as buyer to query collectibles (any authenticated user can list)
COOKIE_JAR=$(mktemp)
curl -s -c "$COOKIE_JAR" \
    -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"buyer1","password":"testpass123"}' > /dev/null 2>&1

CSRF_TOKEN=$(grep csrf_token "$COOKIE_JAR" 2>/dev/null | awk '{print $NF}')

RESPONSE=$(curl -s -w "\n%{http_code}" -b "$COOKIE_JAR" \
    -H "X-CSRF-Token: $CSRF_TOKEN" \
    "$BASE_URL/api/collectibles" 2>/dev/null)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    TOTAL=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('total_count',0))" 2>/dev/null)
    if [ "$TOTAL" -ge 3 ] 2>/dev/null; then
        echo "  PASS: Seed collectibles present (count=$TOTAL)"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: Expected >= 3 seed collectibles, found $TOTAL"
        echo "        Run: docker compose exec postgres psql -U ledgermint -d ledgermint -f /dev/stdin < scripts/seed.sql"
        FAIL=$((FAIL + 1))
    fi
else
    echo "  FAIL: Could not list collectibles (HTTP $HTTP_CODE) — seed check skipped"
    FAIL=$((FAIL + 1))
fi

# Check specific collectible IDs used by other tests
for CID in "10000000-0000-0000-0000-000000000001" \
            "10000000-0000-0000-0000-000000000002" \
            "10000000-0000-0000-0000-000000000003"; do
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$COOKIE_JAR" \
        -H "X-CSRF-Token: $CSRF_TOKEN" \
        "$BASE_URL/api/collectibles/$CID" 2>/dev/null)
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    if [ "$HTTP_CODE" = "200" ]; then
        TITLE=$(echo "$RESPONSE" | sed '$d' | python3 -c "import sys,json; print(json.load(sys.stdin)['collectible']['title'])" 2>/dev/null)
        echo "  PASS: Collectible $CID exists ($TITLE)"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: Collectible $CID not found (HTTP $HTTP_CODE)"
        FAIL=$((FAIL + 1))
    fi
done

rm -f "$COOKIE_JAR"

# ---- Summary ----
echo ""
if [ "$FAIL" -gt 0 ]; then
    echo "  *** SETUP FAILED — remaining API tests will likely produce false failures ***"
    echo "  Fix the issues above and re-run."
fi
echo "=== Setup Checks Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
