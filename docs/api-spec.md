# LedgerMint API Specification (Implementation-Aligned)

## Runtime Reality (Important)

- This document is aligned to the current Go/Echo backend implementation under `repo/backend`.
- Routes, request validation, and response behavior are derived from router + handlers + DTOs.
- If this document and code diverge, code is the source of truth.

## Source of Truth Used

- Route registration: `repo/backend/internal/router/router.go`
- Endpoint behavior: `repo/backend/internal/handler/*.go`
- Request/response contracts: `repo/backend/internal/dto/*.go`
- Middleware/security semantics: `repo/backend/internal/middleware/*.go`
- Core model fields returned from handlers: `repo/backend/internal/model/*.go`

## Base URL and Conventions

- Base path: `/api`
- Content type: JSON by default, except multipart message send and binary attachment download.
- Authentication transport: HttpOnly cookies (`access_token`, `refresh_token`) plus readable CSRF cookie (`csrf_token`).
- All non-safe methods (`POST`, `PATCH`, `PUT`, `DELETE`) require header `X-CSRF-Token` matching cookie `csrf_token`, except:
	- `POST /api/auth/login`
	- `POST /api/setup/admin`
- Pagination (where supported):
	- Query: `page` (default 1), `page_size` (default 20, max 100)
	- Response envelope:

```json
{
	"data": [],
	"page": 1,
	"page_size": 20,
	"total_count": 0,
	"total_pages": 0
}
```

## Global API State Gate

Before initial admin bootstrap is complete, most API routes are blocked by setup middleware:

- Error status: `503 Service Unavailable`
- Error code: `ERR_SETUP_REQUIRED`
- Message: `Initial setup required. Create an administrator account via POST /api/setup/admin before using the API.`

Setup endpoints available pre-bootstrap:

- `GET /api/setup/status`
- `POST /api/setup/admin`

## Error Envelope

All structured errors use:

```json
{
	"error": {
		"code": "ERR_...",
		"message": "human-readable message",
		"request_id": "uuid"
	}
}
```

## Error Codes

- `ERR_NOT_FOUND`
- `ERR_FORBIDDEN`
- `ERR_UNAUTHORIZED`
- `ERR_CONFLICT`
- `ERR_RATE_LIMITED`
- `ERR_VALIDATION`
- `ERR_ACCOUNT_LOCKED`
- `ERR_INVALID_CREDENTIALS`
- `ERR_ATTACHMENT_TOO_LARGE`
- `ERR_ORDER_DUPLICATE`
- `ERR_INVALID_TRANSITION`
- `ERR_OVERSOLD`
- `ERR_INTERNAL`
- `ERR_SETUP_REQUIRED`

## Authentication and Security Model

### Cookies Set on Login/Refresh

- `access_token`
	- HttpOnly, SameSite=Strict, Path=`/`, MaxAge 900s
- `refresh_token`
	- HttpOnly, SameSite=Strict, Path=`/api/auth`, MaxAge 7d
- `csrf_token`
	- Readable by JS (HttpOnly=false), SameSite=Strict, Path=`/`, MaxAge 7d

### Role Guards

Role names used by route guards:

- `buyer`
- `seller`
- `administrator`
- `compliance_analyst`

### Rate Limits

- Login: per IP
- Orders create: per user + per IP
- Messages send: per user + per IP
- Collectible create: per user + per IP

## Canonical DTOs

### Key Request DTOs

```ts
type BootstrapAdminRequest = {
	username: string;      // min 3, max 100
	password: string;      // min 12, max 128
	display_name: string;  // max 200
}

type LoginRequest = {
	username: string; // min 3, max 100
	password: string; // min 8
}

type CreateOrderRequest = {
	collectible_id: string; // UUID
}

type UpdateNotificationPrefsRequest = {
	preferences: Record<string, boolean>;
	subscription_mode?: 'status_only' | 'all_events';
}
```

### Key Response DTOs

