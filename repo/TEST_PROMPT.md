# LedgerMint - Test Cases and Verification Steps

This document contains all test cases and verification steps for validating the LedgerMint Digital Collectibles Exchange platform.

---

## 1. Infrastructure Verification

### 1.1 Docker Compose Startup
- [ ] Run `docker compose up` from the repository root
- [ ] Verify all three services start: `postgres`, `backend`, `frontend`
- [ ] Verify PostgreSQL health check passes (backend depends on it)
- [ ] Verify backend starts and applies migrations (check logs for "applied migration")
- [ ] Verify frontend Nginx serves the app

### 1.2 Service Accessibility
| Check | URL | Expected |
|-------|-----|----------|
| Frontend loads | http://localhost | Login page HTML with "LedgerMint" title |
| API responds | http://localhost:8080/api/auth/login | JSON error (method/body validation) |
| API via Nginx proxy | http://localhost/api/auth/login | Same JSON error via Nginx proxy |
| PostgreSQL accessible | `docker compose exec postgres psql -U ledgermint -c "SELECT 1"` | Returns 1 |

### 1.3 Seed Data
- [ ] Run: `docker compose exec postgres psql -U ledgermint -d ledgermint -f /dev/stdin < scripts/seed.sql`
- [ ] Verify 4 users exist: `admin`, `seller1`, `buyer1`, `analyst1`
- [ ] Verify 3 collectibles exist with correct titles

---

## 2. Authentication Tests

### 2.1 Login - Valid Credentials
| User | Password | Expected Status | Expected Roles |
|------|----------|----------------|----------------|
| admin | testpass123 | 200 | administrator |
| seller1 | testpass123 | 200 | seller |
| buyer1 | testpass123 | 200 | buyer |
| analyst1 | testpass123 | 200 | compliance_analyst |

**Verify for each login:**
- Response contains `user.username` and `roles` array
- `access_token` cookie is set (HttpOnly)
- `refresh_token` cookie is set (HttpOnly, path=/api/auth)
- `csrf_token` cookie is set (NOT HttpOnly, readable by JavaScript)

### 2.2 Login - Invalid Credentials
| Input | Expected Status | Expected Error Code |
|-------|----------------|-------------------|
| Wrong password | 401 | ERR_INVALID_CREDENTIALS |
| Nonexistent username | 401 | ERR_INVALID_CREDENTIALS |
| Username < 3 chars | 422 | ERR_VALIDATION |
| Password < 8 chars | 422 | ERR_VALIDATION |
| Missing password field | 422 | ERR_VALIDATION |
| Empty body | 400 | ERR_VALIDATION |

### 2.3 Account Lockout
- [ ] Send 5 failed login attempts for the same user
- [ ] Verify 6th attempt returns HTTP 423 (ERR_ACCOUNT_LOCKED)
- [ ] Verify account auto-unlocks after 30 minutes (or admin unlock)

### 2.4 Token Refresh
- [ ] After login, POST to `/api/auth/refresh` with refresh_token cookie
- [ ] Verify new access_token and refresh_token cookies are issued
- [ ] Verify refresh without cookie returns 401

### 2.5 Logout
- [ ] POST to `/api/auth/logout` with valid auth
- [ ] Verify cookies are cleared (MaxAge=-1)
- [ ] Verify subsequent requests with old tokens return 401

### 2.6 CSRF Protection
- [ ] POST to a protected endpoint without X-CSRF-Token header
- [ ] Verify request is rejected (403)
- [ ] Verify GET requests are not affected by CSRF

---

## 3. User Management Tests (Administrator Only)

### 3.1 Create User
- [ ] POST `/api/users` as admin with valid data -> 201
- [ ] Verify response contains user ID, username, display_name
- [ ] POST `/api/users` as buyer -> 403 (permission denied)
- [ ] POST `/api/users` with duplicate username -> 409 (conflict)
- [ ] POST `/api/users` with missing required fields -> 422

### 3.2 List Users
- [ ] GET `/api/users` as admin -> 200, paginated response
- [ ] Verify `data`, `page`, `page_size`, `total_count`, `total_pages` fields
- [ ] GET `/api/users` as buyer -> 403

### 3.3 Role Management
- [ ] POST `/api/users/:id/roles` with `{"role_name":"buyer"}` -> 200/201
- [ ] Verify invalid role name returns 422
- [ ] DELETE `/api/users/:id/roles/:roleId` -> 200

