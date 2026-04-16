#!/bin/bash
# API tests for reviews, notification mark-read, and attachment download:
#   POST  /api/collectibles/:id/reviews
#   PATCH /api/notifications/:id/read
#   GET   /api/messages/:messageId/attachment

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

SELLER_JAR=$(mktemp); BUYER_JAR=$(mktemp)
login_as "seller1" "$SELLER_JAR"; SELLER_CSRF=$(get_csrf "$SELLER_JAR")
login_as "buyer1" "$BUYER_JAR"; BUYER_CSRF=$(get_csrf "$BUYER_JAR")
cleanup() { rm -f "$SELLER_JAR" "$BUYER_JAR"; }
trap cleanup EXIT

echo "=== Reviews, Notifications Read & Attachments Tests ==="

# ===========================================================
# 1. POST /api/collectibles/:id/reviews
# ===========================================================
echo ""
echo "--- Post Review ---"

COLLECTIBLE_ID="10000000-0000-0000-0000-000000000001"

# Buyer posts a review
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/collectibles/$COLLECTIBLE_ID/reviews" \
    -d "{\"collectible_id\":\"$COLLECTIBLE_ID\",\"rating\":5,\"body\":\"Excellent digital dragon!\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Post review returns 201" "201" "$HTTP_CODE"

# Review with invalid rating (0) returns 422
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/collectibles/$COLLECTIBLE_ID/reviews" \
    -d "{\"collectible_id\":\"$COLLECTIBLE_ID\",\"rating\":0,\"body\":\"Bad rating test\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Review with rating 0 rejected (422)" "422" "$HTTP_CODE"

# Review with rating > 5 returns 422
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/collectibles/$COLLECTIBLE_ID/reviews" \
    -d "{\"collectible_id\":\"$COLLECTIBLE_ID\",\"rating\":6,\"body\":\"Too high rating\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Review with rating 6 rejected (422)" "422" "$HTTP_CODE"

# Review without body returns 422
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/collectibles/$COLLECTIBLE_ID/reviews" \
    -d "{\"collectible_id\":\"$COLLECTIBLE_ID\",\"rating\":3}")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Review without body rejected (422)" "422" "$HTTP_CODE"

# Review on nonexistent collectible returns 404
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
    -X POST "$BASE_URL/api/collectibles/00000000-0000-0000-0000-999999999999/reviews" \
    -d '{"collectible_id":"00000000-0000-0000-0000-999999999999","rating":4,"body":"Does not exist"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Review on nonexistent collectible returns 404" "404" "$HTTP_CODE"

# ===========================================================
# 2. PATCH /api/notifications/:id/read
# ===========================================================
echo ""
echo "--- Mark Notification Read ---"

# First trigger a notification by confirming an order
NOTIF_CID=$(curl -s -b "$SELLER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"title":"Notif Read Test","price_cents":1500}' \
    | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

NOTIF_OID=""
if [ -n "$NOTIF_CID" ] && [ "$NOTIF_CID" != "None" ]; then
    IDEM=$(python3 -c "import uuid; print(uuid.uuid4())")
    NOTIF_OID=$(curl -s -b "$BUYER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
        -H "Idempotency-Key: $IDEM" \
        -X POST "$BASE_URL/api/orders" \
        -d "{\"collectible_id\":\"$NOTIF_CID\"}" \
        | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)
fi

if [ -n "$NOTIF_OID" ] && [ "$NOTIF_OID" != "None" ]; then
    # Confirm order to trigger notification
    curl -s -b "$SELLER_JAR" -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
        -X POST "$BASE_URL/api/orders/$NOTIF_OID/confirm" > /dev/null
    sleep 1

    # Get notification ID
    NOTIF_RESP=$(curl -s -b "$BUYER_JAR" -H "X-CSRF-Token: $BUYER_CSRF" \
        "$BASE_URL/api/notifications?unread=true")
    NOTIF_ID=$(echo "$NOTIF_RESP" | python3 -c "
import sys,json
d=json.load(sys.stdin)
data=d.get('data',[]) or []
print(data[0]['id'] if data else '')
" 2>/dev/null)

    if [ -n "$NOTIF_ID" ] && [ "$NOTIF_ID" != "" ]; then
        # Mark notification as read
        RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
            -H "X-CSRF-Token: $BUYER_CSRF" \
            -X PATCH "$BASE_URL/api/notifications/$NOTIF_ID/read")
        HTTP_CODE=$(echo "$RESPONSE" | tail -1)
        assert_status "Mark notification read returns 200" "200" "$HTTP_CODE"

        # Seller cannot mark buyer's notification as read (403)
        RESPONSE=$(curl -s -w "\n%{http_code}" -b "$SELLER_JAR" \
            -H "X-CSRF-Token: $SELLER_CSRF" \
            -X PATCH "$BASE_URL/api/notifications/$NOTIF_ID/read")
        HTTP_CODE=$(echo "$RESPONSE" | tail -1)
        assert_status "Other user cannot mark notification read (403)" "403" "$HTTP_CODE"
    else
        echo "  SKIP: No notifications found for mark-read test"
    fi
else
    echo "  SKIP: Could not create order for notification test"
fi

# Invalid notification ID returns 400
RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
    -H "X-CSRF-Token: $BUYER_CSRF" \
    -X PATCH "$BASE_URL/api/notifications/not-a-uuid/read")
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
assert_status "Invalid notification ID returns 400" "400" "$HTTP_CODE"

# ===========================================================
# 3. GET /api/messages/:messageId/attachment
# ===========================================================
echo ""
echo "--- Attachment Download ---"

# Create an order and send a message with attachment, then download it
ATT_CID=$(curl -s -b "$SELLER_JAR" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $SELLER_CSRF" \
    -X POST "$BASE_URL/api/collectibles" \
    -d '{"title":"Attachment Test Item","price_cents":1000}' \
    | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)

ATT_OID=""
if [ -n "$ATT_CID" ] && [ "$ATT_CID" != "None" ]; then
    IDEM=$(python3 -c "import uuid; print(uuid.uuid4())")
    ATT_OID=$(curl -s -b "$BUYER_JAR" \
        -H "Content-Type: application/json" -H "X-CSRF-Token: $BUYER_CSRF" \
        -H "Idempotency-Key: $IDEM" \
        -X POST "$BASE_URL/api/orders" \
        -d "{\"collectible_id\":\"$ATT_CID\"}" \
        | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)
fi

if [ -n "$ATT_OID" ] && [ "$ATT_OID" != "None" ]; then
    # Send message with text file attachment
    TMPFILE=$(mktemp /tmp/test_attach_XXXXXX.txt)
    echo "This is a clean test file for attachment download." > "$TMPFILE"

    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        -X POST "$BASE_URL/api/orders/$ATT_OID/messages" \
        -F "body=Please see attached shipping label" \
        -F "attachment=@$TMPFILE;type=text/plain")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    MSG_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)
    rm -f "$TMPFILE"

    if [ "$HTTP_CODE" = "201" ] && [ -n "$MSG_ID" ] && [ "$MSG_ID" != "None" ]; then
        # Download the attachment
        RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
            -H "X-CSRF-Token: $BUYER_CSRF" \
            "$BASE_URL/api/messages/$MSG_ID/attachment")
        HTTP_CODE=$(echo "$RESPONSE" | tail -1)
        assert_status "Download attachment returns 200" "200" "$HTTP_CODE"
    else
        echo "  SKIP: Could not send message with attachment (HTTP $HTTP_CODE)"
    fi

    # Non-existent message attachment returns 404
    RESPONSE=$(curl -s -w "\n%{http_code}" -b "$BUYER_JAR" \
        -H "X-CSRF-Token: $BUYER_CSRF" \
        "$BASE_URL/api/messages/00000000-0000-0000-0000-999999999999/attachment")
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    assert_status "Nonexistent message attachment returns 404" "404" "$HTTP_CODE"
else
    echo "  SKIP: Could not create order for attachment test"
fi

echo ""
echo "=== Reviews, Notifications & Attachments Tests Summary: $PASS passed, $FAIL failed ==="
exit $FAIL