```ts
type AuthResponse = {
	user: UserResponse;
	roles: string[];
}

type UserResponse = {
	id: string;
	username: string;
	display_name: string;
	email?: string;
	is_locked: boolean;
	created_at: string;
	updated_at: string;
}

type DashboardResponse = {
	owned_collectibles: number;
	open_orders: number;
	unread_notifications: number;
	seller_open_orders: number;
	listed_items: number;
}
```

## Endpoint Catalog

---

## Setup

### `GET /api/setup/status`

- Auth: none
- CSRF: not required (GET)
- Response `200`:

```json
{ "setup_complete": true }
```

### `POST /api/setup/admin`

- Auth: none
- CSRF: exempt
- Body: `BootstrapAdminRequest`
- Behavior: creates first admin only if no admin exists
- Responses:
	- `201`:

```json
{
	"status": "ok",
	"user_id": "uuid",
	"message": "Administrator account created. You may now log in."
}
```

	- `409` if setup already complete

---

## Auth

### `POST /api/auth/login`

- Auth: none (setup must be complete)
- CSRF: exempt
- Rate limit: login limiter
- Body: `LoginRequest`
- Response `200`: `AuthResponse` + auth cookies set

### `POST /api/auth/refresh`

- Auth: none (requires `refresh_token` cookie)
- CSRF: required
- Response `200`:

```json
{ "status": "ok" }
```

- Also rotates tokens and sets fresh cookies

### `GET /api/auth/me`

- Auth: required (JWT cookie)
- CSRF: not required (GET)
- Response `200`: `AuthResponse`

### `POST /api/auth/logout`

- Auth: required
- CSRF: required
- Response `200`:

```json
{ "status": "ok" }
```

---

## Dashboard

### `GET /api/dashboard`

- Auth: required
- Response `200`: `DashboardResponse`

---

## Users (Administrator Only)

### `POST /api/users`

- Auth: administrator
- Body:

```json
{
	"username": "string",
	"password": "string",
	"display_name": "string",
	"email": "optional@email"
}
```

- Response `201`: `UserResponse`

### `GET /api/users`

- Auth: administrator
- Query: `page`, `page_size`
- Response `200`: paginated `UserResponse[]`

### `GET /api/users/:id`

- Auth: administrator
- Response `200`: `UserResponse` (includes masked `email` when present)

### `PATCH /api/users/:id`

- Auth: administrator
- Body (all optional):
	- `display_name`
	- `password`
	- `email`
- Response `200`: `UserResponse`

### `POST /api/users/:id/roles`

- Auth: administrator
- Body:

```json
{ "role_name": "buyer|seller|administrator|compliance_analyst" }
```

- Response `200`:

```json
{ "status": "ok" }
```

### `DELETE /api/users/:id/roles/:roleId`

- Auth: administrator
- Response `200`:

```json
{ "status": "ok" }
```

### `POST /api/users/:id/unlock`

- Auth: administrator
- Response `200`:

```json
{ "status": "ok" }
```

---

## Collectibles

### `GET /api/collectibles`

- Auth: required
- Query:
	- `status` (default `published`)
	- `page`, `page_size`
- Response `200`: paginated `Collectible[]`

### `GET /api/collectibles/mine`

- Auth: seller
- Query: `page`, `page_size`
- Response `200`: paginated `Collectible[]`

### `GET /api/collectibles/:id`

- Auth: required
- Response `200`:

```json
{
	"collectible": { "id": "...", "title": "..." },
	"transaction_history": []
}
```

### `POST /api/collectibles`

- Auth: seller
- Rate limit: listing per-user + per-IP
- Body:

```json
{
	"title": "string",
	"description": "string",
	"contract_address": "optional",
	"chain_id": 1,
	"token_id": "optional",
	"metadata_uri": "optional URL",
	"image_url": "optional URL",
	"price_cents": 1000,
	"currency": "USD"
}
```

- Response `201`: `Collectible`

### `PATCH /api/collectibles/:id`

- Auth: seller (owner-enforced)
- Body: partial update fields
- Response `200`: `Collectible`

### `POST /api/collectibles/:id/reviews`

- Auth: required
- Body:

```json
{
	"collectible_id": "uuid",
	"rating": 1,
	"body": "review text"
}
```

- Response `201`:

```json
{ "status": "ok" }
```

### `PATCH /api/collectibles/:id/hide`

- Auth: administrator
- Body:

```json
{ "reason": "string" }
```

- Response `200`: `{ "status": "ok" }`

### `PATCH /api/collectibles/:id/publish`

- Auth: administrator
- Response `200`: `{ "status": "ok" }`

---

## Orders

### `POST /api/orders`

- Auth: buyer
- Rate limit: per-user + per-IP
- Required header: `Idempotency-Key`
- Body: `CreateOrderRequest`
- Response `201`: `Order`

### `GET /api/orders`

- Auth: required
- Query:
	- `role=seller|buyer` (default buyer path)
	- `page`, `page_size`
- Response `200`: paginated `Order[]`

### `GET /api/orders/:id`

- Auth: required (buyer/seller participant enforced)
- Response `200`: `Order`

### `POST /api/orders/:id/confirm`

- Auth: seller
- Response `200`: `Order`

### `POST /api/orders/:id/process`

- Auth: seller
- Response `200`: `Order`

### `POST /api/orders/:id/complete`

- Auth: seller
- Response `200`: `Order`

### `POST /api/orders/:id/cancel`

- Auth: required (participant enforcement in service)
- Body:

```json
{ "reason": "string" }
```

- Response `200`: `Order`

### `POST /api/orders/:id/refund`

- Auth: seller
- Body:

```json
{ "reason": "string" }
```

- Response `200`: `Order`

### `POST /api/orders/:id/arbitration`

- Auth: required (buyer/seller participant enforced)
- Body:

```json
{ "reason": "string" }
```

- Response `200`: `Order`

### `PATCH /api/orders/:id/fulfillment`

- Auth: seller
- Body:

```json
{
	"carrier": "optional",
	"tracking_number": "optional"
}
```

- Response `200`: `{ "status": "ok" }`

---

## Messages

### `GET /api/orders/:orderId/messages`

- Auth: required (order participant enforced)
- Query: `page`, `page_size`
- Response `200`: paginated `Message[]`

### `POST /api/orders/:orderId/messages`

- Auth: required
- Rate limit: per-user + per-IP
- Content type: `multipart/form-data`
- Fields:
	- `body` (required, max 10000 chars)
	- `attachment` (optional)
- Attachment rules:
	- Max 10MB
	- PII detection applies to message and extracted attachment text
	- Allowed types: text, CSV, images, PDF (other binaries rejected)
- Response `201`: `Message`

### `GET /api/messages/:messageId/attachment`

- Auth: required (message/order access enforced)
- Response `200`: binary blob with attachment MIME type

---

## Notifications

### `GET /api/notifications`

- Auth: required
- Query modes:
	- list mode: `page`, `page_size`, optional `unread=true`
	- count mode: `count=true` (returns unread count)
- Responses:
	- `200` list mode: paginated `Notification[]`
	- `200` count mode:

```json
{ "unread_count": 3 }
```

### `PATCH /api/notifications/:id/read`

- Auth: required (owner enforced)
- Response `200`: `{ "status": "ok" }`

### `POST /api/notifications/read-all`

- Auth: required
- Response `200`: `{ "status": "ok" }`

### `POST /api/notifications/:id/retry`

- Auth: required (owner enforced)
- Allowed states: `failed` or `permanently_failed`
- Response `200`: `{ "status": "ok" }`

### `GET /api/notifications/preferences`

- Auth: required
- Response `200`:

```json
{
	"user_id": "uuid",
	"preferences": { "order_confirmed": true },
	"subscription_mode": "all_events"
}
```

### `PUT /api/notifications/preferences`

- Auth: required
- Body: `UpdateNotificationPrefsRequest`
- `subscription_mode` values:
	- `all_events`
	- `status_only`
- Response `200`: `{ "status": "ok" }`

---

## Analytics (Administrator or Compliance Analyst)

### `GET /api/analytics/funnel`

- Query: `days` (default 7)
- Response `200`: `FunnelResponse`

### `GET /api/analytics/retention`

- Query: `days` (default 30)
- Response `200`: `RetentionCohort[]`