### 3.4 Account Unlock
- [ ] POST `/api/users/:id/unlock` as admin -> 200
- [ ] Verify locked user can log in again after unlock

---

## 4. Collectible Tests

### 4.1 List Collectibles
- [ ] GET `/api/collectibles` -> 200, paginated
- [ ] Default status filter is "published"
- [ ] Query param `?status=hidden` returns hidden items (admin)
- [ ] Unauthenticated request -> 401

### 4.2 Get Collectible
- [ ] GET `/api/collectibles/:id` -> 200
- [ ] Response contains `collectible` and `transaction_history`
- [ ] Invalid UUID -> 400
- [ ] Nonexistent ID -> 404

### 4.3 Create Collectible (Seller Only)
- [ ] POST `/api/collectibles` as seller with valid data -> 201
- [ ] Verify auto-generated token_id and chain_id
- [ ] POST as buyer -> 403
- [ ] Missing title -> 422
- [ ] Price cents = 0 -> 422
- [ ] Price cents negative -> 422

### 4.4 Update Collectible (Seller Only)
- [ ] PATCH `/api/collectibles/:id` as owner seller -> 200
- [ ] PATCH as different seller -> 403 (forbidden)
- [ ] Verify only provided fields are updated

### 4.5 Moderation (Administrator Only)
- [ ] PATCH `/api/collectibles/:id/hide` with reason -> 200
- [ ] Verify collectible no longer appears in published list
- [ ] PATCH `/api/collectibles/:id/publish` -> 200
- [ ] Verify collectible reappears in published list
- [ ] Hide without reason -> 422
- [ ] Hide as seller -> 403

### 4.6 Seller's Own Listings
- [ ] GET `/api/collectibles/mine` as seller -> 200
- [ ] GET `/api/collectibles/mine` as buyer -> 403

---

## 5. Order Tests

### 5.1 Create Order (Buyer Only)
- [ ] POST `/api/orders` with valid collectible_id and Idempotency-Key -> 201
- [ ] Verify order status = "pending"
- [ ] Verify price_snapshot_cents matches collectible price
- [ ] POST without Idempotency-Key -> 400
- [ ] POST as seller -> 403
- [ ] POST for own collectible (seller also has buyer role) -> 422

### 5.2 Idempotency
- [ ] Send same request with same Idempotency-Key twice
- [ ] Verify same order ID returned both times (no duplicate created)

### 5.3 Overselling Prevention
- [ ] Create order for a collectible that already has an active order
- [ ] Verify 409 ERR_OVERSOLD response

### 5.4 Order State Transitions

**Happy path:**
| From | To | Actor | Endpoint | Expected |
|------|-----|-------|----------|----------|
| pending | confirmed | seller | POST /orders/:id/confirm | 200 |
| confirmed | processing | seller | POST /orders/:id/process | 200 |
| processing | completed | seller | POST /orders/:id/complete | 200 |

**Cancellation:**
| From | Actor | Endpoint | Expected |
|------|-------|----------|----------|
| pending | buyer or seller | POST /orders/:id/cancel | 200 |
| confirmed | buyer or seller | POST /orders/:id/cancel | 200 |

**Invalid transitions:**
| From | To | Expected |
|------|-----|----------|
| pending | processing | 422 ERR_INVALID_TRANSITION |
| pending | completed | 422 ERR_INVALID_TRANSITION |
| processing | cancelled | 422 ERR_INVALID_TRANSITION |
| completed | any | 422 ERR_INVALID_TRANSITION |
| cancelled | any | 422 ERR_INVALID_TRANSITION |

**Permission errors:**
- [ ] Buyer cannot confirm/process/complete orders (403)
- [ ] Cancel requires reason field (422 without it)

### 5.5 Fulfillment Tracking
- [ ] PATCH `/api/orders/:id/fulfillment` as seller with carrier/tracking_number -> 200
- [ ] Verify fulfillment_tracking JSON is stored on order

### 5.6 List Orders
- [ ] GET `/api/orders?role=buyer` -> buyer's orders
- [ ] GET `/api/orders?role=seller` -> seller's orders
- [ ] Pagination works correctly

---

## 6. Message Tests

### 6.1 Send Message
- [ ] POST `/api/orders/:orderId/messages` as buyer with body -> 201
- [ ] POST as seller of that order -> 201
- [ ] POST as unrelated user -> 403
- [ ] POST with empty body -> 422

