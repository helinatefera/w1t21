#!/bin/bash
# API tests for message endpoints
# Requires: services running, seed data loaded, at least one order existing

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

# Login as buyer
BUYER_JAR=$(mktemp)
curl -s -c "$BUYER_JAR" -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"buyer1","password":"testpass123"}' > /dev/null
BUYER_CSRF=$(grep csrf_token "$BUYER_JAR" | awk '{print $NF}')

# Login as seller
SELLER_JAR=$(mktemp)
curl -s -c "$SELLER_JAR" -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"seller1","password":"testpass123"}' > /dev/null
SELLER_CSRF=$(grep csrf_token "$SELLER_JAR" | awk '{print $NF}')

# Login as analyst (should not be able to send messages)
ANALYST_JAR=$(mktemp)
curl -s -c "$ANALYST_JAR" -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"analyst1","password":"testpass123"}' > /dev/null
ANALYST_CSRF=$(grep csrf_token "$ANALYST_JAR" | awk '{print $NF}')

cleanup() { rm -f "$BUYER_JAR" "$SELLER_JAR" "$ANALYST_JAR"; }
trap cleanup EXIT

echo "=== Messages API Tests ==="

# First create an order to test messages on
COLLECTIBLE_ID="10000000-0000-0000-0000-000000000002"
MSG_IDEM=$(python3 -c "import uuid; print(uuid.uuid4())")
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -H "Idempotency-Key: $MSG_IDEM" \
    -X POST "$BASE_URL/api/orders" \
    -d "{\"collectible_id\":\"$COLLECTIBLE_ID\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
ORDER_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

if [ -z "$ORDER_ID" ] || [ "$ORDER_ID" = "None" ] || [ "$ORDER_ID" = "" ]; then
    # Try to get from existing orders
    RESPONSE=$(curl -s -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        "$BASE_URL/api/orders?role=buyer")
    ORDER_ID=$(echo "$RESPONSE" | python3 -c "import sys,json; d=json.load(sys.stdin)['data']; print(d[0]['id'] if d else '')" 2>/dev/null)
fi

if [ -z "$ORDER_ID" ] || [ "$ORDER_ID" = "None" ] || [ "$ORDER_ID" = "" ]; then
    echo "  SKIP: No order available for message tests"
    echo "=== Messages Tests Summary: 0 passed, 0 failed (skipped) ==="
    exit 0
fi

echo "  Using order: $ORDER_ID"

# Test 1: Buyer sends message
echo ""
echo "--- Send Messages ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/orders/$ORDER_ID/messages" \
    -F "body=Hello, when will this ship?")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Buyer sends message (201)" "201" "$HTTP_CODE"

# Test 2: Seller sends message
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
    -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/orders/$ORDER_ID/messages" \
    -F "body=It will ship tomorrow!")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Seller sends message (201)" "201" "$HTTP_CODE"

# Test 3: List messages
echo ""
echo "--- List Messages ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/orders/$ORDER_ID/messages")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "List messages returns 200" "200" "$HTTP_CODE"

MSG_COUNT=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['total_count'])" 2>/dev/null)
if [ "$MSG_COUNT" -ge 2 ] 2>/dev/null; then
    echo "  PASS: Messages returned (count=$MSG_COUNT)"
    PASS=$((PASS + 1))
else
    echo "  FAIL: Expected at least 2 messages, got $MSG_COUNT"
    FAIL=$((FAIL + 1))
fi

# Test 4: Send empty message
echo ""
echo "--- Message Validation ---"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/orders/$ORDER_ID/messages" \
    -F "body=")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Send empty message returns 422" "422" "$HTTP_CODE"

# Test 5: Analyst cannot send message (not buyer/seller of order)
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$ANALYST_JAR" \
    -H "X-CSRF-Token: $ANALYST_CSRF" \
    -X POST "$BASE_URL/api/orders/$ORDER_ID/messages" \
    -F "body=I should not be able to send this")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Non-party cannot send message (403)" "403" "$HTTP_CODE"

# Test 6: Messages on nonexistent order
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    "$BASE_URL/api/orders/00000000-0000-0000-0000-999999999999/messages")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Messages on nonexistent order returns 404" "404" "$HTTP_CODE"

# Test 7: Text attachment with SSN is blocked
echo ""
echo "--- Attachment PII Scanning ---"
SSN_FILE=$(mktemp /tmp/pii_ssn_XXXXXX.txt)
echo "Customer SSN: 123-45-6789" > "$SSN_FILE"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/orders/$ORDER_ID/messages" \
    -F "body=See attached" \
    -F "attachment=@$SSN_FILE;type=text/plain")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Text attachment with SSN blocked (422)" "422" "$HTTP_CODE"
assert_json_field "SSN attachment error mentions sensitive info" "$BODY" "['error']['message']" "attachment blocked: contains sensitive personal information (SSN)"
rm -f "$SSN_FILE"

# Test 8: Text attachment with phone number is blocked
PHONE_FILE=$(mktemp /tmp/pii_phone_XXXXXX.txt)
echo "Call me at (555) 123-4567" > "$PHONE_FILE"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/orders/$ORDER_ID/messages" \
    -F "body=See attached" \
    -F "attachment=@$PHONE_FILE;type=text/plain")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "Text attachment with phone blocked (422)" "422" "$HTTP_CODE"
assert_json_field "Phone attachment error mentions sensitive info" "$BODY" "['error']['message']" "attachment blocked: contains sensitive personal information (phone number)"
rm -f "$PHONE_FILE"

# Test 9: Text attachment with email address is blocked
EMAIL_FILE=$(mktemp /tmp/pii_email_XXXXXX.csv)
echo "name,email" > "$EMAIL_FILE"
echo "John,john@example.com" >> "$EMAIL_FILE"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/orders/$ORDER_ID/messages" \
    -F "body=See attached" \
    -F "attachment=@$EMAIL_FILE;type=text/csv")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')
assert_status "CSV attachment with email blocked (422)" "422" "$HTTP_CODE"
assert_json_field "Email attachment error mentions sensitive info" "$BODY" "['error']['message']" "attachment blocked: contains sensitive personal information (email address)"
rm -f "$EMAIL_FILE"

# Test 10: Clean text attachment is allowed
CLEAN_FILE=$(mktemp /tmp/clean_XXXXXX.txt)
echo "This is a clean document with no PII" > "$CLEAN_FILE"
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/orders/$ORDER_ID/messages" \
    -F "body=Clean attachment" \
    -F "attachment=@$CLEAN_FILE;type=text/plain")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Clean text attachment allowed (201)" "201" "$HTTP_CODE"
rm -f "$CLEAN_FILE"

echo ""
echo "=== Messages Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