### `GET /api/analytics/content-performance`

- Query: `limit` (default 20)
- Response `200`: `ContentPerformance[]`

---

## A/B Tests

### `POST /api/ab-tests`

- Auth: administrator or compliance analyst
- Body: `CreateABTestRequest`
- Response `201`: `ABTest`

### `GET /api/ab-tests`

- Auth: administrator or compliance analyst
- Response `200`: `ABTest[]`

### `GET /api/ab-tests/:id`

- Auth: administrator or compliance analyst
- Response `200`:

```json
{
	"test": { "id": "...", "name": "..." },
	"results": []
}
```

### `PATCH /api/ab-tests/:id`

- Auth: administrator or compliance analyst
- Body: `UpdateABTestRequest`
- Response `200`: `ABTest`

### `POST /api/ab-tests/:id/complete`

- Auth: administrator or compliance analyst
- Response `200`: `{ "status": "ok" }`

### `POST /api/ab-tests/:id/rollback`

- Auth: administrator or compliance analyst
- Response `200`: `{ "status": "ok" }`

### `GET /api/ab-tests/assignments`

- Auth: any authenticated user
- Response `200`:

```json
[
	{ "test_name": "catalog_layout", "variant": "grid" }
]
```

### `GET /api/ab-tests/registry`

- Auth: any authenticated user
- Response `200`:

```json
{
	"catalog_layout": {
		"description": "Controls the collectible catalog grid layout (CatalogPage.tsx)",
		"variants": ["grid", "list"]
	}
}
```

---

## Admin

### IP Rules (Administrator Only)

#### `GET /api/admin/ip-rules`

- Response `200`: `IPRule[]`

#### `POST /api/admin/ip-rules`

- Body:

```json
{ "cidr": "192.168.1.0/24", "action": "allow|deny" }
```

- Response `201`: created `IPRule`

#### `DELETE /api/admin/ip-rules/:id`

- Response `200`: `{ "status": "ok" }`

### Metrics (Administrator Only)

#### `GET /api/admin/metrics`

- Response `200`: `MetricsResponse`

### Anomalies (Administrator or Compliance Analyst)

#### `GET /api/admin/anomalies`

- Query:
	- `acknowledged=true|false` (optional)
	- `page`, `page_size`
- Response `200`: paginated anomaly events

#### `PATCH /api/admin/anomalies/:id/acknowledge`

- Response `200`: `{ "status": "ok" }`

---

## Selected Domain Shapes Returned by API

### Order

```json
{
	"id": "uuid",
	"idempotency_key": "string",
	"buyer_id": "uuid",
	"collectible_id": "uuid",
	"seller_id": "uuid",
	"status": "pending|confirmed|processing|completed|cancelled",
	"price_snapshot_cents": 0,
	"cancellation_reason": "optional",
	"cancelled_by": "optional uuid",
	"fulfillment_tracking": {},
	"created_at": "timestamp",
	"updated_at": "timestamp"
}
```

### Message

```json
{
	"id": "uuid",
	"order_id": "uuid",
	"sender_id": "uuid",
	"body": "string",
	"attachment_id": "optional string",
	"attachment_size": 0,
	"attachment_mime": "optional MIME",
	"created_at": "timestamp"
}
```

### Notification

```json
{
	"id": "uuid",
	"user_id": "uuid",
	"template_id": "uuid",
	"template_slug": "optional",
	"params": {},
	"rendered_title": "string",
	"rendered_body": "string",
	"is_read": false,
	"status": "pending|delivered|failed|permanently_failed",
	"retry_count": 0,
	"max_retries": 3,
	"next_retry_at": "optional timestamp",
	"delivered_at": "optional timestamp",
	"created_at": "timestamp"
}
```

## Notes for Client Implementers

- Send credentials with every request (`withCredentials=true`) to include auth cookies.
- For non-GET requests, include `X-CSRF-Token` read from `csrf_token` cookie.
- Handle `401` by attempting `POST /api/auth/refresh` then retrying original request.
- Handle pre-bootstrap `503 ERR_SETUP_REQUIRED` by routing user to setup flow.