### 6.2 List Messages
- [ ] GET `/api/orders/:orderId/messages` -> 200, paginated
- [ ] Only buyer and seller of the order can list

### 6.3 Attachment
- [ ] Send message with file attachment < 10MB -> 201
- [ ] Send message with file > 10MB -> 413

### 6.4 PII Detection (Frontend)
- [ ] Enter SSN pattern (123-45-6789) in message box -> blocked with warning
- [ ] Enter phone number -> blocked with warning
- [ ] Normal text -> no warning, message sends

---

## 7. Notification Tests

### 7.1 List Notifications
- [ ] GET `/api/notifications` -> 200, paginated
- [ ] `?unread=true` filters to unread only

### 7.2 Mark Read
- [ ] PATCH `/api/notifications/:id/read` -> 200
- [ ] POST `/api/notifications/read-all` -> 200
- [ ] Verify unread count updates

### 7.3 Retry Failed Notification
- [ ] POST `/api/notifications/:id/retry` -> 200

### 7.4 Notification Preferences
- [ ] GET `/api/notifications/preferences` -> 200
- [ ] PUT with `{"preferences":{"order_confirmed":false}}` -> 200
- [ ] Verify preference persists on next GET

### 7.5 Order Status Notifications
- [ ] Confirm an order -> buyer receives notification with template "order_confirmed"
- [ ] Complete an order -> buyer receives notification with template "order_completed"
- [ ] Cancel an order -> buyer receives notification with template "order_cancelled"

---

## 8. Analytics Tests (Administrator + Compliance Analyst)

### 8.1 Funnel Analytics
- [ ] GET `/api/analytics/funnel?days=7` as admin -> 200
- [ ] Response contains `views`, `orders`, `rate`, `days`
- [ ] GET as buyer -> 403

### 8.2 Retention Cohorts
- [ ] GET `/api/analytics/retention?days=30` -> 200
- [ ] Response is array of `{cohort_date, cohort_size, retained_count, retention_rate}`

### 8.3 Content Performance
- [ ] GET `/api/analytics/content-performance?limit=10` -> 200
- [ ] Response contains collectible-level view/order/conversion data

### 8.4 Compliance Analyst Access
- [ ] Login as `analyst1` (compliance_analyst role)
- [ ] Verify can access `/api/analytics/funnel` -> 200
- [ ] Verify can access `/api/admin/anomalies` -> 200
- [ ] Verify CANNOT access `/api/ab-tests` -> 403
- [ ] Verify CANNOT access `/api/users` -> 403

---

## 9. A/B Test Management (Administrator Only)

### 9.1 CRUD Operations
- [ ] POST `/api/ab-tests` with valid data -> 201
- [ ] GET `/api/ab-tests` -> 200, array of tests
- [ ] GET `/api/ab-tests/:id` -> 200, test with results
- [ ] POST `/api/ab-tests/:id/rollback` for running test -> 200

### 9.2 Assignments
- [ ] GET `/api/ab-tests/assignments` as any authenticated user -> 200
- [ ] Verify deterministic variant assignment (same user gets same variant)

### 9.3 Permissions
- [ ] All A/B test management endpoints require administrator role
- [ ] Buyer/seller cannot create or list tests (403)

---

## 10. Admin Operations

### 10.1 IP Rules
- [ ] GET `/api/admin/ip-rules` as admin -> 200
- [ ] POST with `{"cidr":"10.0.0.0/8","action":"deny"}` -> 201
- [ ] DELETE `/api/admin/ip-rules/:id` -> 200
- [ ] Invalid CIDR format -> 422
- [ ] Invalid action -> 422

### 10.2 Anomaly Alerts
- [ ] GET `/api/admin/anomalies` -> 200, paginated
- [ ] PATCH `/api/admin/anomalies/:id/acknowledge` -> 200

### 10.3 Metrics
- [ ] GET `/api/admin/metrics` -> 200
- [ ] Response contains `active_users` and `orders_by_status`

---

## 11. Error Response Format

All API errors must follow this structure:
```json
{
  "error": {
    "code": "ERR_<TYPE>",
    "message": "Human-readable message",
    "request_id": "uuid"
  }
}
```

### Verify for each error type:
| HTTP Status | Error Code | Trigger |
|-------------|-----------|---------|
| 400 | ERR_VALIDATION | Invalid request body |
| 401 | ERR_UNAUTHORIZED | No auth cookie |
| 401 | ERR_INVALID_CREDENTIALS | Wrong password |
| 403 | ERR_FORBIDDEN | Insufficient role |
| 404 | ERR_NOT_FOUND | Resource doesn't exist |
| 409 | ERR_CONFLICT | Duplicate resource |
| 409 | ERR_OVERSOLD | Collectible has active order |
| 413 | ERR_ATTACHMENT_TOO_LARGE | Attachment > 10MB |
| 422 | ERR_VALIDATION | Failed validation |
| 422 | ERR_INVALID_TRANSITION | Invalid state change |
| 423 | ERR_ACCOUNT_LOCKED | Account locked |
| 429 | ERR_RATE_LIMITED | Too many requests |

---

## 12. Frontend UI Verification

### 12.1 Login Page
- [ ] Open http://localhost
- [ ] Redirected to /login if not authenticated
- [ ] "LedgerMint" branding visible
- [ ] Username and password fields with validation messages
- [ ] Submit button shows "Signing in..." while loading
- [ ] Invalid credentials show error message in red

### 12.2 Dashboard
- [ ] After login, Dashboard shows welcome message with user's display name
- [ ] Stats cards: Open Orders, Unread Notifications, My Listings (seller) or Roles
- [ ] Recent collectibles grid
- [ ] Links to orders and notifications work

### 12.3 Catalog
- [ ] Grid of collectible cards with title, description, price, view count
- [ ] Click on card navigates to detail page
- [ ] Pagination controls when > 20 items
- [ ] Seller sees "+ Add Collectible" button

### 12.4 Collectible Detail
- [ ] Full detail view with image placeholder, price, description
- [ ] NFT metadata (contract address, chain ID, token ID) if present
- [ ] Buyer sees "Place Order" button
- [ ] Owner sees "Edit Listing" button
- [ ] Transaction history table (immutable)

### 12.5 Orders
- [ ] List view with status badges (colored by status)
- [ ] Buyer/Seller toggle when user has both roles
- [ ] Click navigates to order detail

### 12.6 Order Detail
- [ ] Status badge, price, creation date
- [ ] Action buttons based on role and current status
- [ ] Cancel with reason dialog
- [ ] Fulfillment tracking display
- [ ] Link to messages

### 12.7 Messages
- [ ] Chat-style layout with sent/received alignment
- [ ] Message input with Enter to send
- [ ] File attachment button with size validation
- [ ] PII detection warning blocks send
- [ ] Auto-scroll to latest message

### 12.8 Notifications
- [ ] List with unread highlighting (blue left border)
- [ ] "Mark read" button per notification
- [ ] "Mark All Read" button
- [ ] Filter toggle (All / Unread)
- [ ] Failed notifications show "Retry" button
- [ ] Unread badge in header

### 12.9 Notification Preferences
- [ ] Toggle switches for each notification type
- [ ] Changes persist after page reload

### 12.10 Analytics (Admin)
- [ ] Funnel bar chart (Views -> Orders)
- [ ] Retention line chart
- [ ] Content performance table
- [ ] Date range selector (7/30 days)

### 12.11 A/B Tests (Admin)
- [ ] List with status badges
- [ ] Create form with all fields
- [ ] Rollback button for running tests

### 12.12 User Management (Admin)
- [ ] User table with username, display name, status, created date
- [ ] Create user form
- [ ] Role dropdown per user
- [ ] Unlock button for locked users

### 12.13 Moderation (Admin/Compliance)
- [ ] List all collectibles with status filter
- [ ] Hide button with reason input
- [ ] Publish button for hidden items

### 12.14 Anomaly Alerts
- [ ] List with red left border for unacknowledged
- [ ] Anomaly type label
- [ ] Details display
- [ ] Acknowledge button

### 12.15 Sidebar Navigation
- [ ] Navigation items visible based on user roles
- [ ] Active page highlighted
- [ ] LedgerMint branding at top

---

## 13. Running Automated Tests

```bash
# Run all tests
./run_tests.sh

# Run only unit tests (no services needed)
./run_tests.sh unit

# Run only API tests (services must be running)
./run_tests.sh api
```

### Expected Results
- **Unit tests**: All tests in `unit_tests/` pass (state machine, PII detection, validation, formatters, error codes)
- **API tests**: All tests in `API_tests/` pass (auth, collectibles, orders, messages, notifications, admin)
- **Summary**: Clear PASS/FAIL counts printed at the end
